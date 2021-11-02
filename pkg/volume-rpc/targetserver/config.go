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
package targetserver

import (
	"context"
	"fmt"

	apis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	clientset "github.com/openebs/api/v3/pkg/client/clientset/versioned"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

//CVReplicationDetails enables to update RF,CF and
//known replicas into etcd
type CVReplicationDetails struct {
	VolumeName        string `json:"volumeName"`
	ReplicationFactor int    `json:"replicationFactor"`
	ConsistencyFactor int    `json:"consistencyFactor"`
	ReplicaID         string `json:"replicaId"`
	ReplicaGUID       string `json:"replicaZvolGuid"`
}

// BuildConfigData builds data based on the CVReplicationDetails
func (csr *CVReplicationDetails) BuildConfigData() map[string]string {
	data := map[string]string{}
	// Since we know what to update in istgt.conf file so constructing
	// key and value pairs
	// key represents what kind of configurations
	// value represents corresponding value for that key
	// TODO: Improve below code by exploring different options
	key := fmt.Sprintf("  ReplicationFactor")
	value := fmt.Sprintf("  ReplicationFactor %d", csr.ReplicationFactor)
	data[key] = value
	key = fmt.Sprintf("  ConsistencyFactor")
	value = fmt.Sprintf("  ConsistencyFactor %d", csr.ConsistencyFactor)
	data[key] = value
	key = fmt.Sprintf("  Replica %s", csr.ReplicaID)
	value = fmt.Sprintf("  Replica %s %s", csr.ReplicaID, csr.ReplicaGUID)
	data[key] = value
	return data
}

// UpdateConfig updates target configuration file by building data
func (csr *CVReplicationDetails) UpdateConfig() error {
	configData := csr.BuildConfigData()
	fileOperator := util.RealFileOperator{}
	types.ConfFileMutex.Lock()
	err := fileOperator.UpdateOrAppendMultipleLines(types.IstgtConfPath, configData, 0644)
	types.ConfFileMutex.Unlock()
	if err == nil {
		klog.V(4).Infof("Successfully updated istgt conf file with %v details\n", csr)
	}
	return err
}

// Validate verifies whether CStorReplication data read on wire is valid or not
func (csr *CVReplicationDetails) Validate() error {
	if csr.VolumeName == "" {
		return errors.Errorf("volume name can not be empty")
	}
	if csr.ReplicaID == "" {
		return errors.Errorf("replicaKey can not be empty to perform "+
			"volume %s update", csr.VolumeName)
	}
	if csr.ReplicaGUID == "" {
		return errors.Errorf("replicaKey can not be empty to perform "+
			"volume %s update", csr.VolumeName)
	}
	if csr.ReplicationFactor == 0 {
		return errors.Errorf("replication factor can't be %d",
			csr.ReplicationFactor)
	}
	if csr.ConsistencyFactor == 0 {
		return errors.Errorf("consistencyFactor factor can't be %d",
			csr.ReplicationFactor)
	}
	return nil
}

// UpdateCVWithReplicationDetails updates the cstorvolume with known replicas
// and updated replication details
func (csr *CVReplicationDetails) UpdateCVWithReplicationDetails(openebsClient clientset.Interface) error {
	if openebsClient == nil {
		return errors.Errorf("failed to update replication details error: empty openebsClient")
	}
	err := csr.Validate()
	if err != nil {
		return errors.Wrapf(err, "validate errors")
	}
	cv, err := openebsClient.CstorV1().CStorVolumes(util.GetNamespace()).Get(context.TODO(), csr.VolumeName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get cstorvolume")
	}
	_, oldReplica := cv.Spec.ReplicaDetails.KnownReplicas[apis.ReplicaID(csr.ReplicaID)]
	if !oldReplica &&
		len(cv.Spec.ReplicaDetails.KnownReplicas) >= cv.Spec.DesiredReplicationFactor {
		return errors.Errorf("can not update cstorvolume %s known replica"+
			" count %d is greater than or equal to desired replication factor %d",
			cv.Name, len(cv.Spec.ReplicaDetails.KnownReplicas),
			cv.Spec.DesiredReplicationFactor,
		)
	}
	if cv.Spec.ReplicationFactor > csr.ReplicationFactor {
		return errors.Errorf("requested replication factor {%d}"+
			" can not be smaller than existing replication factor {%d}",
			csr.ReplicationFactor, cv.Spec.ReplicationFactor,
		)
	}
	if cv.Spec.ConsistencyFactor > csr.ConsistencyFactor {
		return errors.Errorf("requested consistencyFactor factor {%d}"+
			" can not be smaller than existing consistencyFactor factor {%d}",
			csr.ReplicationFactor, cv.Spec.ConsistencyFactor,
		)
	}
	cv.Spec.ReplicationFactor = csr.ReplicationFactor
	cv.Spec.ConsistencyFactor = csr.ConsistencyFactor
	if cv.Spec.ReplicaDetails.KnownReplicas == nil {
		cv.Spec.ReplicaDetails.KnownReplicas = map[apis.ReplicaID]string{}
	}
	if cv.Status.ReplicaDetails.KnownReplicas == nil {
		cv.Status.ReplicaDetails.KnownReplicas = map[apis.ReplicaID]string{}
	}
	// Updating both spec and status known replica list
	cv.Spec.ReplicaDetails.KnownReplicas[apis.ReplicaID(csr.ReplicaID)] = csr.ReplicaGUID
	cv.Status.ReplicaDetails.KnownReplicas[apis.ReplicaID(csr.ReplicaID)] = csr.ReplicaGUID
	_, err = openebsClient.CstorV1().CStorVolumes(util.GetNamespace()).Update(context.TODO(), cv, metav1.UpdateOptions{})
	if err == nil {
		klog.Infof("Successfully updated %s volume with following replication "+
			"information replication fator: from %d to %d, consistencyFactor from "+
			"%d to %d and known replica list with replicaId %s and GUID %v",
			cv.Name, cv.Spec.ReplicationFactor, csr.ReplicationFactor,
			cv.Spec.ConsistencyFactor, csr.ConsistencyFactor, csr.ReplicaID,
			csr.ReplicaGUID,
		)
		err = csr.UpdateConfig()
	}
	return err
}
