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

package provisioning_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openebs/cstor-operators/tests/pkg/cache/cspccache"
	"github.com/openebs/cstor-operators/tests/pkg/infra"
	"github.com/openebs/cstor-operators/tests/pkg/k8sclient"
	"k8s.io/klog"
)

/*
NOTE: The test cases adjusts depending on the number of nodes
in the Kubernetes cluster.
Meaning, if only 1 node is present then the test result expectations(output)
are in accordance with what it should be with 1 node.

if only 3 node is present then the test result expectations(output)
are in accordance with what it should be with 3 node.

Before starting the test suite, it should be specified whether it is
a 3 node or 1 node test.

Test suite only supports either a 1 node or 3 node test.

*/
func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSPC Integration Tests")
}

func init() {
	infra.ParseFlags()
	k8sclient.ParseFlags()
}

type CSPCTestSuite struct {
	client    *k8sclient.Client
	infra     *infra.Infrastructure
	CSPCCache *cspccache.CSPCResourceCache
}

func NewCSPCTestSuite() *CSPCTestSuite {
	k8sClient, err := k8sclient.NewClient(k8sclient.KubeConfigPath)
	if err != nil {
		klog.Fatalf("failed to build CSPC test suite:%s", err.Error())
	}

	infra := infra.NewInfrastructure()
	cspcCache := cspccache.NewCSPCCache(k8sClient, infra)

	return &CSPCTestSuite{
		client:    k8sClient,
		infra:     infra,
		CSPCCache: cspcCache,
	}
}

var cspcsuite *CSPCTestSuite

var _ = BeforeSuite(func() {
	cspcsuite = NewCSPCTestSuite()
})
