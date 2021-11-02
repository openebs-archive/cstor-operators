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
package snapshottest

import (
	v1proto "github.com/openebs/api/v3/pkg/proto"
	"github.com/pkg/errors"
)

// FakeSnapshotter is used to mock the snapshot operations
type FakeSnapshotter struct {
	ShouldReturnFakeError bool
}

// CreateSnapshot mocks snapshot create operation
func (fs *FakeSnapshotter) CreateSnapshot(ip, volName, snapName string) (*v1proto.VolumeSnapCreateResponse, error) {
	if fs.ShouldReturnFakeError {
		return nil, errors.Errorf("injected fake errors during snapshot create operation")
	}
	return &v1proto.VolumeSnapCreateResponse{}, nil
}

//DestroySnapshot mocks snapshot delete operation
func (fs *FakeSnapshotter) DestroySnapshot(ip, volName, snapName string) (*v1proto.VolumeSnapDeleteResponse, error) {
	if fs.ShouldReturnFakeError {
		return nil, errors.Errorf("injected fake errors during snapshot delete operation")
	}
	return &v1proto.VolumeSnapDeleteResponse{}, nil
}
