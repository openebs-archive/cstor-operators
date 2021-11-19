// Copyright © 2020 The OpenEBS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapshot

import (
	"context"

	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	v1proto "github.com/openebs/api/v3/pkg/proto"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// Snapshot holds the information required to perform snapshot related operations
type Snapshot struct {
	VolumeName   string
	SnapshotName string
	Namespace    string
	SnapClient   Snapshotter
}

// CreateSnapshot creates snapshot for provided CStor Volume
// TODO: Think better name something like CreateSnapshotByFetchingIP
func (s *Snapshot) CreateSnapshot(clientset clientset.Interface) (*v1proto.VolumeSnapCreateResponse, error) {
	// If snapshot client is not specified then return error
	if s.SnapClient == nil {
		return nil, errors.Errorf("snapshot client is not initilized to perform snapshot operations")
	}
	// Fetch IPAddress of snapshot server
	ipAddr, err := getVolumeIP(s.VolumeName, s.Namespace, clientset)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CStor volumeIP")
	}

	klog.Infof("Creating snapshot %s for volume %q", s.SnapshotName, s.VolumeName)

	snapResp, err := s.SnapClient.CreateSnapshot(ipAddr, s.VolumeName, s.SnapshotName)
	if err != nil {
		klog.Errorf("Failed to create snapshot:%s error '%s'", s.SnapshotName, err.Error())
		return nil, errors.Wrapf(err, "failed to create snapshot: %s for volume: %s", s.SnapshotName, s.VolumeName)
	}
	return snapResp, nil
}

// DeleteSnapshot deletes snapshot for provided volume
func (s *Snapshot) DeleteSnapshot(clientset clientset.Interface) (*v1proto.VolumeSnapDeleteResponse, error) {
	// If snapshot client is not specified then return error
	if s.SnapClient == nil {
		return nil, errors.Errorf("snapshot client is not initilized to perform snapshot operations")
	}
	// Fetch IPAddress of snapshot server
	ipAddr, err := getVolumeIP(s.VolumeName, s.Namespace, clientset)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CStor volumeIP")
	}

	klog.Infof("Deleting snapshot %s for volume %q", s.SnapshotName, s.VolumeName)

	snapResp, err := s.SnapClient.DestroySnapshot(ipAddr, s.VolumeName, s.SnapshotName)
	if err != nil {
		klog.Errorf("Failed to delete snapshot:%s error '%s'", s.SnapshotName, err.Error())
		return nil, errors.Wrapf(err, "failed to delete snapshot: %s for volume: %s", s.SnapshotName, s.VolumeName)
	}
	return snapResp, nil
}

// getVolumeIP fetches the cstor target service IP Address
func getVolumeIP(volumeName, namespace string, clientset clientset.Interface) (string, error) {
	// Fetch the corresponding cstorvolume
	cstorvolume, err := clientset.CstorV1().CStorVolumes(namespace).
		Get(context.TODO(), volumeName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return cstorvolume.Spec.TargetIP, nil
}
