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

package webhook

import (
	"os"
	"strconv"
	"testing"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
	openebsapi "github.com/openebs/api/pkg/apis/openebs.io/v1alpha1"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeFakeClient "k8s.io/client-go/kubernetes/fake"
)

func TestValidateSpecChanges(t *testing.T) {
	tests := map[string]struct {
		commonPoolSpecs *poolspecs
		pOps            *PoolOperations
		expectedOutput  bool
	}{
		"No change in poolSpecs": {
			commonPoolSpecs: &poolspecs{
				oldSpec: []cstor.PoolSpec{
					cstor.PoolSpec{
						DataRaidGroups: []cstor.RaidGroup{
							cstor.RaidGroup{
								CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
									cstor.CStorPoolInstanceBlockDevice{
										BlockDeviceName: "bd1",
									},
									cstor.CStorPoolInstanceBlockDevice{
										BlockDeviceName: "bd2",
									},
								},
							},
						},
						PoolConfig: cstor.PoolConfig{
							DataRaidGroupType: "mirror",
						},
					},
				},
				newSpec: []cstor.PoolSpec{
					cstor.PoolSpec{
						DataRaidGroups: []cstor.RaidGroup{
							cstor.RaidGroup{
								CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
									cstor.CStorPoolInstanceBlockDevice{
										BlockDeviceName: "bd1",
									},
									cstor.CStorPoolInstanceBlockDevice{
										BlockDeviceName: "bd2",
									},
								},
							},
						},
						PoolConfig: cstor.PoolConfig{
							DataRaidGroupType: "mirror",
						},
					},
				},
			},
			pOps: &PoolOperations{
				OldCSPC: &cstor.CStorPoolCluster{},
				NewCSPC: &cstor.CStorPoolCluster{},
			},
			expectedOutput: true,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			isValid, _ := ValidateSpecChanges(test.commonPoolSpecs, test.pOps)
			if isValid != test.expectedOutput {
				t.Errorf("test: %s failed expected output %t but got %t", name, isValid, test.expectedOutput)
			}
		})
	}
}

func (f *fixture) withKubeObjects(objects ...runtime.Object) *fixture {
	f.openebsObjects = objects
	f.wh.kubeClient = kubeFakeClient.NewSimpleClientset(objects...)
	return f
}

func fakeGetCSPCError(name, namespace string, clientset clientset.Interface) (*cstor.CStorPoolCluster, error) {
	return nil, errors.Errorf("fake error")
}

func getfakeNodeSpec(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"beta.kubernetes.io/arch": "amd64",
				"beta.kubernetes.io/os":   "linux",
				"kubernetes.io/arch":      "amd64",
				"kubernetes.io/hostname":  name,
				"kubernetes.io/os":        "linux",
			},
		},
	}
}

func getfakeBDs(nodeName string) []*openebsapi.BlockDevice {
	bds := []*openebsapi.BlockDevice{}
	for i := 1; i < 7; i++ {
		bd := &openebsapi.BlockDevice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blockdevice-" + strconv.Itoa(i),
				Namespace: "openebs",
				Labels: map[string]string{
					"kubernetes.io/hostname":  nodeName,
					"ndm.io/blockdevice-type": "blockdevice",
					"ndm.io/managed":          "true",
				},
			},
			Spec: openebsapi.DeviceSpec{
				Capacity: openebsapi.DeviceCapacity{
					Storage: 10737418240,
				},
				NodeAttributes: openebsapi.NodeAttribute{
					NodeName: nodeName,
				},
			},
			Status: openebsapi.DeviceStatus{
				ClaimState: openebsapi.BlockDeviceUnclaimed,
				State:      openebsapi.BlockDeviceActive,
			},
		}
		bds = append(bds, bd)
	}
	return bds
}

