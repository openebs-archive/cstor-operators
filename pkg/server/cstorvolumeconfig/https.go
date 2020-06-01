/*
Copyright 2020 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cstorvolumeconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/ugorji/go/codec"
	"k8s.io/klog"
)

const (
	// ErrInvalidPath is used if the HTTP path is not supported
	ErrInvalidPath = "Invalid path"

	// ErrInvalidMethod is used if the HTTP method is not supported
	ErrInvalidMethod = "Invalid http method"

	// ErrGetMethodRequired is used if the HTTP GET method is required"
	ErrGetMethodRequired = "GET method required"

	// ErrPutMethodRequired is used if the HTTP PUT/POST method is required"
	ErrPutMethodRequired = "PUT/POST method required"
)

var (
	// jsonHandle and jsonHandlePretty are the codec handles to JSON encode
	// structs. The pretty handle will add indents for easier human consumption.
	jsonHandle       = &codec.JsonHandle{}
	jsonHandlePretty = &codec.JsonHandle{Indent: 4}
)

// HTTPServer is used to wrap cvc server and expose it over an HTTP interface
type HTTPServer struct {
	cvcServer *CVCServer

	mux      *http.ServeMux
	listener net.Listener
	logger   *log.Logger
	addr     string
}

// HTTPCodedError is used to provide the HTTP error code
type HTTPCodedError interface {
	error
	Code() int
}

type codedError struct {
	s    string
	code int
}

func (e *codedError) Error() string {
	return e.s
}

func (e *codedError) Code() int {
	return e.code
}

// CodedErrorWrapf is used to provide HTTP error
// Code and corresponding error as well additional
// details in a format decided by the caller
func CodedErrorWrapf(code int, err error, msg string, args ...interface{}) HTTPCodedError {
	errMsg := fmt.Sprintf("error: {%v}, msg: {%s}", err, msg)
	finalMsg := fmt.Sprintf(errMsg, args...)
	return CodedError(code, finalMsg)
}

// CodedErrorWrap is used to provide HTTP error
// Code and corresponding error
func CodedErrorWrap(code int, err error) HTTPCodedError {
	errMsg := fmt.Sprintf("%v", err)
	return CodedError(code, errMsg)
}

// CodedErrorf is used to provide HTTP error
// Code and corresponding error details in
// a format decided by the caller
func CodedErrorf(code int, msg string, args ...interface{}) HTTPCodedError {
	errMsg := fmt.Sprintf("%v", msg)
	finalMsg := fmt.Sprintf(errMsg, args...)
	return CodedError(code, finalMsg)
}

// CodedError is used to provide HTTP error
// Code and corresponding error msg
func CodedError(c int, msg string) HTTPCodedError {
	return &codedError{msg, c}
}

// NewHTTPServer starts new HTTP server over CVC Server
func NewHTTPServer(cvcServer *CVCServer) (*HTTPServer, error) {
	if cvcServer.config == nil {
		return nil, errors.Errorf("failed to instantiate http server: provided empty config")
	}
	lnAddr, err := net.ResolveTCPAddr("tcp", cvcServer.config.NormalizedAddrs.HTTP)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate http server: %v", cvcServer.config)
	}

	// Start the TCP listener
	listner, err := cvcServer.config.Listener("tcp", lnAddr.IP.String(), lnAddr.Port)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate http server: %v", cvcServer.config)
	}

	// Create new mux server
	mux := http.NewServeMux()

	// Create the server
	srv := &HTTPServer{
		cvcServer: cvcServer,
		mux:       mux,
		listener:  listner,
		logger:    cvcServer.logger,
		addr:      listner.Addr().String(),
	}
	klog.Infof("starting CVC REST server")

	// Register REST endpoints to serve the request
	srv.registerHandlers()

	// Start the server
	go http.Serve(listner, mux)

	return srv, nil
}

// Shutdown is used to shutdown the HTTP server
func (s *HTTPServer) Shutdown() {
	if s != nil {
		s.logger.Printf("[DEBUG] http: Shutting down http server")
		s.listener.Close()
	}
	s.cvcServer.Shutdown()
}

// registerHandlers is used to attach handlers to the mux
func (s *HTTPServer) registerHandlers() {
	// TODO: Check with team is promethous metrics is required

	// Request w.r.t to backup is handled here
	s.mux.HandleFunc("/latest/backups/", s.wrap(s.backupV1alpha1SpecificRequest))

	// Request w.r.t to restore is handled here
	s.mux.HandleFunc("/latest/restore/", s.wrap(s.restoreV1alpha1SpecificRequest))
}

// wrap is a convenient method used to wrap the handler function &
// return this handler curried with common logic.
func (s *HTTPServer) wrap(
	handler func(resp http.ResponseWriter, req *http.Request) (interface{}, error)) func(resp http.ResponseWriter, req *http.Request) {
	var code int
	f := func(resp http.ResponseWriter, req *http.Request) {
		// some book keeping stuff
		setHeaders(resp, s.cvcServer.config.HTTPAPIResponseHeaders)
		reqURL := req.URL.String()
		start := time.Now()
		defer func() {
			klog.V(4).Infof("[DEBUG] http: Request %v (%v)", reqURL, time.Since(start))
		}()

		klog.V(4).Infof("[DEBUG] http: Request %v (%v)", reqURL, req.Method)

		// Invoke original handler
		obj, err := handler(resp, req)

		// Check for an error & set it as an http error
		// Below err block for re-usability
	HAS_ERR:
		if err != nil {
			s.logger.Printf("[ERR] http: Request %v %v\n%v", req.Method, reqURL, err)
			code = 500
			if http, ok := err.(HTTPCodedError); ok {
				code = http.Code()
			}
			resp.WriteHeader(code)
			resp.Write([]byte(err.Error()))
			return
		}

		prettyPrint := false
		if v, ok := req.URL.Query()["pretty"]; ok {
			if len(v) > 0 && (len(v[0]) == 0 || v[0] != "0") {
				prettyPrint = true
			}
		}

		// Transform the response structure to its JSON equivalent
		if obj != nil {
			var buf bytes.Buffer
			if prettyPrint {
				enc := codec.NewEncoder(&buf, jsonHandlePretty)
				err = enc.Encode(obj)
				if err == nil {
					buf.Write([]byte("\n"))
				}
			} else {
				enc := codec.NewEncoder(&buf, jsonHandle)
				err = enc.Encode(obj)
			}

			// err is handled for both pretty & plain
			if err != nil {
				goto HAS_ERR
			}
			// no error, set the response as json
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(buf.Bytes())
		}
	}
	return f
}

// setHeaders is used to set canonical response header fields
func setHeaders(resp http.ResponseWriter, headers map[string]string) {
	for field, value := range headers {
		resp.Header().Set(http.CanonicalHeaderKey(field), value)
	}
}

// Get the value of Content-Type that is set in http request header
func getContentType(req *http.Request) (string, error) {

	if req.Header == nil {
		return "", fmt.Errorf("Request does not have any header")
	}

	return req.Header.Get("Content-Type"), nil
}

// Decode the request body to appropriate structure based on content
// type
func decodeBody(req *http.Request, out interface{}) error {

	cType, err := getContentType(req)
	if err != nil {
		return err
	}

	if strings.Contains(cType, "yaml") {
		return decodeYamlBody(req, out)
	}

	// default is assumed to be json content
	return decodeJsonBody(req, out)
}

// decodeJsonBody is used to decode a JSON request body
func decodeJsonBody(req *http.Request, out interface{}) error {
	dec := json.NewDecoder(req.Body)
	return dec.Decode(&out)
}

// decodeYamlBody is used to decode a YAML request body
func decodeYamlBody(req *http.Request, out interface{}) error {
	// Get []bytes from io.Reader
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, &out)
}
