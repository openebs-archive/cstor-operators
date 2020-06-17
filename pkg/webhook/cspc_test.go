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
	"github.com/openebs/api/pkg/apis/types"
	clientset "github.com/openebs/api/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	kubeFakeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
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

func (f *fixture) fakeNodeCreator(nodeCount int) {
	for i := 1; i <= nodeCount; i++ {
		name := "worker-" + strconv.Itoa(i)
		nodeObj := getfakeNodeSpec(name)
		_, err := f.wh.kubeClient.CoreV1().Nodes().Create(nodeObj)
		if err != nil {
			klog.Error(err)
		}
	}
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

func (f *fixture) fakeBlockDeviceCreator(totalDisk, totalNodeCount int) {
	// Create some fake block device objects over nodes.
	var key, diskLabel string

	diskCountPerNode := totalDisk / totalNodeCount
	// nodeIdentifer will help in naming a node and attaching multiple disks to a single node.
	nodeIdentifer := 1
	for diskListIndex := 1; diskListIndex <= totalDisk; diskListIndex++ {
		diskIdentifier := strconv.Itoa(diskListIndex)

		if diskListIndex%diskCountPerNode == 0 {
			nodeIdentifer++
		}

		key = "ndm.io/blockdevice-type"
		diskLabel = "blockdevice"
		bdObj := &openebsapi.BlockDevice{
			TypeMeta: metav1.TypeMeta{
				Kind: "BlockDevices",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "blockdevice-" + diskIdentifier,
				UID:  k8stypes.UID("bdtest" + strconv.Itoa(nodeIdentifer) + diskIdentifier),
				Labels: map[string]string{
					"kubernetes.io/hostname": "worker-" + strconv.Itoa(nodeIdentifer),
					key:                      diskLabel,
				},
			},
			Spec: openebsapi.DeviceSpec{
				Details: openebsapi.DeviceDetails{
					DeviceType: "disk",
				},
				Partitioned: "NO",
				Capacity: openebsapi.DeviceCapacity{
					Storage: 120000000000,
				},
				NodeAttributes: openebsapi.NodeAttribute{
					NodeName: "worker-" + strconv.Itoa(nodeIdentifer),
				},
			},
			Status: openebsapi.DeviceStatus{
				State: openebsapi.BlockDeviceActive,
			},
		}
		_, err := f.wh.clientset.OpenebsV1alpha1().BlockDevices("openebs").Create(bdObj)
		if err != nil {
			klog.Error(err)
		}

	}
}

func (f *fixture) markBlockDeviceWithReplacementMarks(
	markBDUnderReplacement map[string]string, cspcName string) error {
	for newBD, oldBD := range markBDUnderReplacement {
		bdObj, err := f.wh.clientset.
			OpenebsV1alpha1().
			BlockDevices("openebs").
			Get(newBD, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get blockdevice %s", newBD)
		}
		hostName := bdObj.Labels["kubernetes.io/hostname"]
		// Build blockdeviceclaim to claim the blockdevice
		bdcObj := &openebsapi.BlockDeviceClaim{
			TypeMeta: metav1.TypeMeta{
				Kind: "BlockDeviceClaim",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blockdeviceclaim-" + newBD,
				UID:       k8stypes.UID("bdctest" + hostName + bdObj.Name),
				Namespace: "openebs",
				Labels: map[string]string{
					string(types.CStorPoolClusterLabelKey): cspcName,
				},
				Annotations: map[string]string{
					types.PredecessorBDLabelKey: oldBD,
				},
			},
			Spec: openebsapi.DeviceClaimSpec{
				BlockDeviceName: newBD,
			},
			Status: openebsapi.DeviceClaimStatus{
				Phase: openebsapi.BlockDeviceClaimStatusDone,
			},
		}
		// Create blockdeviceclaim for blockdevice
		_, err = f.wh.clientset.
			OpenebsV1alpha1().
			BlockDeviceClaims("openebs").
			Create(bdcObj)
		if err != nil {
			return errors.Wrapf(err, "failed to create claim for blockdevice %s", newBD)
		}
		// Bound blockdevice with blockdeviceclaim
		bdObj.Status.ClaimState = openebsapi.BlockDeviceClaimed
		bdObj.Spec.ClaimRef = &corev1.ObjectReference{
			Kind:      "BlockDeviceClaim",
			Name:      bdcObj.Name,
			Namespace: "openebs",
		}
		_, err = f.wh.clientset.
			OpenebsV1alpha1().
			BlockDevices("openebs").
			Update(bdObj)
		if err != nil {
			return errors.Wrapf(err, "failed to mark blockdevice %s as claimed", newBD)
		}
	}
	return nil
}

// TODO: remove below function
func getfakeBDs(nodeName string, diskCount int) []*openebsapi.BlockDevice {
	bds := []*openebsapi.BlockDevice{}
	for i := 1; i <= diskCount; i++ {
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
			for _, bd := range getfakeBDs("node1", 7) {
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

func TestBlockDeviceReplacement(t *testing.T) {
	f := newFixture().withOpenebsObjects().withKubeObjects()
	f.fakeNodeCreator(3)
	// Each node will have 20 blockdevices
	f.fakeBlockDeviceCreator(60, 3)
	tests := map[string]struct {
		// existingObj is object existing in etcd via fake client
		existingObj                      *cstor.CStorPoolCluster
		requestedObj                     *cstor.CStorPoolCluster
		markBlockDevicesUnderReplacement map[string]string
		expectedRsp                      bool
		getCSPCObj                       getCSPC
	}{
		"Replacement triggered on stripe pool": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
								WithName("blockdevice-1"))),
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
								WithName("blockdevice-21"))),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-stripe").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
								WithName("blockdevice-1"))),
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("stripe")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(*cstor.NewCStorPoolInstanceBlockDevice().
								WithName("blockdevice-22"))),
				),
			expectedRsp: false,
			getCSPCObj:  getCSPCObject,
		},
		"Replacement triggered on mirror pool": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
							),
						),
				),
			expectedRsp: true,
			getCSPCObj:  getCSPCObject,
		},
		"Replacement triggered on mirror pool which has two raid groups": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror-2").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-6"),
							),
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-7"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror-2").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-9"),
							),
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-10"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
							),
						),
				),
			expectedRsp: true,
			getCSPCObj:  getCSPCObject,
		},
		"Replacement triggered on RAIDZ pool": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-11"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-12"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-13"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-14"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-12"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-13"),
							),
						),
				),
			expectedRsp: true,
			getCSPCObj:  getCSPCObject,
		},
		"Invalid Replacement triggered on RAIDZ pool": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz-2").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								// Until claims are created there should not be any problem in using same blockdevices
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-22"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-23"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-24"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-raidz-2").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("raidz")).
						WithDataRaidGroups(
							*cstor.NewRaidGroup().WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-25"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-26"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-24"),
							),
						),
				),
			expectedRsp: false,
			getCSPCObj:  getCSPCObject,
		},
		"Replace blockdevice in raidgroup which is currently undergoing replacement": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror-3").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-27"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-28"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror-3").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-27"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-29"),
							),
						),
				),
			markBlockDevicesUnderReplacement: map[string]string{
				"blockdevice-27": "blockdevice-24",
			},
			expectedRsp: false,
			getCSPCObj:  getCSPCObject,
		},
	}
	// Set OPENEBS_NAMESPACE env
	os.Setenv("OPENEBS_NAMESPACE", "openebs")
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			ar := &v1beta1.AdmissionRequest{
				Operation: v1beta1.Create,
				Object: runtime.RawExtension{
					Raw: serialize(test.requestedObj),
				},
			}
			// Create fake object in etcd
			_, err := f.wh.clientset.CstorV1().
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
			if test.markBlockDevicesUnderReplacement != nil {
				if err = f.markBlockDeviceWithReplacementMarks(
					test.markBlockDevicesUnderReplacement, test.existingObj.Name); err != nil {
					t.Fatalf(
						"failed to mark blockdevice with replacement in progress error: %v",
						err,
					)
				}
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
	// Set OPENEBS_NAMESPACE env
	os.Unsetenv("OPENEBS_NAMESPACE")
}

