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

package vstats

// PredicateFunc defines data-type for validation function
type PredicateFunc func(*VolumeStats) bool

// IsProplistSet method check if the Property field of VolumeStats object is set.
func IsProplistSet() PredicateFunc {
	return func(v *VolumeStats) bool {
		return len(v.Proplist) != 0
	}
}

// IsDatasetSet method check if the Dataset field of VolumeStats object is set.
func IsDatasetSet() PredicateFunc {
	return func(v *VolumeStats) bool {
		return len(v.Dataset) != 0
	}
}

// IsCommandSet method check if the Command field of VolumeStats object is set.
func IsCommandSet() PredicateFunc {
	return func(v *VolumeStats) bool {
		return len(v.Command) != 0
	}
}

// IsExecutorSet method check if the Executor field of VolumeStats object is set.
func IsExecutorSet() PredicateFunc {
	return func(v *VolumeStats) bool {
		return v.Executor != nil
	}
}
