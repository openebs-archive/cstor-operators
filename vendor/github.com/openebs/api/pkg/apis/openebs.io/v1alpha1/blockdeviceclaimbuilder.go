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

package v1alpha1

import (
	"github.com/openebs/api/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StoragePoolKindCSPC holds the value of CStorPoolCluster
	StoragePoolKindCSPC = "CStorPoolCluster"
	// APIVersion holds the value of OpenEBS version
	APIVersion = "openebs.io/v1alpha1"
)

func NewBlockDeviceClaim() *BlockDeviceClaim {
	return &BlockDeviceClaim{}
}

// WithName sets the Name field of BDC with provided value.
func (bdc *BlockDeviceClaim) WithName(name string) *BlockDeviceClaim {
	bdc.Name = name
	return bdc
}

// WithNamespace sets the Namespace field of BDC provided arguments
func (bdc *BlockDeviceClaim) WithNamespace(namespace string) *BlockDeviceClaim {
	bdc.Namespace = namespace
	return bdc
}

// WithAnnotationsNew sets the Annotations field of BDC with provided arguments
func (bdc *BlockDeviceClaim) WithAnnotationsNew(annotations map[string]string) *BlockDeviceClaim {
	bdc.Annotations = make(map[string]string)
	for key, value := range annotations {
		bdc.Annotations[key] = value
	}
	return bdc
}

// WithAnnotations appends or overwrites existing Annotations
// values of BDC with provided arguments
func (bdc *BlockDeviceClaim) WithAnnotations(annotations map[string]string) *BlockDeviceClaim {
	if bdc.Annotations == nil {
		return bdc.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		bdc.Annotations[key] = value
	}
	return bdc
}

// WithLabelsNew sets the Labels field of BDC with provided arguments
func (bdc *BlockDeviceClaim) WithLabelsNew(labels map[string]string) *BlockDeviceClaim {
	bdc.Labels = make(map[string]string)
	for key, value := range labels {
		bdc.Labels[key] = value
	}
	return bdc
}

// WithLabels appends or overwrites existing Labels
// values of BDC with provided arguments
func (bdc *BlockDeviceClaim) WithLabels(labels map[string]string) *BlockDeviceClaim {
	if bdc.Labels == nil {
		return bdc.WithLabelsNew(labels)
	}
	for key, value := range labels {
		bdc.Labels[key] = value
	}
	return bdc
}

// WithBlockDeviceName sets the BlockDeviceName field of BDC provided arguments
func (bdc *BlockDeviceClaim) WithBlockDeviceName(bdName string) *BlockDeviceClaim {
	bdc.Spec.BlockDeviceName = bdName
	return bdc
}

// WithDeviceType sets the DeviceType field of BDC provided arguments
func (bdc *BlockDeviceClaim) WithDeviceType(dType string) *BlockDeviceClaim {
	bdc.Spec.DeviceType = dType
	return bdc
}

// WithHostName sets the hostName field of BDC provided arguments
func (bdc *BlockDeviceClaim) WithHostName(hName string) *BlockDeviceClaim {
	bdc.Spec.BlockDeviceNodeAttributes.HostName = hName
	return bdc
}

// WithNodeName sets the node name field of BDC provided arguments
func (bdc *BlockDeviceClaim) WithNodeName(nName string) *BlockDeviceClaim {
	bdc.Spec.BlockDeviceNodeAttributes.NodeName = nName
	return bdc
}

// WithCapacity sets the Capacity field in BDC with provided arguments
func (bdc *BlockDeviceClaim) WithCapacity(capacity resource.Quantity) *BlockDeviceClaim {
	resourceList := corev1.ResourceList{
		corev1.ResourceName(ResourceStorage): capacity,
	}
	bdc.Spec.Resources.Requests = resourceList
	return bdc
}

// WithCSPCOwnerReference sets the OwnerReference field in BDC with required
//fields
func (bdc *BlockDeviceClaim) WithCSPCOwnerReference(reference metav1.OwnerReference) *BlockDeviceClaim {
	bdc.OwnerReferences = append(bdc.OwnerReferences, reference)
	return bdc
}

// WithFinalizer sets the finalizer field in the BDC
func (bdc *BlockDeviceClaim) WithFinalizer(finalizers ...string) *BlockDeviceClaim {
	bdc.Finalizers = append(bdc.Finalizers, finalizers...)
	return bdc
}

// WithBlockVolumeMode sets the volumeMode as volumeModeBlock,
// if persistentVolumeMode is set to "Block"
func (bdc *BlockDeviceClaim) WithBlockVolumeMode(mode corev1.PersistentVolumeMode) *BlockDeviceClaim {
	if mode == corev1.PersistentVolumeBlock {
		bdc.Spec.Details.BlockVolumeMode = VolumeModeBlock
	}
	return bdc
}

// RemoveFinalizer removes the given finalizer from the object.
func (bdc *BlockDeviceClaim) RemoveFinalizer(finalizer string) {
	bdc.Finalizers = util.RemoveString(bdc.Finalizers, finalizer)
}

// GetBlockDeviceClaimFromBDName return block device claim if claim exists for
// provided blockdevice name in claim list else return error
func (bdcl *BlockDeviceClaimList) GetBlockDeviceClaimFromBDName(
	bdName string) (*BlockDeviceClaim, error) {
	for _, bdc := range bdcl.Items {
		// pin it
		bdc := bdc
		if bdc.Spec.BlockDeviceName == bdName {
			return &bdc, nil
		}
	}
	return nil, errors.Errorf("claim doesn't exist for blockdevice %s", bdName)
}
