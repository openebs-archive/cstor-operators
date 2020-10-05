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

package core

import (
	corev1 "k8s.io/api/core/v1"
)

// Volume is a wrapper over named volume api object, used
// within Pods. It provides build, validations and other common
// logic to be used by various feature specific callers.
type Volume struct {
	*corev1.Volume
}

// NewVolume returns a new instance of volume
func NewVolume() *Volume {
	return &Volume{
		&corev1.Volume{},
	}
}

// WithName sets the Name field of Volume with provided value.
func (v *Volume) WithName(name string) *Volume {
	v.Name = name
	return v
}

// WithHostDirectory sets the VolumeSource field of Volume with provided hostpath
// as type directory.
func (v *Volume) WithHostDirectory(path string) *Volume {
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: path,
		},
	}

	v.VolumeSource = volumeSource
	return v
}

// WithHostPathAndType sets the VolumeSource field of Volume with provided
// hostpath as directory path and type as directory type
func (v *Volume) WithHostPathAndType(dirpath string, dirtype *corev1.HostPathType) *Volume {
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: dirpath,
			Type: dirtype,
		},
	}

	v.VolumeSource = volumeSource
	return v
}

// WithPVCSource sets the Volume field of Volume with provided pvc
func (v *Volume) WithPVCSource(pvcName string) *Volume {
	volumeSource := corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pvcName,
		},
	}
	v.VolumeSource = volumeSource
	return v
}

// WithEmptyDir sets the EmptyDir field of the Volume with provided dir
func (v *Volume) WithEmptyDir(dir *corev1.EmptyDirVolumeSource) *Volume {
	v.EmptyDir = dir
	return v
}

func (v *Volume) Build() *corev1.Volume {
	return v.Volume
}
