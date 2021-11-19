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
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstorapis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/tests/pkg/cspc/cspcspecbuilder"
	"github.com/openebs/cstor-operators/tests/pkg/cstorvolumeconfig/cvcspecbuilder"
	"github.com/openebs/cstor-operators/tests/pkg/k8sclient"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	// openebsNamespace defines namespace where openebs is installed
	openebsNamespace = "openebs"
	// cspcSpecBuilder for building the CSPC spec
	cspcSpecBuilder *cspcspecbuilder.CSPCSpecBuilder
	// cvcSpecBuilder for building the CVC Spec
	cvcSpecBuilder *cvcspecbuilder.CVCSpecBuilder
	// cspc will holds the CStorPoolCluster
	cspc *cstorapis.CStorPoolCluster
)

/*
This test file covers following test cases :
1. CSI volume provisioing with multiple replicas on
   Stripe CSPC base pool.
*/

var _ = Describe("Volume Provisioning Tests", func() {
	CSIVolumeProvisioningTest()
	ProvisionVolumeWithReplicaCountMoreThanAvailablePools()
	CSIVolumeProvisioningTestWithResourceLimits()
	CSIVolumeProvisioningTestWithTolerations()
	ProvisionVolumeWithPriorityClass()
	ProvisionVolumeAndUpdateTunables()
	NegativeScaleupAndScaleDownCStorVolume()
	ScaleupAndScaleDownCStorVolume()
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
				Expect(cspc).NotTo(BeNil(), "cstor-stripe CSPC is not created successfully")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")
				ProvisionCSIVolume(pvcName, testNS, scName, cstorsuite.ReplicaCount)
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.ReplicaCount)
			})
		})

		Context("Verify Status of CStorVolume related resources", func() {
			Specify("no error should be returned and all the resources must be healthy", func() {
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.ReplicaCount)
				Expect(cvcSpecBuilder.CVC).NotTo(BeNil(), "cvc in spec builder is not initilized")
			})
		})

		Context("De-Provision CStor volume", func() {
			Specify("no error should be returned and all the CStor volume resources should be delete", func() {
				DeProvisionVolume(pvcName, testNS, scName)
			})
		})

		Context("Deleting PVC Namespace", func() {
			Specify("no error should occur during deletion of namespace", func() {
				Expect(cspc).NotTo(BeNil(), "cstor-stripe CSPC is not created successfully")
				err := cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				// if CSPC creation is failed no need to delete CSPC
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})
	})
}

