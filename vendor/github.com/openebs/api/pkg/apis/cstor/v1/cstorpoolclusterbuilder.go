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

package v1

import (
	"github.com/openebs/api/pkg/util"
)

func NewCStorPoolCluster() *CStorPoolCluster {
	return &CStorPoolCluster{}
}

// WithName sets the Name field of cspc with provided value.
func (cspc *CStorPoolCluster) WithName(name string) *CStorPoolCluster {
	cspc.Name = name
	return cspc
}

// WithNamespace sets the Namespace field of cspc provided arguments
func (cspc *CStorPoolCluster) WithNamespace(namespace string) *CStorPoolCluster {
	cspc.Namespace = namespace
	return cspc
}

// WithAnnotationsNew sets the Annotations field of cspc with provided arguments
func (cspc *CStorPoolCluster) WithAnnotationsNew(annotations map[string]string) *CStorPoolCluster {
	cspc.Annotations = make(map[string]string)
	for key, value := range annotations {
		cspc.Annotations[key] = value
	}
	return cspc
}

// WithAnnotations appends or overwrites existing Annotations
// values of cspc with provided arguments
func (cspc *CStorPoolCluster) WithAnnotations(annotations map[string]string) *CStorPoolCluster {

	if cspc.Annotations == nil {
		return cspc.WithAnnotationsNew(annotations)
	}
	for key, value := range annotations {
		cspc.Annotations[key] = value
	}
	return cspc
}

// WithLabelsNew sets the Labels field of cspc with provided arguments
func (cspc *CStorPoolCluster) WithLabelsNew(labels map[string]string) *CStorPoolCluster {
	cspc.Labels = make(map[string]string)
	for key, value := range labels {
		cspc.Labels[key] = value
	}
	return cspc
}

// WithLabels appends or overwrites existing Labels
// values of cspc with provided arguments
func (cspc *CStorPoolCluster) WithLabels(labels map[string]string) *CStorPoolCluster {
	if cspc.Labels == nil {
		return cspc.WithLabelsNew(labels)
	}
	for key, value := range labels {
		cspc.Labels[key] = value
	}
	return cspc
}

// WithFinalizer sets the finalizer field in the BDC
func (cspc *CStorPoolCluster) WithFinalizer(finalizers ...string) *CStorPoolCluster {
	cspc.Finalizers = append(cspc.Finalizers, finalizers...)
	return cspc
}

// HasFinalizer returns true if the provided finalizer is present on the object.
func (cspc *CStorPoolCluster) HasFinalizer(finalizer string) bool {
	finalizersList := cspc.GetFinalizers()
	return util.ContainsString(finalizersList, finalizer)
}

// RemoveFinalizer removes the given finalizer from the object.
func (cspc *CStorPoolCluster) RemoveFinalizer(finalizer string) {
	cspc.Finalizers = util.RemoveString(cspc.Finalizers, finalizer)
}

// HasAnnotation return true if provided annotation
// key and value are present on the object.
func (cspc *CStorPoolCluster) HasAnnotation(key, value string) bool {
	val, ok := cspc.GetAnnotations()[key]
	if ok {
		return val == value
	}
	return false
}

// HasLabel returns true if provided label
// key and value are present on the object.
func (cspc *CStorPoolCluster) HasLabel(key, value string) bool {
	val, ok := cspc.GetLabels()[key]
	if ok {
		return val == value
	}
	return false
}

// GetBlockDevices returns list of blockdevice names exist in the raid group
func (rg RaidGroup) GetBlockDevices() []string {
	var bdNames []string
	for _, cspcBD := range rg.CStorPoolInstanceBlockDevices {
		bdNames = append(bdNames, cspcBD.BlockDeviceName)
	}
	return bdNames
}
