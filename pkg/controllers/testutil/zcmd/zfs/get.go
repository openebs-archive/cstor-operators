/*
Copyright 2020 The OpenEBS Authors.

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
package zfs

import (
	"fmt"
	"strings"
)

// GetProperty mocks the zfs get command and returns the error based on the output
// TODO: Having GetProperty as a method will help to return desired value
// set for ZFS properties by the TestCase(As of now we are not setting
// in test configuration)
func (volumeMocker *VolumeMocker) GetProperty(cmd string) ([]byte, error) {
	var isProperty bool
	var output string

	values := strings.Split(cmd, " ")
	for _, val := range values {
		if val == " " {
			continue
		}
		if val == "value," {
			isProperty = true
		}
		if isProperty && strings.Contains(val, "compression") {
			output = addToOutput(output, "lz4")
		}
	}
	return []byte(output), nil
}

func addToOutput(output, value string) string {
	if output == "" {
		output = value
	} else {
		output = fmt.Sprintf("%s\n%s", output, value)
	}
	return output
}
