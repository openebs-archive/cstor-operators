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

package maps

import (
	"strconv"
	"strings"
)

// IsSubset compares two maps to determine if one of them is fully contained in the other.
func IsSubset(toCheck, fullSet map[string]string) bool {
	if len(toCheck) > len(fullSet) {
		return false
	}

	for k, v := range toCheck {
		if currValue, ok := fullSet[k]; !ok || currValue != v {
			return false
		}
	}

	return true
}

// Merge merges source into destination, overwriting existing values if necessary.
func Merge(dest, src map[string]string) map[string]string {
	if dest == nil {
		if src == nil {
			return nil
		}
		dest = make(map[string]string, len(src))
	}

	for k, v := range src {
		dest[k] = v
	}

	return dest
}

// MergePreservingExistingKeys merges source into destination while skipping any keys that exist in the destination.
func MergePreservingExistingKeys(dest, src map[string]string) map[string]string {
	if dest == nil {
		if src == nil {
			return nil
		}
		dest = make(map[string]string, len(src))
	}

	for k, v := range src {
		if _, exists := dest[k]; !exists {
			dest[k] = v
		}
	}

	return dest
}

// ContainsKeys determines if a set of label (keys) are present in a map of labels (keys and values).
func ContainsKeys(m map[string]string, labels ...string) bool {
	for _, label := range labels {
		if _, exists := m[label]; !exists {
			return false
		}
	}
	return true
}

// IsCurrentLessThanNewVersion compares current and new version and returns true
// if currentversion is less `<` then new version (return true in case of equal version)
// TODO use version lib to properly handle versions https://github.com/hashicorp/go-version
func IsCurrentLessThanNewVersion(old, new string) bool {
	oldVersions := strings.Split(strings.Split(old, "-")[0], ".")
	newVersions := strings.Split(strings.Split(new, "-")[0], ".")
	for i := 0; i < len(oldVersions); i++ {
		oldVersion, _ := strconv.Atoi(oldVersions[i])
		newVersion, _ := strconv.Atoi(newVersions[i])
		if oldVersion > newVersion {
			return false
		}
	}
	return true
}
