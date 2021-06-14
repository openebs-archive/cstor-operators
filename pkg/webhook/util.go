// Copyright 2020 The OpenEBS Authors
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

package webhook

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	unit = 1024
)

// ByteCount converts bytes into corresponding unit
func ByteCount(b uint64) string {
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, index := uint64(unit), 0
	for val := b / unit; val >= unit; val /= unit {
		div *= unit
		index++
	}
	return fmt.Sprintf("%d%c",
		uint64(b)/uint64(div), "KMGTPE"[index])
}

// if currentversion is less `<` then new version (return true in case of equal version)
// TODO use version lib to properly handle versions https://github.com/hashicorp/go-version
func IsCurrentLessThanNewVersion(old, new string) bool {
	oldVersions := strings.Split(strings.Split(old, "-")[0], ".")
	newVersions := strings.Split(strings.Split(new, "-")[0], ".")
	for i := 0; i < len(oldVersions); i++ {
		oldVersion, _ := strconv.Atoi(oldVersions[i])
		newVersion, _ := strconv.Atoi(newVersions[i])
		if oldVersion == newVersion {
			continue
		}
		return oldVersion < newVersion
	}
	return false
}
