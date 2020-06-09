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

package server

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
)

// Addresses encapsulates all of the addresses we bind to for various
// network services. Everything is optional and defaults to BindAddr.
type Addresses struct {
	HTTP string
}

// Config is the configuration for server.
type Config struct {
	// Region is the region this server is supposed to deal in.
	// Defaults to global.
	Region string

	// LogLevel is the level of the logs to putout
	LogLevel string

	// BindAddr is the address on which server services will
	// be bound. If not specified, this defaults to 127.0.0.1.
	BindAddr string

	// Port is used to control the network ports we bind to.
	Port *int

	// Addresses is used to override the network addresses we bind to.
	//
	// Use normalizedAddrs if you need the host+port to bind to.
	Addresses *Addresses

	// NormalizedAddr is set to the Address+Port by normalizeAddrs()
	NormalizedAddrs *Addresses

	// LeaveOnTerm is used to gracefully leave on the terminate signal
	LeaveOnTerm bool

	// HTTPAPIResponseHeaders allows users to configure the http agent to
	// set arbitrary headers on API responses
	HTTPAPIResponseHeaders map[string]string
}

// DefaultServerConfig is a the baseline configuration for server
func DefaultServerConfig() *Config {
	return &Config{
		LogLevel:    "INFO",
		Region:      "global",
		BindAddr:    "127.0.0.1",
		Addresses:   &Addresses{},
		LeaveOnTerm: true,
	}
}

// NormalizeAddrs normalizes Addresses to always be
// initialized and have sane defaults.
func (c *Config) NormalizeAddrs() error {
	if c.Port == nil {
		return errors.Errorf("Can not normaize address empty port provided")
	}
	c.Addresses.HTTP = normalizeBind(c.Addresses.HTTP, c.BindAddr)
	c.NormalizedAddrs = &Addresses{
		HTTP: net.JoinHostPort(c.Addresses.HTTP, strconv.Itoa(*c.Port)),
	}
	return nil
}

// normalizeBind returns a normalized bind address.
//
// If addr is set it is used, if not the default bind address is used.
func normalizeBind(addr, bind string) string {
	if addr == "" {
		return bind
	}
	return addr
}

// Listener can be used to get a new listener using a custom bind address.
// If the bind provided address is empty, the BindAddr is used instead.
func (c *Config) Listener(proto, addr string, port int) (net.Listener, error) {
	if addr == "" {
		addr = c.BindAddr
	}

	if 0 > port || port > 65535 {
		return nil, &net.OpError{
			Op:  "listen",
			Net: proto,
			Err: &net.AddrError{Err: "invalid port", Addr: fmt.Sprint(port)},
		}
	}
	return net.Listen(proto, net.JoinHostPort(addr, strconv.Itoa(port)))
}