func TestCSPCScaleDown(t *testing.T) {
	f := newFixture().withOpenebsObjects().withKubeObjects()
	f.fakeNodeCreator(3)
	// Each node will have 20 blockdevices
	f.fakeBlockDeviceCreator(60, 3)
	tests := map[string]struct {
		// existingObj is object existing in etcd via fake client
		existingObj   *cstor.CStorPoolCluster
		requestedObj  *cstor.CStorPoolCluster
		expectedRsp   bool
		getCSPCObj    getCSPC
		existingCSPIs []*cstor.CStorPoolInstance
		existingCVRs  []*cstor.CStorVolumeReplica
	}{
		"Negative scaledown case": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
							),
						),
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-3"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-4"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-foo-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-1"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-2"),
							),
						),
				),
			existingCSPIs: []*cstor.CStorPoolInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cspc-foo-mirror-1",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:         "worker-1",
							types.CStorPoolClusterLabelKey: "cspc-foo-mirror",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cspc-foo-mirror-2",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:         "worker-2",
							types.CStorPoolClusterLabelKey: "cspc-foo-mirror",
						},
					},
				},
			},
			existingCVRs: []*cstor.CStorVolumeReplica{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cvr-cspc-foo-mirror-1",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:              "worker-1",
							types.CStorPoolInstanceNameLabelKey: "cspc-foo-mirror-1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cvr-cspc-foo-mirror-2",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:              "worker-2",
							types.CStorPoolInstanceNameLabelKey: "cspc-foo-mirror-2",
						},
					},
				},
			},
			expectedRsp: false,
			getCSPCObj:  getCSPCObject,
		},
		"Positive scaledown case": {
			existingObj: cstor.NewCStorPoolCluster().
				WithName("cspc-bar-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-6"),
							),
						),
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-2"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-7"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-8"),
							),
						),
				),
			requestedObj: cstor.NewCStorPoolCluster().
				WithName("cspc-bar-mirror").
				WithNamespace("openebs").
				WithPoolSpecs(
					*cstor.NewPoolSpec().
						WithNodeSelector(map[string]string{types.HostNameLabelKey: "worker-1"}).
						WithPoolConfig(*cstor.NewPoolConfig().
							WithDataRaidGroupType("mirror")).
						WithDataRaidGroups(*cstor.NewRaidGroup().
							WithCStorPoolInstanceBlockDevices(
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-5"),
								*cstor.NewCStorPoolInstanceBlockDevice().WithName("blockdevice-6"),
							),
						),
				),
			existingCSPIs: []*cstor.CStorPoolInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cspc-bar-mirror-1",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:         "worker-1",
							types.CStorPoolClusterLabelKey: "cspc-bar-mirror",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cspc-bar-mirror-2",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:         "worker-2",
							types.CStorPoolClusterLabelKey: "cspc-bar-mirror",
						},
					},
				},
			},
			existingCVRs: []*cstor.CStorVolumeReplica{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cvr=cspc-bar-mirror-1",
						Namespace: "openebs",
						Labels: map[string]string{
							types.HostNameLabelKey:              "worker-1",
							types.CStorPoolInstanceNameLabelKey: "cspc-bar-mirror-1",
						},
					},
				},
			},
			expectedRsp: true,
			getCSPCObj:  getCSPCObject,
		},
	}
	// Set OPENEBS_NAMESPACE env
	os.Setenv("OPENEBS_NAMESPACE", "openebs")
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			ar := &v1beta1.AdmissionRequest{
				Operation: v1beta1.Create,
				Object: runtime.RawExtension{
					Raw: serialize(test.requestedObj),
				},
			}
			// Create fake object in etcd
			_, err := f.wh.clientset.CstorV1().
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
			for _, cspi := range test.existingCSPIs {
				_, err := f.wh.clientset.CstorV1().
					CStorPoolInstances(cspi.Namespace).
					Create(cspi)
				if err != nil {
					t.Fatalf(
						"failed to create fake CSPI %s Object in Namespace %s error: %v",
						cspi.Name,
						cspi.Namespace,
						err,
					)
				}
			}
			for _, cvr := range test.existingCVRs {
				_, err := f.wh.clientset.CstorV1().
					CStorVolumeReplicas(cvr.Namespace).
					Create(cvr)
				if err != nil {
					t.Fatalf(
						"failed to create fake CVR %s Object in Namespace %s error: %v",
						cvr.Name,
						cvr.Namespace,
						err,
					)
				}
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
	// Set OPENEBS_NAMESPACE env
	os.Unsetenv("OPENEBS_NAMESPACE")
}
