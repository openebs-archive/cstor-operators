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
package version

import (
	"strconv"
	"strings"
)

var (
	minCurrentVersion   = "1.10.0"
	validDesiredVersion = strings.Split(GetVersion(), "-")[0]
	// these are the versions used in various pipelines for ci testing
	exceptions = []string{"master", "develop"}
)

// IsCurrentVersionValid verifies if the  current version is valid or not
func IsCurrentVersionValid(v string) bool {
	currentVersion := strings.Split(v, "-")[0]
	return CanCurrentVersionBeUpgraded(currentVersion)
}

// IsDesiredVersionValid verifies the desired version is valid or not
func IsDesiredVersionValid(v string) bool {
	desiredVersion := strings.Split(v, "-")[0]
	return validDesiredVersion == desiredVersion
}

// CanCurrentVersionBeUpgraded determines whether the current version
// is within the range of minCurrentVersion and validDesiredVersion
func CanCurrentVersionBeUpgraded(version string) bool {
	return IsOldLessThanOrEqualNewVersion(minCurrentVersion, version) &&
		IsOldLessThanOrEqualNewVersion(version, validDesiredVersion)
}

// IsOldLessThanOrEqualNewVersion compares old and new version and returns true
// if old version is less `<` or equal then new version
func IsOldLessThanOrEqualNewVersion(old, new string) bool {
	oldVersions := strings.Split(strings.Split(old, "-")[0], ".")
	newVersions := strings.Split(strings.Split(new, "-")[0], ".")
	for _, exception := range exceptions {
		if newVersions[0] == exception {
			return true
		}
	}
	for i := 0; i < len(oldVersions); i++ {
		oldVersion, _ := strconv.Atoi(oldVersions[i])
		newVersion, _ := strconv.Atoi(newVersions[i])
		if oldVersion == newVersion {
			continue
		}
		return oldVersion < newVersion
	}
	return true
}
