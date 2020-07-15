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
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/tests/pkg/cspc/cspcspecbuilder"
	"github.com/openebs/cstor-operators/tests/pkg/cstorvolumeconfig/cvcspecbuilder"
	"github.com/openebs/cstor-operators/tests/pkg/k8sclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// openebsNamespace defines namespace where openebs is installed
	openebsNamespace = "openebs"
	// cspc will holds the CStorPoolCluster
	cspc *cstorapis.CStorPoolCluster
	// specBuilder for building the CSPC spec
	specBuilder *cspcspecbuilder.CSPCSpecBuilder
	// cvcSpecBuilder for building the CVC Spec
	cvcSpecBuilder *cvcspecbuilder.CVCSpecBuilder
)

/*
This test file covers following test cases :
1. CSI volume provisioing with multiple replicas on
   Stripe CSPC base pool.
*/

var _ = Describe("Volume Provisioning Tests", func() {
	CSIVolumeProvisioningTest()
	ProvisionVolumeWithReplicaCountMoreThanAvailablePools()
})

func CSIVolumeProvisioningTest() {
	testNS := "test-provisioning"
	pvcName := "pvc-vol"
	scName := "cstor-provision-sc"

	Describe("Intantiating Volume tests", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe", openebsNamespace, "stripe", 1)
			})
		})

		Context("Provision CStor-CSI volume", func() {
			It("Should provision volume and PVC should bound to PV", func() {
				ProvisionCSIVolume(pvcName, testNS, scName)
			})
		})

		Context("Verify Status of CStorVolume related resources", func() {
			Specify("no error should be returned and all the resources must be healthy", func() {
				VerifyCStorVolumeResourcesStatus(pvcName, testNS)
			})
		})

		Context("De-Provision CStor volume", func() {
			Specify("no error should be returned and all the CStor volume resources should be delete", func() {
				DeProvisionVolume(pvcName, testNS, scName)
			})
		})

		Context("Deleting PVC Namespace", func() {
			Specify("no error should occur during deletion of namespace", func() {
				err := cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(testNS, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				DeProvisionCSPC(cspc)
			})
		})
	})
}

// ProvisionVolumeWithReplicaCountMoreThanAvailablePools will create volume replica with more than available pools
// Negative test case: Expected to fail
func ProvisionVolumeWithReplicaCountMoreThanAvailablePools() {
	testNS := "test-provisioning"
	pvcName := "pvc-with-more-replicas"
	scName := "sc-with-more-replicas"
	Describe("Intantiating Volume provisioing test with replica count more than available pools", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-more-replicas", openebsNamespace, "stripe", 1)
			})
		})
		Context("Provision CStor-CSI volume", func() {
			It("Shouldn't provision volume and CVC should be in Pending state", func() {
				// Build parameters required for provisioning volume
				scParameters := map[string]string{
					"cas-type":         "cstor",
					"cstorPoolCluster": cspc.Name,
					"replicaCount":     strconv.Itoa(cstorsuite.infra.NodeCount + 1),
				}

				_ = createStorageClass(scName, scParameters)

				err := cstorsuite.client.CreateNamespace(testNS)
				Expect(err).To(BeNil())

				pvc := createPersistentVolumeClaim(pvcName, testNS, scName)

				err = cstorsuite.client.WaitForPersistentVolumeClaimPhase(
					pvc.Name, pvc.Namespace, corev1.ClaimBound, k8sclient.Poll, k8sclient.ClaimBindingTimeout)
				Expect(err).To(BeNil())

				// Fetch PVC to use in later
				pvc, err = cstorsuite.client.KubeClientSet.
					CoreV1().
					PersistentVolumeClaims(testNS).
					Get(pvc.Name, metav1.GetOptions{})
				Expect(err).To(BeNil())

				err = cstorsuite.client.WaitForCStorVolumeConfigPhase(
					pvc.Spec.VolumeName, openebsNamespace, cstorapis.CStorVolumeConfigPhaseBound, k8sclient.Poll, 30*time.Second)
				// Error should occur
				Expect(err).NotTo(BeNil())
			})
		})

		Context("De-Provision CStor volume", func() {
			Specify("no error should be returned and all the CStor volume resources should be delete", func() {
				DeProvisionVolume(pvcName, testNS, scName)
			})
		})

		Context("Deleting PVC Namespace", func() {
			Specify("no error should occur during deletion of namespace", func() {
				err := cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(testNS, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				DeProvisionCSPC(cspc)
			})
		})
	})
}