// CSIVolumeProvisioningTestWithResourceLimits will create volume and perform following steps
// 1. Add resource limits to CVC and verify whether resource limits are propogated to the target pod
// 2. Remove resource limits to CVC and verify whether resource limits are removed from target pod
func CSIVolumeProvisioningTestWithResourceLimits() {
	testNS := "test-provisioning-resource-limits"
	pvcName := "pvc-vol-resource-limits"
	scName := "cstor-provision-sc-resource-limits"

	Describe("Intantiating dynamic Volume resource limits tests", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-for-resource-limits", openebsNamespace, "stripe", 1)
			})
		})

		Context("Provision CStor-CSI volume and add and remove resource limits dynamically", func() {
			It("Should propogated to volume manager pod", func() {

				Expect(cspc).NotTo(BeNil(), "cstor-stripe-with-resource-limits CSPC is not created successfully")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")

				// resourcelimits for main container
				resourceLimits := &corev1.ResourceRequirements{
					Requests: corev1.ResourceList(
						map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceMemory: resource.MustParse("500Mi"),
							corev1.ResourceCPU:    resource.MustParse("0"),
						},
					),
					Limits: corev1.ResourceList(
						map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceMemory: resource.MustParse("1Gi"),
							corev1.ResourceCPU:    resource.MustParse("0"),
						},
					),
				}

				// resourcelimits for side car container
				auxResourceLimits := &corev1.ResourceRequirements{
					Requests: corev1.ResourceList(
						map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceMemory: resource.MustParse("500Mi"),
							corev1.ResourceCPU:    resource.MustParse("0"),
						},
					),
					Limits: corev1.ResourceList(
						map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceMemory: resource.MustParse("700Mi"),
							corev1.ResourceCPU:    resource.MustParse("0"),
						},
					),
				}

				// Provision cStor CSI volume
				ProvisionCSIVolume(pvcName, testNS, scName, cstorsuite.ReplicaCount)
				// Verify whether all the cStor volumes created
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.ReplicaCount)

				// add resource limits on CVC.Spec.Policy
				cvcSpecBuilder.SetResourceLimits(resourceLimits, auxResourceLimits)
				// updatedCVC, err := cstorsuite.client.PatchCVCResourceLimits(cvcSpecBuilder.CVC, resourceLimits, auxResourceLimits)
				updatedCVC, err := cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc with resource limits")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether resource limits are propogated to volume manager pod
				klog.Infof("Waiting for volumemanager to have both resource limits")
				err = cstorsuite.client.WaitForVolumeManagerResourceLimits(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					*resourceLimits, *auxResourceLimits, k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified limits")

				// remove auxresource limits alone
				cvcSpecBuilder.SetResourceLimits(resourceLimits, nil)
				//	updatedCVC, err = cstorsuite.client.PatchCVCResourceLimits(cvcSpecBuilder.CVC, resourceLimits, nil)
				updatedCVC, err = cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc without auxresource limits")
				klog.Infof("Resource limits %+v - %+v", updatedCVC.Spec.Policy.Target.Resources, updatedCVC.Spec.Policy.Target.AuxResources)
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether resource limits are propogated to volume manager pod
				klog.Infof("Waiting for volumemanager to have resource limits alone")
				err = cstorsuite.client.WaitForVolumeManagerResourceLimits(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					*resourceLimits, corev1.ResourceRequirements{}, k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified limits")

				// remove resource limits on main container
				cvcSpecBuilder.SetResourceLimits(nil, nil)
				updatedCVC, err = cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				//updatedCVC, err = cstorsuite.client.PatchCVCResourceLimits(cvcSpecBuilder.CVC, nil, nil)
				Expect(err).To(BeNil(), "failed to patch cvc without auxresource limits")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether resource limits are propogated to volume manager pod
				klog.Infof("Waiting for volumemanager not to have any resource limits")
				err = cstorsuite.client.WaitForVolumeManagerResourceLimits(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					corev1.ResourceRequirements{}, corev1.ResourceRequirements{},
					k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified limits")

				DeProvisionVolume(pvcName, testNS, scName)

				err = cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				// if CSPC creation is failed no need to delete CSPC
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})
	})

}

// CSIVolumeProvisioningTestWithTolerations will create volume and perform following steps
// 1. Add tolerations to CVC and verify whether tolerations are propogated to the target deployment
// 2. Remove tolerations from CVC and verify whether tolerations are removed from target deployment
// NOTE: Test will verify toleration on deployment because adding node will cause some problem
func CSIVolumeProvisioningTestWithTolerations() {
	testNS := "test-provisioning-tolerations"
	pvcName := "pvc-vol-toleratations"
	scName := "cstor-provision-sc-tolerations"

	Describe("Intantiating dynamic Volume provisioning and add tolerations dynamically", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-for-tolerations", openebsNamespace, "stripe", 1)
			})
		})

		Context("Provision CStor-CSI volume and add and remove tolertions dynamically", func() {
			It("Should propogated to volume manager pod", func() {

				Expect(cspc).NotTo(BeNil(), "cstor-stripe-for-toleration CSPC is not created successfully")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")

				tolerations := []corev1.Toleration{
					{
						Key:      "test-tolerations",
						Operator: corev1.TolerationOpEqual,
						Value:    "value",
						Effect:   "NoSchedule",
					},
				}

				// Provision cStor CSI volume
				ProvisionCSIVolume(pvcName, testNS, scName, cstorsuite.ReplicaCount)
				// Verify whether all the cStor volumes created
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.ReplicaCount)

				// add tolerations on CVC.Spec.Policy
				cvcSpecBuilder.SetTolerations(tolerations)
				updatedCVC, err := cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc with tolerations")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether tolerations are propogated to volume manager deployment
				klog.Infof("Waiting for cstor volumemanager deployment to have tolerations")
				err = cstorsuite.client.WaitForVolumeManagerTolerations(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					tolerations, k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified limits")

				// remove tolerations on CVC.Spec.Policy
				cvcSpecBuilder.SetTolerations([]corev1.Toleration{})
				updatedCVC, err = cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc with tolerations")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether tolerations are propogated to volume manager deployment
				klog.Infof("Waiting for cstor volumemanager deployment to have tolerations")
				err = cstorsuite.client.WaitForVolumeManagerTolerations(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					[]corev1.Toleration{}, k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified limits")
				DeProvisionVolume(pvcName, testNS, scName)

				err = cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				// if CSPC creation is failed no need to delete CSPC
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})
	})
}

