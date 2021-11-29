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

package algorithm

import (
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	cstorstoredversion "github.com/openebs/api/v3/pkg/client/clientset/versioned/typed/cstor/v1"
	openebsstoredversion "github.com/openebs/api/v3/pkg/client/clientset/versioned/typed/openebs.io/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

// Config embeds CSPC object and namespace where openebs is installed.
type Config struct {
	// CSPC is the CStorPoolCluster object.
	CSPC *cstor.CStorPoolCluster

	// Namespace is the namespace where openebs is installed.
	Namespace string

	// VisitedNodes is a map which contains the node names which has already been
	// processed for pool provisioning
	VisitedNodes map[string]bool

	// clientset is a openebs custom resource package generated for custom API group.
	clientset clientset.Interface

	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// GetSpec returns the CSPI spec. Useful in unit testing.
	GetSpec func() (*cstor.CStorPoolInstance, error)

	// Select selects a node for CSPI. Useful in unit testing.
	Select func() (*cstor.PoolSpec, string, error)
}

// Builder embeds the Config object.
type Builder struct {
	ConfigObj *Config
	errs      []error
}

// NewBuilder returns an empty instance of Builder object
// ToDo: Add openebs and kube client set
func NewBuilder() *Builder {
	return &Builder{
		ConfigObj: &Config{
			CSPC:         &cstor.CStorPoolCluster{},
			Namespace:    "",
			VisitedNodes: make(map[string]bool),
		},
	}
}

// WithNameSpace sets the Namespace field of config object with provided value.
func (b *Builder) WithNameSpace(ns string) *Builder {
	if len(ns) == 0 {
		b.errs = append(b.errs, errors.New("failed to build algorithm config object: missing namespace"))
		return b
	}
	b.ConfigObj.Namespace = ns
	return b
}

// WithOpenEBSClient sets the clientset field of config object with provided value.
func (b *Builder) WithOpenEBSClient(oc clientset.Interface) *Builder {
	b.ConfigObj.clientset = oc
	return b
}

// WithKubeClient sets the kubeclientset field of config object with provided value.
func (b *Builder) WithKubeClient(kc kubernetes.Interface) *Builder {
	b.ConfigObj.kubeclientset = kc
	return b
}

// WithCSPC sets the CSPC field of the config object with the provided value.
func (b *Builder) WithCSPC(cspc *cstor.CStorPoolCluster) *Builder {
	if cspc == nil {
		b.errs = append(b.errs, errors.New("failed to build algorithm config object: nil cspc object"))
		return b
	}
	b.ConfigObj.CSPC = cspc
	return b
}

// Build returns the Config  instance
func (b *Builder) Build() (*Config, error) {
	if len(b.errs) > 0 {
		return nil, errors.Errorf("%+v", b.errs)
	}
	return b.ConfigObj, nil
}

func (ac *Config) GetStoredOpenEBSVersionClient() openebsstoredversion.OpenebsV1alpha1Interface {
	return ac.clientset.OpenebsV1alpha1()
}

func (ac *Config) GetStoredCStorVersionClient() cstorstoredversion.CstorV1Interface {
	return ac.clientset.CstorV1()
}
