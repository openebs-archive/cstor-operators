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
	"strings"
)

var (
	validCurrentVersions = map[string]bool{
		"1.10.0": true, "1.11.0": true, "1.12.0": true,
		"2.0.0": true,
	}
	validDesiredVersion = strings.Split(GetVersion(), "-")[0]
)

// IsCurrentVersionValid verifies if the  current version is valid or not
func IsCurrentVersionValid(v string) bool {
	currentVersion := strings.Split(v, "-")[0]
	return validCurrentVersions[currentVersion]
}

// IsDesiredVersionValid verifies the desired version is valid or not
func IsDesiredVersionValid(v string) bool {
	desiredVersion := strings.Split(v, "-")[0]
	return validDesiredVersion == desiredVersion
}
