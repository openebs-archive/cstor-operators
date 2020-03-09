/*
Copyright 2019 The OpenEBS Authors.

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

package v1alpha2

import (
	openebsclientset "github.com/openebs/api/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type OperationsConfig struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// openebsclientset is a openebs custom resource package generated for custom API group.
	openebsclientset openebsclientset.Interface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewOperationsConfig() *OperationsConfig {
	return &OperationsConfig{}
}

// WithKubeClientSet fills the kubernetes client to perform operation on kubernetes resorces
func (oc *OperationsConfig) WithKubeClientSet(ks kubernetes.Interface) *OperationsConfig {
	oc.kubeclientset = ks
	return oc
}

// withOpenEBSClient fills openebs client to controller object.
func (oc *OperationsConfig) WithOpenEBSClient(ocs openebsclientset.Interface) *OperationsConfig {
	oc.openebsclientset = ocs
	return oc
}

// withRecorder fills recorder to generate events on CSPI
func (oc *OperationsConfig) WithRecorder(recorder record.EventRecorder) *OperationsConfig {
	oc.recorder = recorder
	return oc
}
