package snapshottest

import (
	v1proto "github.com/openebs/api/pkg/proto"
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
