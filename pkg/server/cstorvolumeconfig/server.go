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
	"io"
	"log"
	"sync"

	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	server "github.com/openebs/cstor-operators/pkg/server"
	"github.com/openebs/cstor-operators/pkg/snapshot"
	"k8s.io/client-go/kubernetes"
)

// CVCServer contains the information to start the
// CVCServer which is helpful to serve the request
type CVCServer struct {
	config       *server.Config
	logger       *log.Logger
	logOutput    io.Writer
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// clientset is a openebs custom resource package generated for custom API group.
	clientset clientset.Interface

	// snapshotter is used to perform snapshot operations on Volumes
	snapshotter snapshot.Snapshotter
}

// NewCVCServer is used to create a new CVC server
// with the given configuration
func NewCVCServer(config *server.Config, logOutput io.Writer) *CVCServer {
	cs := &CVCServer{
		config:     config,
		logger:     log.New(logOutput, "", log.LstdFlags|log.Lmicroseconds),
		logOutput:  logOutput,
		shutdownCh: make(chan struct{}),
	}
	return cs
}

// WithKubernetesClientSet sets the kubeclientset whith provided argument
func (cs *CVCServer) WithKubernetesClientSet(kubeclientset kubernetes.Interface) *CVCServer {
	cs.kubeclientset = kubeclientset
	return cs
}

// WithOpenebsClientSet sets the kubeclientset whith provided argument
func (cs *CVCServer) WithOpenebsClientSet(openebsClientSet clientset.Interface) *CVCServer {
	cs.clientset = openebsClientSet
	return cs
}

// WithSnapshotter sets the snapshotter with provided argument
func (cs *CVCServer) WithSnapshotter(snapshotter snapshot.Snapshotter) *CVCServer {
	cs.snapshotter = snapshotter
	return cs
}

// Shutdown is used to terminate CVCServer
func (cs *CVCServer) Shutdown() {

	cs.shutdownLock.Lock()
	defer cs.shutdownLock.Unlock()

	cs.logger.Println("[INFO] CVC server: requesting shutdown")

	if cs.shutdown {
		return
	}

	cs.logger.Println("[INFO] CVC server: shutdown complete")
	cs.shutdown = true

	close(cs.shutdownCh)

	return
}
