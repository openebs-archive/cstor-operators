/*
Copyright 2019 The OpenEBS Authors.

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

package vlist

// PredicateFunc defines data-type for validation function
type PredicateFunc func(*VolumeList) bool

// IsProplistSet method check if the Proplist field of VolumeList object is set.
func IsProplistSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return len(v.PropList) != 0
	}
}

// IsFieldListSet method check if the FieldList field of VolumeList object is set.
func IsFieldListSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return len(v.FieldList) != 0
	}
}

// IsScriptedModeSet method check if the IsScriptedMode field of VolumeList object is set.
func IsScriptedModeSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return v.IsScriptedMode
	}
}

// IsParsableModeSet method check if the IsParsableMode field of VolumeList object is set.
func IsParsableModeSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return v.IsParsableMode
	}
}

// IsDatasetSet method check if the Dataset field of VolumeStats object is set.
func IsDatasetSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return len(v.Dataset) != 0
	}
}

// IsCommandSet method check if the Command field of VolumeStats object is set.
func IsCommandSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return len(v.Command) != 0
	}
}

// IsExecutorSet method check if the Executor field of VolumeStats object is set.
func IsExecutorSet() PredicateFunc {
	return func(v *VolumeList) bool {
		return v.Executor != nil
	}
}
