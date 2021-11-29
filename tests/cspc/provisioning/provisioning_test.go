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
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/tests/pkg/cspc/cspcspecbuilder"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

/*
 This test file covers following test cases :

 1. Stripe pool provisioning with multiple disks and raid groups
    ( includes write cache and data raid groups )
 2. Mirror pool provisioning with multiple disks and raid groups
    (includes write cache and data raid groups )
 3. Raidz1 pool provisioning with multiple disks and raid groups
    (includes write cache and data raid groups )
 4. Raidz2 pool provisioning with multiple disks and raid groups
    (includes write cache and data raid groups )

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

var _ = Describe("CSPC", func() {
	ProvisioningTest("stripe", 1)
	ProvisioningTest("mirror", 2)
	ProvisioningTest("raidz", 5)
	ProvisioningTest("raidz2", 6)

	OperationsTest("mirror", 2)
	OperationsTest("raidz", 5)
	OperationsTest("raidz2", 6)

	TunablesTest("stripe", 1)

})

func NewResourceLimit() *corev1.ResourceRequirements {
	return &corev1.ResourceRequirements{
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}
}

func NewTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:    "openebs.io/integration-test",
			Value:  "",
			Effect: corev1.TaintEffectNoExecute,
		},
	}
}

func OperationsTest(poolType string, bdCount int) {
	var cspc *cstor.CStorPoolCluster
	var specBuilder *cspcspecbuilder.CSPCSpecBuilder
	Describe(poolType+" CSPC", func() {
		Context("Block Device replacment", func() {
			Specify("creatin the cspc,no error should be returned", func() {
				specBuilder = cspcspecbuilder.
					NewCSPCSpecBuilder(cspcsuite.CSPCCache, cspcsuite.infra)

				cspc = specBuilder.BuildCSPC("cspc-foo", "openebs", poolType, bdCount, cspcsuite.infra.NodeCount).GetCSPCSpec()
				_, err := cspcsuite.
					client.
					OpenEBSClientSet.
					CstorV1().
					CStorPoolClusters(cspc.Namespace).
					Create(context.TODO(), cspc, metav1.CreateOptions{})
				Expect(err).To(BeNil())

			})

			Specify("All the CSPI should be healthy", func() {
				gotHealthyCSPiCount := cspcsuite.
					client.
					GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, cspcsuite.infra.NodeCount)
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
			})

			Specify("replacement of a block device should be successful",
				func() {
					poolSpecPos := 0
					updatedSuccessfully := false
					rt := cspcspecbuilder.NewReplacementTracer()
					for i := 0; i < 4; i++ {
						gotCSPC, err := cspcsuite.
							client.
							OpenEBSClientSet.
							CstorV1().
							CStorPoolClusters(cspc.Namespace).
							Get(context.TODO(), cspc.Name, metav1.GetOptions{})
						if err != nil {
							klog.Warningf("Retrying to update CSPC:%s", err.Error())
							time.Sleep(3 * time.Second)
							continue
						}
						specBuilder.SetCSPCSpec(gotCSPC)

						if rt.Replaced {
							cspc = specBuilder.ReplaceBlockDevice(rt.OldBD, rt.NewBD).GetCSPCSpec()
						} else {
							cspc = specBuilder.ReplaceBlockDeviceAtPos(poolSpecPos, 0, 0, rt).GetCSPCSpec()
						}

						_, err = cspcsuite.
							client.
							OpenEBSClientSet.
							CstorV1().
							CStorPoolClusters(cspc.Namespace).
							Update(context.TODO(), cspc, metav1.UpdateOptions{})
						if err == nil {
							updatedSuccessfully = true
							break
						} else {
							klog.Warningf("Retrying to update CSPC:%s", err.Error())
							time.Sleep(3 * time.Second)
						}
					}

					if !updatedSuccessfully {
						klog.Fatal("could not update the cspc for bd replacment")
					} else {
						klog.Info("updated cspc successfully for bd replacment")
					}

					cspiHostName := cspc.Spec.Pools[poolSpecPos].NodeSelector[types.HostNameLabelKey]
					gotStatus := cspcsuite.
						client.
						GetBDReplacmentStatusOnCSPI(cspc.Name, cspc.Namespace, cspiHostName, true)
					Expect(gotStatus).To(BeTrue())
				})
			// Following are cleanup test cases
			Context("Deleting the cspc", func() {

				It("No error should be returned", func() {
					err := cspcsuite.
						client.
						OpenEBSClientSet.
						CstorV1().
						CStorPoolClusters(cspc.Namespace).
						Delete(context.TODO(), cspc.Name, metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					// The CSPCSpecData should be cleared
					specBuilder.ResetCSPCSpecData()
				})

				It("No corresponding cspi(s) should be present", func() {
					gotCSPICount := cspcsuite.
						client.
						GetCSPICountEventually(cspc.Name, cspc.Namespace, 0)
					Expect(gotCSPICount).To(BeNumerically("==", 0))
				})

				It("No corresponding pool-manger deployments should be present", func() {
					gotPoolMangerCount := cspcsuite.
						client.
						GetPoolManagerCountEventually(cspc.Name, cspc.Namespace, 0)
					Expect(gotPoolMangerCount).To(BeNumerically("==", 0))
				})

				It("the bdc(s) created by cstor-operator should get deleted", func() {
					gotCount := cspcsuite.
						client.
						GetBDCCountEventually(cspc.Name, cspc.Namespace, 0)
					Expect(gotCount).To(BeNumerically("==", 0))
				})

				It("CSPC should get removed from cluster", func() {
					var isCSPCDeleted bool
					retryCount := 20
					for retryCount > 0 {
						_, err := cspcsuite.client.OpenEBSClientSet.CstorV1().CStorPoolClusters(cspc.Namespace).Get(context.TODO(), cspc.Name, metav1.GetOptions{})
						if err != nil && k8serrors.IsNotFound(err) {
							isCSPCDeleted = true
							break
						}
						retryCount--
						time.Sleep(5 * time.Second)
					}
					Expect(isCSPCDeleted).Should(BeTrue(), "cspc %s/%s should get deleted", cspc.Namespace, cspc.Name)
				})
			})

		})
	})
}

func ProvisioningTest(poolType string, bdCount int) {
	var cspc *cstor.CStorPoolCluster
	var specBuilder *cspcspecbuilder.CSPCSpecBuilder
	Describe(poolType+" tests", func() {
		Context("Creating cspc", func() {
			Specify("no error should be returned", func() {
				specBuilder = cspcspecbuilder.
					NewCSPCSpecBuilder(cspcsuite.CSPCCache, cspcsuite.infra)

				cspc = specBuilder.BuildCSPC("cspc-foo", "openebs", poolType, bdCount, cspcsuite.infra.NodeCount).GetCSPCSpec()
				_, err := cspcsuite.
					client.
					OpenEBSClientSet.
					CstorV1().
					CStorPoolClusters(cspc.Namespace).
					Create(context.TODO(), cspc, metav1.CreateOptions{})
				Expect(err).To(BeNil())

			})

			Specify("desired count should be on cspc",
				func() {
					gotCount := cspcsuite.
						client.
						GetDesiredInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(cspcsuite.infra.NodeCount))
					Expect(gotCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
				})

		})

		Context("All the cspi(s) of the cspc", func() {
			It("Should be healthy", func() {
				gotHealthyCSPiCount := cspcsuite.
					client.
					GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, cspcsuite.infra.NodeCount)
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
			})
		})

		Context("Staus of the cspc i.e. provisionedInstances and healthyInstances ", func() {
			It("Should be updated", func() {
				gotProvisionedCount := cspcsuite.
					client.
					GetProvisionedInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(cspcsuite.infra.NodeCount))
				Expect(gotProvisionedCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))

				gotHealthyCount := cspcsuite.
					client.
					GetHealthyInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(cspcsuite.infra.NodeCount))
				Expect(gotHealthyCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
			})
		})

		// Following are scale up and down test cases.
		Context("Remove 1 pool spec from the CSPC", func() {
			Specify("desired count should be updated on cspc",
				func() {
					// We can remove spec only if test is running on multi node cluster
					if cspcsuite.infra.NodeCount == 1 {
						return
					}
					gotCSPC, err := cspcsuite.
						client.
						OpenEBSClientSet.
						CstorV1().
						CStorPoolClusters(cspc.Namespace).
						Get(context.TODO(), cspc.Name, metav1.GetOptions{})
					Expect(err).To(BeNil())
					specBuilder.SetCSPCSpec(gotCSPC)
					cspc = specBuilder.RemovePoolSpec().GetCSPCSpec()

					_, err = cspcsuite.
						client.
						OpenEBSClientSet.
						CstorV1().
						CStorPoolClusters(cspc.Namespace).
						Update(context.TODO(), cspc, metav1.UpdateOptions{})
					Expect(err).To(BeNil())

					gotCount := cspcsuite.
						client.
						GetDesiredInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(len(cspc.Spec.Pools)))
					Expect(gotCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))
				})

			It("CSPI copunt should be equal to no. of pool spec and be healthy", func() {
				gotHealthyCSPiCount := cspcsuite.
					client.
					GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, len(cspc.Spec.Pools))
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))
			})

			It("Staus of the cspc i.e. provisionedInstances and healthyInstances should be updated", func() {
				gotProvisionedCount := cspcsuite.
					client.
					GetProvisionedInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(len(cspc.Spec.Pools)))
				Expect(gotProvisionedCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))

				gotHealthyCount := cspcsuite.
					client.
					GetHealthyInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(len(cspc.Spec.Pools)))
				Expect(gotHealthyCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))
			})

		})

		Context("Add 1 pool spec to the CSPC", func() {

			Specify("desired count should be updated on cspc",
				func() {
					// We can add spec only if test is running on multi node cluster
					if cspcsuite.infra.NodeCount == 1 {
						return
					}
					var nodeName string

					for k := range specBuilder.CSPCSpecData.UnUsedNodes {
						nodeName = k
						break
					}

					gotCSPC, err := cspcsuite.
						client.
						OpenEBSClientSet.
						CstorV1().
						CStorPoolClusters(cspc.Namespace).
						Get(context.TODO(), cspc.Name, metav1.GetOptions{})
					Expect(err).To(BeNil())
					specBuilder.SetCSPCSpec(gotCSPC)
					cspc = specBuilder.AddPoolSpec(nodeName, poolType, bdCount).GetCSPCSpec()
					_, err = cspcsuite.
						client.
						OpenEBSClientSet.
						CstorV1().
						CStorPoolClusters(cspc.Namespace).
						Update(context.TODO(), cspc, metav1.UpdateOptions{})
					Expect(err).To(BeNil())
					gotCount := cspcsuite.
						client.
						GetDesiredInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(len(cspc.Spec.Pools)))
					Expect(gotCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))
				})

			It("CSPI copunt should be equal to no. of pool spec and be healthy", func() {
				gotHealthyCSPiCount := cspcsuite.
					client.
					GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, len(cspc.Spec.Pools))
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))
			})

			It("Staus of the cspc i.e. provisionedInstances and healthyInstances should be updated", func() {
				gotProvisionedCount := cspcsuite.
					client.
					GetProvisionedInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(len(cspc.Spec.Pools)))
				Expect(gotProvisionedCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))

				gotHealthyCount := cspcsuite.
					client.
					GetHealthyInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(len(cspc.Spec.Pools)))
				Expect(gotHealthyCount).To(BeNumerically("==", int32(len(cspc.Spec.Pools))))
			})

		})

		// Following are cleanup test cases
		Context("Deleting the cspc", func() {

			It("No error should be returned", func() {
				err := cspcsuite.
					client.
					OpenEBSClientSet.
					CstorV1().
					CStorPoolClusters(cspc.Namespace).
					Delete(context.TODO(), cspc.Name, metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				// The CSPCSpecData should be cleared
				specBuilder.ResetCSPCSpecData()
			})

			It("No corresponding cspi(s) should be present", func() {
				gotCSPICount := cspcsuite.
					client.
					GetCSPICountEventually(cspc.Name, cspc.Namespace, 0)
				Expect(gotCSPICount).To(BeNumerically("==", 0))
			})

			It("No corresponding pool-manger deployments should be present", func() {
				gotPoolMangerCount := cspcsuite.
					client.
					GetPoolManagerCountEventually(cspc.Name, cspc.Namespace, 0)
				Expect(gotPoolMangerCount).To(BeNumerically("==", 0))
			})

			It("the bdc(s) created by cstor-operator should get deleted", func() {
				gotCount := cspcsuite.
					client.
					GetBDCCountEventually(cspc.Name, cspc.Namespace, 0)
				Expect(gotCount).To(BeNumerically("==", 0))
			})

			It("CSPC should removed from cluster", func() {
				var isCSPCDeleted bool
				retryCount := 20
				for retryCount > 0 {
					_, err := cspcsuite.client.OpenEBSClientSet.CstorV1().CStorPoolClusters(cspc.Namespace).Get(context.TODO(), cspc.Name, metav1.GetOptions{})
					if err != nil && k8serrors.IsNotFound(err) {
						isCSPCDeleted = true
						break
					}
					retryCount--
					time.Sleep(5 * time.Second)
				}
				Expect(isCSPCDeleted).Should(BeTrue(), "cspc %s/%s should get deleted", cspc.Namespace, cspc.Name)
			})
		})
	})
}

func TunablesTest(poolType string, bdCount int) {
	var cspc *cstor.CStorPoolCluster
	var specBuilder *cspcspecbuilder.CSPCSpecBuilder
	priorityClassName := "integration-test-priority"
	compression := "lzjb"
	roThreshold := 70
	Describe(poolType+" CSPC", func() {
		Context("Pass resource and limit via CSPC", func() {
			Specify("creating the cspc,no error should be returned", func() {
				specBuilder = cspcspecbuilder.
					NewCSPCSpecBuilder(cspcsuite.CSPCCache, cspcsuite.infra)

				cspc = specBuilder.
					BuildCSPC("cspc-foo", "openebs", poolType, bdCount, cspcsuite.infra.NodeCount).
					AddResourceLimits(NewResourceLimit()).
					AddTolerations(NewTolerations()).
					AddPriorityClass(&priorityClassName).
					AddCompression(compression).
					AddRoThreshold(&roThreshold).
					GetCSPCSpec()

				_, err := cspcsuite.
					client.
					OpenEBSClientSet.
					CstorV1().
					CStorPoolClusters(cspc.Namespace).
					Create(context.TODO(), cspc, metav1.CreateOptions{})
				Expect(err).To(BeNil())

			})
			// Here we are only checking for CSPI creation as passing
			// parameters like priority class, tolerations can cause the pool
			// manager pod to be in pending state.
			// The intent of the tests here is to only check whether the tunables are
			// passed or not.
			Specify("Expected number of CSPI should be created", func() {
				gotHealthyCSPiCount := cspcsuite.
					client.
					GetCSPICountEventually(cspc.Name, cspc.Namespace, cspcsuite.infra.NodeCount)
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
			})

			Specify("Expected number of pool manager deployments should be created", func() {
				gotHealthyCSPiCount := cspcsuite.
					client.
					GetPoolManagerCountEventually(cspc.Name, cspc.Namespace, cspcsuite.infra.NodeCount)
				Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
			})

			Specify("Resource limits should be passed to the CSPI", func() {
				resoureLimitMatches := cspcsuite.
					client.HasResourceLimitOnCSPIEventually(cspc.Name, cspc.Namespace, NewResourceLimit())
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("Resource limits should be passed to the cstor-pool container", func() {
				resoureLimitMatches := cspcsuite.
					client.HasResourceLimitOnPoolManagerEventually(cspc.Name, cspc.Namespace, NewResourceLimit())
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("Tolerations should be passed to the CSPI", func() {
				resoureLimitMatches := cspcsuite.
					client.HasTolerationsOnCSPIEventually(cspc.Name, cspc.Namespace, NewTolerations())
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("Tolerations should be passed to the pool manager deployments", func() {
				resoureLimitMatches := cspcsuite.
					client.HasTolerationsOnPoolManagerEventually(cspc.Name, cspc.Namespace, NewTolerations())
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("Priority class should be passed to the CSPI", func() {
				resoureLimitMatches := cspcsuite.
					client.HasPriorityClassOnCSPIEventually(cspc.Name, cspc.Namespace, &priorityClassName)
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("Priority class should be passed to the pool manager deployments", func() {
				resoureLimitMatches := cspcsuite.
					client.HasPriorityClassOnPoolManagerEventually(cspc.Name, cspc.Namespace, &priorityClassName)
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("Compression should be passed to the CSPI", func() {
				resoureLimitMatches := cspcsuite.
					client.HasCompressionOnCSPIEventually(cspc.Name, cspc.Namespace, compression)
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Specify("RO threshold should be passed to the CSPI", func() {
				resoureLimitMatches := cspcsuite.
					client.HasROThresholdOnCSPIEventually(cspc.Name, cspc.Namespace, &roThreshold)
				Expect(resoureLimitMatches).To(BeTrue())
			})

			Context("Deleting the cspc", func() {

				It("No error should be returned", func() {
					err := cspcsuite.
						client.
						OpenEBSClientSet.
						CstorV1().
						CStorPoolClusters(cspc.Namespace).
						Delete(context.TODO(), cspc.Name, metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					// The CSPCSpecData should be cleared
					specBuilder.ResetCSPCSpecData()
				})

				It("No corresponding cspi(s) should be present", func() {
					gotCSPICount := cspcsuite.
						client.
						GetCSPICountEventually(cspc.Name, cspc.Namespace, 0)
					Expect(gotCSPICount).To(BeNumerically("==", 0))
				})

				It("No corresponding pool-manger deployments should be present", func() {
					gotPoolMangerCount := cspcsuite.
						client.
						GetPoolManagerCountEventually(cspc.Name, cspc.Namespace, 0)
					Expect(gotPoolMangerCount).To(BeNumerically("==", 0))
				})

				It("the bdc(s) created by cstor-operator should get deleted", func() {
					gotCount := cspcsuite.
						client.
						GetBDCCountEventually(cspc.Name, cspc.Namespace, 0)
					Expect(gotCount).To(BeNumerically("==", 0))
				})
			})

		})
	})
}
