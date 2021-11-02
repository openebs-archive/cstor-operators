/*
Copyright 2021 The OpenEBS Authors

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

package core

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// GetImagePullSecrets  parses and transforms the
// string to corev1.LocalObjectReference.
// multiple secrets are separated by commas
func GetImagePullSecrets(s string) []corev1.LocalObjectReference {
	s = strings.TrimSpace(s)
	list := make([]corev1.LocalObjectReference, 0)
	if len(s) == 0 {
		return list
	}
	arr := strings.Split(s, ",")
	for _, item := range arr {
		if len(item) > 0 {
			l := corev1.LocalObjectReference{Name: strings.TrimSpace(item)}
			list = append(list, l)
		}
	}
	return list
}
