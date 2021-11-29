/*
Copyright 2021 The OpenEBS Authors

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
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/cstor-operators/tests/pkg/cspc/cspcspecbuilder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("VERIFY CSPC POOL PROPERTIES", func() {
	var (
		cspcName         = "cspc-property-check"
		openebsNamespace = "openebs"
		poolType         = "stripe"

		poolName        string
		selectedPoolPod *corev1.Pod
		cspc            *cstor.CStorPoolCluster
		specBuilder     *cspcspecbuilder.CSPCSpecBuilder
	)

	When("CSPC stripe based configuration is applied", func() {
		It("Should create cStor pool", func() {
			var err error
			specBuilder = cspcspecbuilder.NewCSPCSpecBuilder(cspcsuite.CSPCCache, cspcsuite.infra)
			builtCSPC := specBuilder.BuildCSPC(cspcName, openebsNamespace, poolType, 1, cspcsuite.infra.NodeCount).GetCSPCSpec()
			cspc, err = cspcsuite.client.OpenEBSClientSet.CstorV1().CStorPoolClusters(builtCSPC.Namespace).Create(context.TODO(), builtCSPC, metav1.CreateOptions{})
			Expect(err).To(BeNil(), "while creating %s/%s cspc", openebsNamespace, cspcName)

			// Verify pool activeness
			gotCount := cspcsuite.client.GetDesiredInstancesStatusOnCSPC(cspc.Name, cspc.Namespace, int32(cspcsuite.infra.NodeCount))
			Expect(gotCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))

			gotHealthyCSPiCount := cspcsuite.client.GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, cspcsuite.infra.NodeCount)
			Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(cspcsuite.infra.NodeCount)))
		})
	})

	When("CSPC stripe pool is created", func() {
		It("Should have default pool and filesystem properties", func() {

			// Verifying default configuration on any one pool pod is good enough
			poolPodList, err := cspcsuite.client.KubeClientSet.CoreV1().
				Pods(openebsNamespace).
				List(context.TODO(), metav1.ListOptions{LabelSelector: string(types.CStorPoolClusterLabelKey) + "=" + cspcName})
			Expect(err).To(BeNil(), "while listing cStor pool pods of cspc %s/%s", openebsNamespace, cspcName)
			selectedPoolPod = &poolPodList.Items[0]
			for _, container := range selectedPoolPod.Spec.Containers {
				if container.Name == "cstor-pool-mgmt" {
					for _, env := range container.Env {
						if env.Name == "OPENEBS_IO_POOL_NAME" {
							poolName = "cstor-" + env.Value
							break
						}
					}
				}
				if poolName != "" {
					break
				}
			}

			cspiName := selectedPoolPod.Labels[string(types.CStorPoolInstanceLabelKey)]
			command := "zfs get compression,canmount -Hp -o property,value " + poolName
			stdout, stderr, err := cspcsuite.client.Exec(command, selectedPoolPod.Name, "cstor-pool-mgmt", openebsNamespace)
			Expect(err).To(BeNil(), "while getting pool properties stderr: %s", stderr)
			stdoutList := strings.Split(stdout, "\n")
			for _, propValue := range stdoutList {
				if strings.HasPrefix(propValue, "compression") {
					Expect(propValue).Should(HaveSuffix("lz4"), "pool %s should have compression lz4", cspiName)
				} else if strings.HasPrefix(propValue, "canmount") {
					Expect(propValue).Should(HaveSuffix("off"), "pool %s should have canmount off", cspiName)
				}
			}
		})
	})

	When("CSPC compression property is updated", func() {
		It("should update compression property in pool", func() {
			var err error
			Expect(selectedPoolPod).NotTo(BeNil(), "pool pod must be selected to run the test")
			cspc, err = cspcsuite.client.OpenEBSClientSet.CstorV1().
				CStorPoolClusters(openebsNamespace).
				Get(context.TODO(), cspcName, metav1.GetOptions{})
			Expect(err).To(BeNil(), "while fetching cspc %s/%s", openebsNamespace, cspcName)
			for index := range cspc.Spec.Pools {
				cspc.Spec.Pools[index].PoolConfig.Compression = "gzip"
			}

			cspc, err = cspcsuite.client.OpenEBSClientSet.CstorV1().
				CStorPoolClusters(openebsNamespace).
				Update(context.TODO(), cspc, metav1.UpdateOptions{})
			Expect(err).To(BeNil(), "while updating cspc pool properties")

			var isCompressionPropertyUpdated bool
			command := "zfs get compression,canmount -Hp -o property,value " + poolName
			for retryCount := 0; retryCount < 20; retryCount++ {
				stdout, stderr, err := cspcsuite.client.Exec(command, selectedPoolPod.Name, "cstor-pool-mgmt", openebsNamespace)
				Expect(err).To(BeNil(), "while getting pool properties stderr: %s", stderr)
				stdoutList := strings.Split(stdout, "\n")
				for _, propValue := range stdoutList {
					if strings.HasPrefix(propValue, "compression") {
						if strings.HasSuffix(propValue, "gzip") {
							isCompressionPropertyUpdated = true
							break
						}
					} else if strings.HasPrefix(propValue, "canmount") {
						Expect(propValue).Should(HaveSuffix("off"), "cspc based pool %s/%s should have canmount off", openebsNamespace, cspcName)
					}
				}

				if isCompressionPropertyUpdated == true {
					break
				}
				time.Sleep(time.Second * 10)
			}
			Expect(isCompressionPropertyUpdated).Should(BeTrue(), "pool %s should configure compression property")
		})
	})

	When("CSPC is deleted", func() {
		It("should delete pool and all its' dependencies", func() {
			Expect(cspc).NotTo(BeNil(), "cspc object must not be empty")
			err := cspcsuite.client.OpenEBSClientSet.CstorV1().CStorPoolClusters(cspc.Namespace).Delete(context.TODO(), cspc.Name, metav1.DeleteOptions{})
			Expect(err).To(BeNil(), "while deleting CSPC pools")

			gotCSPICount := cspcsuite.client.GetCSPICountEventually(cspc.Name, cspc.Namespace, 0)
			Expect(gotCSPICount).To(BeNumerically("==", 0))

			gotPoolMangerCount := cspcsuite.client.GetPoolManagerCountEventually(cspc.Name, cspc.Namespace, 0)
			Expect(gotPoolMangerCount).To(BeNumerically("==", 0))
		})
	})
})