func ProvisionVolumeWithPriorityClass() {
	testNS := "test-provisioning-priority-class"
	pvcName := "pvc-vol-priority-class"
	scName := "cstor-provision-sc-priority"

	Describe("Provisioning cStor Volume and dynamically adding priority class tests", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-for-priority", openebsNamespace, "stripe", 1)
			})
		})

		Context("Provision CStor-CSI volume and add and remove priority class dynamically", func() {
			It("Should propogated to volume manager pod", func() {

				Expect(cspc).NotTo(BeNil(), "cstor-stripe-for-priority CSPC is not created successfully")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")

				priorityClass := &schedulingv1.PriorityClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "priority-class",
					},
					Value:         100,
					GlobalDefault: false,
				}

				// Create Priority class
				priorityClassObj, err := cstorsuite.client.KubeClientSet.SchedulingV1().PriorityClasses().Create(context.TODO(), priorityClass, metav1.CreateOptions{})
				Expect(err).To(BeNil(), "failed to create %s priority class", priorityClass.Name)

				// Provision cStor CSI volume
				ProvisionCSIVolume(pvcName, testNS, scName, cstorsuite.ReplicaCount)
				// Verify whether all the cStor volumes created
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.ReplicaCount)

				// add priority class to CVC.Spec.Policy
				cvcSpecBuilder.SetPriorityClass(priorityClass.Name)
				updatedCVC, err := cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc with priority class")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether priority class is propogated to volume manager
				klog.Infof("Waiting for cstor volumemanager deployment to have priority class")
				err = cstorsuite.client.WaitForVolumeManagerPriorityClass(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					priorityClassObj.Name, k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified priority class")

				// remove priority on CVC.Spec.Policy
				cvcSpecBuilder.SetPriorityClass("")
				updatedCVC, err = cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to remove priority class from CVC")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether priority class is propogated to volume manager deployment
				klog.Infof("Waiting for cstor volumemanager to have priority class")
				err = cstorsuite.client.WaitForVolumeManagerPriorityClass(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					"", k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified priority class")

				DeProvisionVolume(pvcName, testNS, scName)

				// Delete Priority class
				err = cstorsuite.client.KubeClientSet.SchedulingV1().PriorityClasses().Delete(context.TODO(), priorityClass.Name, metav1.DeleteOptions{})
				Expect(err).To(BeNil(), "failed to delete %s priority class", priorityClass.Name)

				err = cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})
		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})
	})
}

func ProvisionVolumeAndUpdateTunables() {
	testNS := "test-provisioning-tunables"
	pvcName := "pvc-vol-tunables"
	scName := "cstor-provision-sc-tunables"

	Describe("Provisioning cStor Volume and dynamically update tunables", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-tunables", openebsNamespace, "stripe", 1)
			})
		})

		Context("Provision CStor-CSI volume and update tunables dynamically", func() {
			It("Should propogated to volume manager pod", func() {

				Expect(cspc).NotTo(BeNil(), "cstor-stripe-tunables CSPC is not created successfully")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")

				// Provision cStor CSI volume
				ProvisionCSIVolume(pvcName, testNS, scName, cstorsuite.ReplicaCount)
				// Verify whether all the cStor volumes created
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.ReplicaCount)

				// add tunables to CVC.Spec.Policy
				cvcSpecBuilder.SetLuWorkers(8)
				cvcSpecBuilder.SetQueueDepth("64")
				updatedCVC, err := cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc with tunables")
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				// Verify whether tunables are propogated to volume manager
				klog.Infof("Waiting for cstor volumemanager to have tunables")
				err = cstorsuite.client.WaitForVolumeManagerTunables(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					cvcSpecBuilder.CVC.Spec.Policy.Target.IOWorkers, cvcSpecBuilder.CVC.Spec.Policy.Target.QueueDepth,
					k8sclient.VolumeMangerConfigTimeout, 5*time.Second)
				Expect(err).To(BeNil(), "volume manger should have specified tunables")

				// Add node selector dynamically
				nodeLabels := map[string]string{}
				for _, labels := range cspcSpecBuilder.CSPCCache.NodeLabels {
					nodeLabels = labels
					break
				}
				cvcSpecBuilder.SetNodeSelector(nodeLabels)
				updatedCVC, err = cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				Expect(err).To(BeNil(), "failed to patch cvc with nodeSelector details")
				klog.Infof("Waiting for cstor volumemanager to have nodeSelector changes")
				err = cstorsuite.client.WaitForVolumeManagerNodeSelector(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace,
					nodeLabels, k8sclient.VolumeMangerConfigTimeout, 5*time.Second)

				DeProvisionVolume(pvcName, testNS, scName)

				err = cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})
		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})
	})
}

