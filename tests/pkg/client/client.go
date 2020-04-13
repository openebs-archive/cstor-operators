/*
Copyright 2020 The OpenEBS Authors.

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

package client

import (
	"flag"
	openebsclientset "github.com/openebs/api/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

type Client struct {
	// kubeclientset is a standard kubernetes clientset
	KubeClientSet kubernetes.Interface

	// clientset is a openebs custom resource package generated for custom API group.
	OpenEBSClientSet openebsclientset.Interface
}

var (
	// KubeConfigPath is the path to
	// the kubeconfig provided at runtime
	KubeConfigPath string
)

// ParseFlags gets the flag values at run time
func ParseFlags() {
	flag.StringVar(&KubeConfigPath, "kubeconfig", "", "path to kubeconfig to invoke kubernetes API calls")
}

func NewClient(path string) (*Client, error) {
	klog.InitFlags(nil)
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return nil, errors.Wrap(err, "failed to set logtostderr flag")
	}

	cfg, err := getClusterConfig(path)
	if err != nil {
		return nil, errors.Wrap(err, "error building kubeconfig")
	}
	// Building Kubernetes Clientset
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error building kubernetes clientset")
	}

	// Building OpenEBS Clientset
	openebsClient, err := openebsclientset.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error building openebs clientset")
	}

	client := &Client{
		KubeClientSet:    kubeClient,
		OpenEBSClientSet: openebsClient,
	}

	return client, nil

}

// GetClusterConfig return the config for k8s.
func getClusterConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	klog.V(2).Info("Kubeconfig flag is empty")
	return rest.InClusterConfig()
}