func TestValidateCSPCUpdateRequest(t *testing.T) {
	f := newFixture().withOpenebsObjects().withKubeObjects()
	tests := map[string]struct {
		// existingObj is object existing in etcd via fake client
		existingObj  *cstor.CStorPoolCluster
		requestedObj *cstor.CStorPoolCluster
		expectedRsp  bool
		getCSPCObj   getCSPC
	}{
		"When Failed to Get Object From etcd": {
			existingObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc1",
					Namespace: "openebs",
				},
			},
			requestedObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc1",
					Namespace: "openebs",
				},
				Status: cstor.CStorPoolClusterStatus{
					ProvisionedInstances: 1,
				},
			},
			expectedRsp: false,
			getCSPCObj:  fakeGetCSPCError,
		},
		"Positive stripe expansion test": {
			existingObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc2",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType: "stripe",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
									},
								},
							},
						},
					},
				},
			},
			requestedObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc2",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType: "stripe",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-3",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRsp: true,
			getCSPCObj:  getCSPCObject,
		},
		"Positive mirror expansion test": {
			existingObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc3",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType: "mirror",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
									},
								},
							},
						},
					},
				},
			},
			requestedObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc3",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType: "mirror",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
									},
								},
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-3",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-4",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRsp: true,
			getCSPCObj:  getCSPCObject,
		},
		"Negative mirror expansion test, adding bds in same raidGroup": {
			existingObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc4",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType: "mirror",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
									},
								},
							},
						},
					},
				},
			},
			requestedObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc4",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType: "mirror",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-3",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-4",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRsp: false,
			getCSPCObj:  getCSPCObject,
		},
		"Negative mirror replacement test, swap between data and writecache": {
			existingObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc5",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType:   "mirror",
								WriteCacheGroupType: "mirror",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
									},
								},
							},
							WriteCacheRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-3",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-4",
										},
									},
								},
							},
						},
					},
				},
			},
			requestedObj: &cstor.CStorPoolCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cspc5",
					Namespace: "openebs",
				},
				Spec: cstor.CStorPoolClusterSpec{
					Pools: []cstor.PoolSpec{
						cstor.PoolSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "node1",
							},
							PoolConfig: cstor.PoolConfig{
								DataRaidGroupType:   "mirror",
								WriteCacheGroupType: "mirror",
							},
							DataRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-1",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-4",
										},
									},
								},
							},
							WriteCacheRaidGroups: []cstor.RaidGroup{
								cstor.RaidGroup{
									CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-3",
										},
										cstor.CStorPoolInstanceBlockDevice{
											BlockDeviceName: "blockdevice-2",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRsp: false,
			getCSPCObj:  getCSPCObject,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			ar := &v1beta1.AdmissionRequest{
				Operation: v1beta1.Create,
				Object: runtime.RawExtension{
					Raw: serialize(test.requestedObj),
				},
			}
			// Set OPENEBS_NAMESPACE env
			os.Setenv("OPENEBS_NAMESPACE", "openebs")
			// Create fake node object in etcd
			_, err := f.wh.kubeClient.CoreV1().Nodes().
				Create(getfakeNodeSpec("node1"))
			// Create fake bd objects in etcd
			for _, bd := range getfakeBDs("node1") {
				_, err = f.wh.clientset.OpenebsV1alpha1().
					BlockDevices(bd.Namespace).
					Create(bd)
			}
			// Create fake object in etcd
			_, err = f.wh.clientset.CstorV1().
				CStorPoolClusters(test.existingObj.Namespace).
				Create(test.existingObj)
			if err != nil {
				t.Fatalf(
					"failed to create fake CSPC %s Object in Namespace %s error: %v",
					test.existingObj.Name,
					test.existingObj.Namespace,
					err,
				)
			}
			resp := f.wh.validateCSPCUpdateRequest(ar, test.getCSPCObj)
			if resp.Allowed != test.expectedRsp {
				t.Errorf(
					"%s test case failed expected response: %t but got %t error: %s",
					name,
					test.expectedRsp,
					resp.Allowed,
					resp.Result.Message,
				)
			}
		})
	}
}