func ScaleupAndScaleDownCStorVolume() {
	testNS := "test-cstorvolume-scaling"
	pvcName := "pvc-vol-scaling"
	scName := "cstor-provision-sc-scaling"
	// Since we are performing scaleup and scaledown of CStorVolume
	// so good to hardcode the value
	replicaCount := 1

	Describe("Provisioning cStor Volume and perform scaleup and scaledown of CStorVolume", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-scaling", openebsNamespace, "stripe", 1)
			})
		})
		Context("Provision CStor-CSI volume and scaleup and scaledown CStorVolume", func() {
			It("Should able to perform scaleup and scaledown", func() {

				Expect(cspc).NotTo(BeNil(), "Specified CStor pools are not in healthy state")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")

				if len(cspc.Spec.Pools) < 3 {
					klog.Infof("Not enough pools are availabl to perform scaleup operation")
					return
				}
				// Provision cStor CSI volume
				ProvisionCSIVolume(pvcName, testNS, scName, replicaCount)
				// Verify whether all the cStor volumes created
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, replicaCount)

				// Since scaled up the replicas increasing the count to 1
				scaleupCStorVolume(2)
				replicaCount += 2
				verifyScaledCStorVolume(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace)

				// Scaledown to 2 replicas
				klog.Infof("Scaling down the CStorVolume from %s", cvcSpecBuilder.CVC.Status.PoolInfo[1])
				cvcSpecBuilder.RemovePoolsFromCVCSpec([]string{cvcSpecBuilder.CVC.Status.PoolInfo[1]})
				updatedCVC, err := cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to scale down the CVC")
				replicaCount--
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				verifyScaledCStorVolume(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace)

				// Scaledown to 1 replica
				klog.Infof("Scaling down the CStorVolume from %s", cvcSpecBuilder.CVC.Status.PoolInfo[1])
				cvcSpecBuilder.RemovePoolsFromCVCSpec([]string{cvcSpecBuilder.CVC.Status.PoolInfo[1]})
				updatedCVC, err = cstorsuite.client.PatchCVCSpec(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				Expect(err).To(BeNil(), "failed to scale down the CVC")
				replicaCount--
				cvcSpecBuilder.SetCVCSpec(updatedCVC)
				verifyScaledCStorVolume(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace)

				//Scaling up when already scaleup is in progress
				scaleupCStorVolume(1)
				replicaCount++
				scaleupCStorVolume(1)
				replicaCount++
				verifyScaledCStorVolume(cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace)

				DeProvisionVolume(pvcName, testNS, scName)

				err = cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())

			})
		})
		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})

	})
}

