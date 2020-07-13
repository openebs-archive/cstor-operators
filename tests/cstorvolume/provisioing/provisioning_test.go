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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/tests/pkg/cspc/cspcspecbuilder"
	"github.com/openebs/cstor-operators/tests/pkg/cstorvolumeconfig/cvcspecbuilder"
	"github.com/openebs/cstor-operators/tests/pkg/k8sclient"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
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

func ProvisionCSIVolume(pvcName, pvcNamespace, scName string) {
	var (
		// PVC will contains the PersistentVolumeClaim object created for test
		pvc *corev1.PersistentVolumeClaim
		// SC will contains the StorageClass object created for test
		sc *storagev1.StorageClass
	)

	parameters := map[string]string{
		"cas-type":         "cstor",
		"cstorPoolCluster": cspc.Name,
		"replicaCount":     strconv.Itoa(cstorsuite.ReplicaCount),
	}
	sc = createStorageClass(scName, parameters)

	err := cstorsuite.client.CreateNamespace(pvcNamespace)
	Expect(err).To(BeNil())

	pvc = createPersistentVolumeClaim(pvcName, pvcNamespace, sc.Name)

	err = cstorsuite.client.WaitForPersistentVolumeClaimPhase(
		pvc.Name, pvc.Namespace, corev1.ClaimBound, k8sclient.Poll, k8sclient.ClaimBindingTimeout)
	Expect(err).To(BeNil())

	//TODO: Uncomment below code
	// cvcSpecBuilder.SetCVCSpec(cvc)
}

func VerifyCStorVolumeResourcesStatus(pvcName, pvcNamespace string) {
	// Read the PVC after bind so that it will contain pv name
	pvc, err := cstorsuite.client.KubeClientSet.
		CoreV1().
		PersistentVolumeClaims(pvcNamespace).
		Get(pvcName, metav1.GetOptions{})
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForCStorVolumeConfigPhase(
		pvc.Spec.VolumeName, openebsNamespace, cstorapis.CStorVolumeConfigPhaseBound, k8sclient.Poll, k8sclient.CVCPhaseTimeout)
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForVolumeManagerCountEventually(
		pvc.Spec.VolumeName, openebsNamespace, 1, k8sclient.Poll, k8sclient.CVCPhaseTimeout)
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForCStorVolumePhase(
		pvc.Spec.VolumeName, openebsNamespace, cstorapis.CStorVolumePhase("Healthy"), k8sclient.Poll, k8sclient.CVPhaseTimeout)
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForCVRCountEventually(
		pvc.Spec.VolumeName, openebsNamespace, cstorsuite.ReplicaCount,
		k8sclient.Poll, k8sclient.CVRPhaseTimeout, cstorapis.IsCVRHealthy)
	Expect(err).To(BeNil())
}

func DeProvisionVolume(pvcName, pvcNamespace, scName string) {
	// Read the PVC after before deleting the PVC
	pvc, err := cstorsuite.client.KubeClientSet.
		CoreV1().
		PersistentVolumeClaims(pvcNamespace).
		Get(pvcName, metav1.GetOptions{})
	Expect(err).To(BeNil())

	err = cstorsuite.client.KubeClientSet.
		CoreV1().
		PersistentVolumeClaims(pvc.Namespace).
		Delete(pvc.Name, &metav1.DeleteOptions{})
	Expect(err).To(BeNil())

	err = cstorsuite.client.KubeClientSet.
		StorageV1().
		StorageClasses().
		Delete(scName, &metav1.DeleteOptions{})
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForCStorVolumeDeleted(
		pvc.Spec.VolumeName, openebsNamespace, k8sclient.Poll, k8sclient.CVDeletingTimeout)
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForVolumeManagerCountEventually(
		pvc.Spec.VolumeName, openebsNamespace, 0, k8sclient.Poll, k8sclient.CVCPhaseTimeout)
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForCVRCountEventually(
		pvc.Spec.VolumeName, openebsNamespace, 0,
		k8sclient.Poll, k8sclient.CVRPhaseTimeout, cstorapis.IsCVRHealthy)
	Expect(err).To(BeNil())

	err = cstorsuite.client.WaitForCStorVolumeConfigDeleted(
		pvc.Spec.VolumeName, openebsNamespace, k8sclient.Poll, k8sclient.CVCDeletingTimeout)
	Expect(err).To(BeNil())
}
