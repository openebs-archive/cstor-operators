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

package algorithm

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	openebsio "github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/v3/pkg/apis/types"
	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type fixture struct {
	kubeclient    *fake.Clientset
	openebsClient *openebsFakeClientset.Clientset
	kubeObjects   []runtime.Object
	openebsObject []runtime.Object
}

func (f *fixture) WithOpenEBSObjects(objects ...runtime.Object) *fixture {
	f.openebsObject = objects
	f.openebsClient = openebsFakeClientset.NewSimpleClientset(objects...)
	return f
}

func (f *fixture) WithKubeObjects(objects ...runtime.Object) *fixture {
	f.kubeObjects = objects
	f.kubeclient = fake.NewSimpleClientset(objects...)
	return f
}

func NewNodeList(nodeCount int) *corev1.NodeList {
	newNodeList := &corev1.NodeList{}
	for i := 1; i <= nodeCount; i++ {
		newNode := &corev1.Node{}
		newNode.Name = "node" + strconv.Itoa(i)
		newNode.Labels = map[string]string{types.HostNameLabelKey: "node" + strconv.Itoa(i)}
		newNodeList.Items = append(newNodeList.Items, *newNode)
	}
	return newNodeList
}

func NewFixture() *fixture {
	return &fixture{
		kubeclient:    fake.NewSimpleClientset(),
		openebsClient: openebsFakeClientset.NewSimpleClientset(),
	}
}

