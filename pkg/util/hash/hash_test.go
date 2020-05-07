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

package hash

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHashList(t *testing.T) {
	fakeTestList := map[string][]string{
		"list1": []string{},
		"list2": []string{"list-1", "list-2", "list-3"},
		"list3": []string{"list-4", "list-5", "list-6"},
	}

	for _, test := range fakeTestList {
		fakeHash, err := Hash(test)
		if err != nil {
			t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", fakeHash, err)
		}
	}
	fakeHash, err := Hash(fakeTestList)
	if err != nil {
		t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", fakeHash, err)
	}
}

func TestHashStruct(t *testing.T) {
	fakeStruct := map[string]struct {
		name string
		list []string
	}{
		"struct1": {
			name: "",
			list: []string{},
		},
		"struct2": {
			name: "abcdefgh",
			list: []string{"abc-1", "abc-2", "abc-3"},
		},
		"struct3": {
			name: "jklmnop",
			list: []string{"abcde", "abcdf", "hash"},
		},
	}
	for _, test := range fakeStruct {
		fakeHash, err := Hash(test)
		if err != nil {
			t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", fakeHash, err)
		}
	}
	fakeHash, err := Hash(fakeStruct)
	if err != nil {
		t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", fakeHash, err)
	}
}

func TestHashInnerStruct(t *testing.T) {
	type fakeStruct struct {
		name string
		list []string
	}
	fakeComplexStruct := map[int]struct {
		innerStructNum int
		innerStruct    struct {
			name string
			list []string
		}
	}{
		1: {
			innerStructNum: 1,
			innerStruct: fakeStruct{
				name: "",
				list: []string{},
			},
		},
		2: {
			innerStructNum: 2,
			innerStruct: fakeStruct{
				name: "hashInnerStruct",
				list: []string{"hash1", "hash2", "hash3", "hash4"},
			},
		},
	}
	for _, test := range fakeComplexStruct {
		fakeHash, err := Hash(test)
		if err != nil {
			t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", fakeHash, err)
		}
	}
	fakeHash, err := Hash(fakeComplexStruct)
	if err != nil {
		t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", fakeHash, err)
	}
}

func TestHashOutput(t *testing.T) {
	list1 := []string{"one", "three", "two", "four"}
	list2 := []string{"four", "one", "three", "two"}
	sort.Strings(list1)
	sort.Strings(list2)
	hash1, err := Hash(list1)
	if err != nil {
		t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", hash1, err)
	}
	hash2, err := Hash(list2)
	if err != nil {
		t.Errorf("Failed to calculate the hash expected string but got: '%s' Error: '%v'", hash2, err)
	}
	if hash1 != hash2 {
		t.Errorf("hash value didn't matched for the same list")
	}
}

func TestHashObject(t *testing.T) {
	// nil objects hash the same
	require.Equal(t, HashObject(nil), HashObject(nil))

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
			Labels: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container",
					Env: []corev1.EnvVar{
						{
							Name:  "var1",
							Value: "value1",
						},
					},
				},
			},
		},
	}
	samePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
			Labels: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container",
					Env: []corev1.EnvVar{
						{
							Name:  "var1",
							Value: "value1",
						},
					},
				},
			},
		},
	}

	// hashes are consistent
	hash := HashObject(pod)
	// same object
	require.Equal(t, hash, HashObject(pod))
	// different object but same content
	require.Equal(t, hash, HashObject(samePod))

	// /!\ hashing an object and its pointer lead to different values
	require.NotEqual(t, hash, HashObject(&pod))

	// hashes ignore different pointer addresses
	userID := int64(123)
	securityContext1 := corev1.PodSecurityContext{RunAsUser: &userID}
	securityContext2 := corev1.PodSecurityContext{RunAsUser: &userID}
	pod.Spec.SecurityContext = &securityContext1
	hash = HashObject(pod)
	pod.Spec.SecurityContext = &securityContext2
	require.Equal(t, hash, HashObject(pod))

	// different hash on any object modification
	pod.Labels["c"] = "newvalue"
	require.NotEqual(t, hash, HashObject(pod))
}

func TestAddTemplateHashLabel(t *testing.T) {
	spec := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "container",
				Env: []corev1.EnvVar{
					{
						Name:  "var1",
						Value: "value1",
					},
				},
			},
		},
	}
	labels := map[string]string{
		"a": "b",
		"c": "d",
	}
	expected := map[string]string{
		"a":                   "b",
		"c":                   "d",
		TemplateHashLabelName: HashObject(spec),
	}
	require.Equal(t, expected, SetTemplateHashLabel(labels, spec))
}
