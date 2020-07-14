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

package provisioning

import (
	"flag"
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

Volume Provisioning test will take the replica count as an argument
if not specified then it will defaults to 3

RUN: ginkgo -v -- -kubeconfig=/var/run/kubernetes/admin.kubeconfig -nodecount=<storage_node_count> -replicacount=<replica_count>

*/
func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSPC Integration Tests")
}

var (
	// ReplicaCount is number of storage replicas required for volume
	ReplicaCount int
)

// parseFlags gets the flag values at run time
func parseFlags() {
	flag.IntVar(&ReplicaCount, "replicacount", 3, "number of storage replicas needs to perform for testing")
}

func init() {
	infra.ParseFlags()
	k8sclient.ParseFlags()
	// Test specific flags
	parseFlags()
}

type CStorTestSuite struct {
	client       *k8sclient.Client
	infra        *infra.Infrastructure
	ReplicaCount int
	CSPCCache    *cspccache.CSPCResourceCache
}

func NewCStorTestSuite() *CStorTestSuite {
	k8sClient, err := k8sclient.NewClient(k8sclient.KubeConfigPath)
	if err != nil {
		klog.Fatalf("failed to build CSPC test suite:%s", err.Error())
	}

	infra := infra.NewInfrastructure()
	cspcCache := cspccache.NewCSPCCache(k8sClient, infra)

	return &CStorTestSuite{
		client:       k8sClient,
		infra:        infra,
		CSPCCache:    cspcCache,
		ReplicaCount: ReplicaCount,
	}
}

var cstorsuite *CStorTestSuite

var _ = BeforeSuite(func() {
	cstorsuite = NewCStorTestSuite()
})
