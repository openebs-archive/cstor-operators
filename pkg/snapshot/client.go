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
	"encoding/json"
	"fmt"
	"net"
	"strings"

	v1proto "github.com/openebs/api/v3/pkg/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// constants
const (
	VolumeGrpcListenPort = 7777
	ProtocolVersion      = 1
)

//CommandStatus is the response from istgt for control commands
type CommandStatus struct {
	Response string `json:"response"`
}

// Snapshotter is used to perform snapshot operations on given volume
type Snapshotter interface {
	CreateSnapshot(ip, volumeName, snapName string) (*v1proto.VolumeSnapCreateResponse, error)
	DestroySnapshot(ip, volumeName, snapName string) (*v1proto.VolumeSnapDeleteResponse, error)
}

// SnapClient is used to perform real snap create and snap delete commands
type SnapClient struct{}

//CreateSnapshot creates snapshot by executing gRPC call
func (s *SnapClient) CreateSnapshot(ip, volName, snapName string) (*v1proto.VolumeSnapCreateResponse, error) {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(net.JoinHostPort(ip, fmt.Sprintf("%d", VolumeGrpcListenPort)), grpc.WithInsecure())
	if err != nil {
		return nil, errors.Errorf("Unable to dial gRPC server on port %d error : %s", VolumeGrpcListenPort, err)
	}
	defer conn.Close()

	c := v1proto.NewRunSnapCommandClient(conn)
	response, err := c.RunVolumeSnapCreateCommand(context.Background(),
		&v1proto.VolumeSnapCreateRequest{
			Version:  ProtocolVersion,
			Volume:   volName,
			Snapname: snapName,
		})

	if err != nil {
		return nil, errors.Errorf("Error when calling RunVolumeSnapCreateCommand: %s", err)
	}

	if response != nil {
		var responseStatus CommandStatus
		json.Unmarshal(response.Status, &responseStatus)
		if strings.Contains(responseStatus.Response, "ERR") {
			return nil, errors.Errorf("Snapshot create failed with error : %v", responseStatus.Response)
		}
	}

	return response, nil
}

//DestroySnapshot destroys snapshots by executing gRPC calls
func (s *SnapClient) DestroySnapshot(ip, volName, snapName string) (*v1proto.VolumeSnapDeleteResponse, error) {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(net.JoinHostPort(ip, fmt.Sprintf("%d", VolumeGrpcListenPort)), grpc.WithInsecure())
	if err != nil {
		return nil, errors.Errorf("Unable to dial gRPC server on port error : %s", err)
	}
	defer conn.Close()

	c := v1proto.NewRunSnapCommandClient(conn)
	response, err := c.RunVolumeSnapDeleteCommand(context.Background(),
		&v1proto.VolumeSnapDeleteRequest{
			Version:  ProtocolVersion,
			Volume:   volName,
			Snapname: snapName,
		})

	if err != nil {
		return nil, errors.Errorf("Error when calling RunVolumeSnapDeleteCommand: %s", err)
	}

	if response != nil {
		var responseStatus CommandStatus
		json.Unmarshal(response.Status, &responseStatus)
		if strings.Contains(responseStatus.Response, "ERR") {
			return nil, errors.Errorf("Snapshot deletion failed with error : %v", responseStatus.Response)
		}
	}
	return response, nil
}
