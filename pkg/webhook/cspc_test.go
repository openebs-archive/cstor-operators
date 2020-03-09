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
	"testing"

	cstor "github.com/openebs/api/pkg/apis/cstor/v1"
)

func TestValidateSpecChanges(t *testing.T) {
	tests := map[string]struct {
		commonPoolSpecs *poolspecs
		bdr             *BlockDeviceReplacement
		expectedOutput  bool
	}{
		"No change in poolSpecs": {
			commonPoolSpecs: &poolspecs{
				oldSpec: []cstor.PoolSpec{
					cstor.PoolSpec{
						DataRaidGroups: []cstor.RaidGroup{
							cstor.RaidGroup{
								BlockDevices: []cstor.CStorPoolClusterBlockDevice{
									cstor.CStorPoolClusterBlockDevice{
										BlockDeviceName: "bd1",
									},
									cstor.CStorPoolClusterBlockDevice{
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
								BlockDevices: []cstor.CStorPoolClusterBlockDevice{
									cstor.CStorPoolClusterBlockDevice{
										BlockDeviceName: "bd1",
									},
									cstor.CStorPoolClusterBlockDevice{
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
			bdr: &BlockDeviceReplacement{
				OldCSPC: &cstor.CStorPoolCluster{},
				NewCSPC: &cstor.CStorPoolCluster{},
			},
			expectedOutput: true,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			isValid, _ := ValidateSpecChanges(test.commonPoolSpecs, test.bdr)
			if isValid != test.expectedOutput {
				t.Errorf("test: %s failed expected output %t but got %t", name, isValid, test.expectedOutput)
			}
		})
	}
}