func TestConfig_GetNodeFromLabelSelector(t *testing.T) {
	// This fixture has 3 nodes with following details
	// 1. Name: "node1" , set of labels on node : {"kubernetes.io/hostname": "node1"}
	// 2. Name: "node2" , set of labels on node : {"kubernetes.io/hostname": "node2"}
	// 3. Name: "node3" , set of labels on node : {"kubernetes.io/hostname": "node3"}
	fixture := NewFixture().WithKubeObjects(NewNodeList(3))

	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "[node1 exists] Select node with labels : {kubernetes.io/hostname:node1}",
			args: args{
				labels: map[string]string{
					"kubernetes.io/hostname": "node1",
				},
			},
			want:    "node1",
			wantErr: false,
		},

		{
			name: "[node2 exists] Select node with labels : {kubernetes.io/hostname:node2}",
			args: args{
				labels: map[string]string{
					"kubernetes.io/hostname": "node2",
				},
			},
			want:    "node2",
			wantErr: false,
		},

		{
			name: "[node2 exists] Select node with labels : {kubernetes.io/hostname:node2, dummy.io/dummy:dummy}",
			args: args{
				labels: map[string]string{
					"kubernetes.io/hostname": "node2",
					"dummy.io/dummy":         "dummy",
				},
			},
			want:    "",
			wantErr: true,
		},

		{
			name: "[node4 does not exist] Select node with labels : {kubernetes.io/hostname:node4}",
			args: args{
				labels: map[string]string{
					"kubernetes.io/hostname": "node4",
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &Config{
				kubeclientset: fixture.kubeclient,
			}
			got, err := ac.GetNodeFromLabelSelector(tt.args.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GetNodeFromLabelSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.GetNodeFromLabelSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func NewCSPIList() *cstor.CStorPoolInstanceList {
	cspcToCspi := CSPCToCSPI()
	newCspiList := &cstor.CStorPoolInstanceList{}
	for cspc, cspiList := range cspcToCspi {
		for _, cspi := range cspiList {
			newCSPI := &cstor.CStorPoolInstance{}
			newCSPI.Name = cspi
			newCSPI.Namespace = "openebs"
			labels := make(map[string]string)
			labels[types.CStorPoolClusterLabelKey] = cspc
			labels[types.HostNameLabelKey] = CSPIToNode()[cspi]
			newCSPI.Labels = labels
			newCspiList.Items = append(newCspiList.Items, *newCSPI)
		}
	}
	return newCspiList
}

func CSPIToNode() map[string]string {
	cspiToNode := make(map[string]string)
	cspiToNode["cspi-mulecspc-node1"] = "node1"
	cspiToNode["cspi-daocspc-node2"] = "node2"
	cspiToNode["cspi-litmuscspc-node2"] = "node2"
	cspiToNode["cspi-mulecspc-node3"] = "node3"
	return cspiToNode
}

func CSPCToCSPI() map[string][]string {
	cspcToCspi := make(map[string][]string)
	cspcToCspi["mulecspc"] = []string{"cspi-mulecspc-node1", "cspi-mulecspc-node3"}
	cspcToCspi["daocspc"] = []string{"cspi-daocspc-node2"}
	cspcToCspi["litmuscspc"] = []string{"cspi-litmuscspc-node2"}
	return cspcToCspi
}

func TestConfig_GetUsedNode(t *testing.T) {
	// This fixture has 3 nodes with following details
	// 1. Name: "node1" , set of labels on node : {"kubernetes.io/hostname": "node1"}
	// Pool Details ( i.e. CSPI that exists on this node) are as follows:
	// 1.1 Name of the CSPI : mulecspc-cspi-node1; Owner of CSPI : mulecspc ( mulecspc is a cspc object)

	// 2. Name: "node2" , set of labels on node : {"kubernetes.io/hostname": "node2"}
	// Pool Details ( i.e. CSPI that exists on this node) are as follows:
	// 2.1 Name of the CSPI : mulecspc-cspi-node2; Owner of CSPI : mulecspc ( mulecspc is a cspc object)

	// 3. Name: "node3" , set of labels on node : {"kubernetes.io/hostname": "node3"}

	fixture := NewFixture().WithKubeObjects(NewNodeList(3)).WithOpenEBSObjects(NewCSPIList())
	type fields struct {
		CSPC      *cstor.CStorPoolCluster
		Namespace string
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "Get used node for mulecspc",
			fields: fields{
				CSPC: &cstor.CStorPoolCluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "mulecspc",
					},
				},
				Namespace: "openebs",
			},
			want: map[string]bool{
				"node1": true,
				"node3": true,
			},
			wantErr: false,
		},

		{
			name: "Get used node for daocspc",
			fields: fields{
				CSPC: &cstor.CStorPoolCluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "daocspc",
					},
				},
				Namespace: "openebs",
			},
			want: map[string]bool{
				"node2": true,
			},
			wantErr: false,
		},

		{
			name: "Get used node for litmuscspc",
			fields: fields{
				CSPC: &cstor.CStorPoolCluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "litmuscspc",
					},
				},
				Namespace: "openebs",
			},
			want: map[string]bool{
				"node2": true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &Config{
				CSPC:          tt.fields.CSPC,
				Namespace:     tt.fields.Namespace,
				clientset:     fixture.openebsClient,
				kubeclientset: fixture.kubeclient,
			}
			got, err := ac.GetUsedNodes()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GetUsedNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.GetUsedNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBDListForNode(t *testing.T) {
	type args struct {
		pool cstor.PoolSpec
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "BD List for pool spec",
			args: args{
				pool: cstor.PoolSpec{
					DataRaidGroups: []cstor.RaidGroup{
						{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{{BlockDeviceName: "bd-1"}, {BlockDeviceName: "bd-2"}}},
						{CStorPoolInstanceBlockDevices: []cstor.CStorPoolInstanceBlockDevice{{BlockDeviceName: "bd-3"}, {BlockDeviceName: "bd-4"}}},
					},
				},
			},
			want: []string{"bd-1", "bd-2", "bd-3", "bd-4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBDListForNode(tt.args.pool); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBDListForNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_ClaimBD(t *testing.T) {
	fixture := NewFixture()
	type fields struct {
		Namespace string
		CSPC      *cstor.CStorPoolCluster
	}
	type args struct {
		bdObj openebsio.BlockDevice
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		isBdcCreated bool
		wantErr      bool
	}{
		{
			name: "Claim BD for BD bd-1",
			fields: fields{
				Namespace: "openebs",
				CSPC: &cstor.CStorPoolCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:      "mulecspc",
						Namespace: "openebs",
					},
				},
			},
			args: args{
				bdObj: openebsio.BlockDevice{
					ObjectMeta: v1.ObjectMeta{
						Name:      "bd-1",
						Namespace: "openebs",
					},
					Spec: openebsio.DeviceSpec{
						Capacity: openebsio.DeviceCapacity{
							Storage: 6000000,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &Config{
				CSPC:          tt.fields.CSPC,
				Namespace:     tt.fields.Namespace,
				clientset:     fixture.openebsClient,
				kubeclientset: fixture.kubeclient,
			}
			err := ac.ClaimBD(tt.args.bdObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.ClaimBD() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			isBdcCreated := false
			ob, _ := ac.clientset.OpenebsV1alpha1().BlockDeviceClaims(ac.Namespace).Get(context.TODO(), tt.args.bdObj.Name+string(tt.args.bdObj.UID), v1.GetOptions{})
			if ob != nil {
				isBdcCreated = true
			}
			if isBdcCreated != tt.isBdcCreated {
				t.Errorf("Config.ClaimBD() error = %v, wantErr %v", isBdcCreated, tt.isBdcCreated)
				return
			}
		})
	}
}

func Test_getAllowedTagMap(t *testing.T) {
	type args struct {
		cspcAnnotation map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]bool
	}{
		{
			name: "Test case #1",
			args: args{
				cspcAnnotation: map[string]string{types.OpenEBSAllowedBDTagKey: "fast,slow"},
			},
			want: map[string]bool{"fast": true, "slow": true},
		},

		{
			name: "Test case #2",
			args: args{
				cspcAnnotation: map[string]string{types.OpenEBSAllowedBDTagKey: "fast,slow"},
			},
			want: map[string]bool{"slow": true, "fast": true},
		},

		{
			name: "Test case #3 -- Nil Annotations",
			args: args{
				cspcAnnotation: nil,
			},
			want: map[string]bool{},
		},

		{
			name: "Test case #4 -- No BD tag Annotations",
			args: args{
				cspcAnnotation: map[string]string{"some-other-annotation-key": "awesome-openebs"},
			},
			want: map[string]bool{},
		},

		{
			name: "Test case #5 -- Improper format 1",
			args: args{
				cspcAnnotation: map[string]string{types.OpenEBSAllowedBDTagKey: ",fast,slow,,"},
			},
			want: map[string]bool{"fast": true, "slow": true},
		},

		{
			name: "Test case #6 -- Improper format 2",
			args: args{
				cspcAnnotation: map[string]string{types.OpenEBSAllowedBDTagKey: ",fast,slow"},
			},
			want: map[string]bool{"fast": true, "slow": true},
		},

		{
			name: "Test case #7 -- Improper format 2",
			args: args{
				cspcAnnotation: map[string]string{types.OpenEBSAllowedBDTagKey: ",fast,,slow"},
			},
			want: map[string]bool{"fast": true, "slow": true},
		},

		{
			name: "Test case #7 -- Improper format 2",
			args: args{
				cspcAnnotation: map[string]string{types.OpenEBSAllowedBDTagKey: "this is improper"},
			},
			want: map[string]bool{"this is improper": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAllowedTagMap(tt.args.cspcAnnotation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllowedTagMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
