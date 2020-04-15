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

package sanity_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/api/pkg/apis/types"
	"github.com/openebs/cstor-operators/tests/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSPC Sanity Tests")
}

func init() {
	client.ParseFlags()
}

var clientSet *client.Client

var _ = BeforeSuite(func() {
	var err error
	clientSet, err = client.NewClient(client.KubeConfigPath)
	Expect(err).To(BeNil())
	NodeList, err := clientSet.KubeClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	Expect(err).To(BeNil())
	Expect(len(NodeList.Items)).Should(BeNumerically(">=", 1))
})

var _ = Describe("CSPC Stripe On One Node", func() {

	var cspc *cstor.CStorPoolCluster
	Describe("Provisioning and cleanup of CSPC", func() {

		Context("Creating a cspc", func() {

			Specify("no error should be returned", func() {
				cspc = getCSPCSpec()
				_, err := clientSet.OpenEBSClientSet.CstorV1().CStorPoolClusters(cspc.Namespace).Create(cspc)
				Expect(err).To(BeNil())
			})

			Specify("desired count should be 1 on cspc", func() {
				gotCount := clientSet.GetDesiredInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, 1)
				Expect(gotCount).To(BeNumerically("==", 1))
			})

		})

		Context("All the cspi(s) of the cspc", func() {
			It("Should be healthy", func() {
				gotHealthyCSPiCount := clientSet.GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, 1)
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", 1))
			})
		})

		Context("Staus of the cspc i.e. provisionedInstances and healthyInstances ", func() {
			It("Should be updated", func() {
				gotProvisionedCount := clientSet.GetProvisionedInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, 1)
				Expect(gotProvisionedCount).To(BeNumerically("==", 1))

				gotHealthyCount := clientSet.GetHealthyInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, 1)
				Expect(gotHealthyCount).To(BeNumerically("==", 1))
			})
		})

		Context("Deleting the cspc", func() {

			It("No error should be returned", func() {
				err := clientSet.OpenEBSClientSet.CstorV1().CStorPoolClusters(cspc.Namespace).Delete(cspc.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})

			It("No corresponding cspi(s) should be present", func() {
				gotCSPICount := clientSet.GetCSPICountEventually(cspc.Name, cspc.Namespace, 0)
				Expect(gotCSPICount).To(BeNumerically("==", 0))
			})

			It("No corresponding pool-manger deployments should be present", func() {
				gotPoolMangerCount := clientSet.GetPoolManagerCountEventually(cspc.Name, cspc.Namespace, 0)
				Expect(gotPoolMangerCount).To(BeNumerically("==", 0))
			})

			It("the bdc(s) created by cstor-operator should get deleted", func() {
				gotCount := clientSet.GetBDCCountEventually(cspc.Name, cspc.Namespace, 0)
				Expect(gotCount).To(BeNumerically("==", 0))
			})
		})
	})

})

func getNodeSelector() map[string]string {
	nodes, err := clientSet.KubeClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	Expect(err).To(BeNil())
	Expect(len(nodes.Items)).To(BeNumerically(">=", 1))
	// pick a node
	node := nodes.Items[0]
	newLabels := make(map[string]string)
	newLabels[types.HostNameLabelKey] = node.Labels[types.HostNameLabelKey]
	return newLabels
}

func getBDName(nodeSelector map[string]string) string {
	bdList, err := clientSet.OpenEBSClientSet.OpenebsV1alpha1().BlockDevices("openebs").
		List(metav1.ListOptions{LabelSelector: types.HostNameLabelKey + "=" + nodeSelector[types.HostNameLabelKey]})
	Expect(err).To(BeNil())
	Expect(len(bdList.Items)).To(BeNumerically(">=", 1))
	bd := bdList.Items[0]
	return bd.Name
}

func getCSPCSpec() *cstor.CStorPoolCluster {
	nodeSelector := getNodeSelector
	Expect(nodeSelector()).ToNot(BeNil())
	cspc := cstor.NewCStorPoolCluster().
		WithName("cspc-foo").
		WithNamespace("openebs").
		WithPoolSpecs(
			*cstor.NewPoolSpec().
				WithNodeSelector(getNodeSelector()).
				WithDataRaidGroups(
					*cstor.NewRaidGroup().
						WithCStorPoolInstanceBlockDevices(
							*cstor.NewCStorPoolInstanceBlockDevice().
								WithName(getBDName(nodeSelector())),
						),
				).
				WithPoolConfig(*cstor.NewPoolConfig().WithDataRaidGroupType("stripe")),
		)
	return cspc
}