func NegativeScaleupAndScaleDownCStorVolume() {
	testNS := "test-cstorvolume-negative-scaling"
	pvcName := "pvc-vol-negative-scaling"
	scName := "cstor-provision-sc-negative-scaling"

	Describe("Provisioning cStor Volume and perform negative test cases on scaleup and scaledown of CStorVolume", func() {
		Context("Provision pools using CSPC", func() {
			It("should provision pools and pools should be marked as Healthy", func() {
				ProvisionCSPC("cstor-stripe-negative-scaling", openebsNamespace, "stripe", 1)
			})
		})
		Context("Provision CStor-CSI volume and negative test cases on scaleup and scaledown of CStorVolume", func() {
			It("Should able to perform scaleup and scaledown", func() {

				Expect(cspc).NotTo(BeNil(), "Specified CStor pools are not in healthy state")
				Expect(cvcSpecBuilder).NotTo(BeNil(), "cvcspec builder is not initilized")

				// Provision cStor CSI volume
				ProvisionCSIVolume(pvcName, testNS, scName, cstorsuite.infra.NodeCount)
				// Verify whether all the cStor volumes created
				VerifyCStorVolumeResourcesStatus(pvcName, testNS, cstorsuite.infra.NodeCount)

				// Below sinppet will scale the CStorVolume in unavailable CStorPoolInstance
				klog.Info("Scaling up CStorVolume in unavailable CStorPoolInstance")

				cvcCopy := cvcSpecBuilder.CVC.DeepCopy()
				cvcCopy.Spec.Policy.ReplicaPoolInfo = append(
					cvcCopy.Spec.Policy.ReplicaPoolInfo, cstorapis.ReplicaPoolInfo{PoolName: "cstor-pool-instance-unavailable"})
				_, err := cstorsuite.client.PatchCVCSpec(
					cvcCopy.Name, cvcCopy.Namespace, cvcCopy.Spec)
				Expect(err).To(BeNil(), "failed to patch cvc with new with unavailable pool name details")
				err = cstorsuite.client.WaitForCStorVolumeReplicaPools(
					cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, k8sclient.Poll, time.Second*1)
				Expect(err).NotTo(BeNil(), "CVC scaled in not existing pool")

				// //Bring CVC Spec to original
				// cvcSpecBuilder.RemovePoolsFromCVCSpec([]string{"cstor-pool-instance-unavailable"})
				// updatedCVC, err := cstorsuite.client.PatchCVCSpec(
				// 	cvcSpecBuilder.CVC.Name, cvcSpecBuilder.CVC.Namespace, cvcSpecBuilder.CVC.Spec)
				// Expect(err).To(BeNil(), "failed to remove unavailable pool name from cvc")
				// cvcSpecBuilder.SetCVCSpec(updatedCVC)

				// Below snippet will scale the CStorVolume in pool where already replica exist
				existingPoolName := cvcSpecBuilder.CVC.Status.PoolInfo[0]
				cvcCopy = cvcSpecBuilder.CVC.DeepCopy()
				cvcCopy.Spec.Policy.ReplicaPoolInfo = append(
					cvcCopy.Spec.Policy.ReplicaPoolInfo, cstorapis.ReplicaPoolInfo{PoolName: existingPoolName})
				_, err = cstorsuite.client.PatchCVCSpec(
					cvcCopy.Name, cvcCopy.Namespace, cvcCopy.Spec)
				Expect(err).NotTo(BeNil(), "Scaling in same pool should error out")

				if cstorsuite.infra.NodeCount > 1 {
					// Below snippet will remove multiple replicas at a time
					klog.Infof("Scaling down CStorVolume more than one replica at a time")
					cvcCopy = cvcSpecBuilder.CVC.DeepCopy()
					cvcCopy.Spec.Policy.ReplicaPoolInfo = []cstorapis.ReplicaPoolInfo{cvcSpecBuilder.CVC.Spec.Policy.ReplicaPoolInfo[0]}
					_, err = cstorsuite.client.PatchCVCSpec(
						cvcCopy.Name, cvcCopy.Namespace, cvcCopy.Spec)
					Expect(err).NotTo(BeNil(), "Scaling down CStorVolume more than one shouldn't be allowed")
				}

				DeProvisionVolume(pvcName, testNS, scName)

				err = cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})
		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				if cspc == nil {
					return
				}
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
				if cspc == nil {
					klog.Errorf("cstor-stripe-more-replicas CSPC is not created")
					return
				}
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
					Get(context.TODO(), pvc.Name, metav1.GetOptions{})
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
				err := cstorsuite.client.KubeClientSet.CoreV1().Namespaces().Delete(context.TODO(), testNS, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("Deprovisioning cspc", func() {
			Specify("no error should be returned during pool deprovisioning", func() {
				if cspc == nil {
					return
				}
				DeProvisionCSPC(cspc)
			})
		})
	})
}
