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
		if isProperty {
			output = addToOutput(output, getPropertyValues(val))
		}
	}
	return []byte(output), nil
}

// getPropertyValues returns the values for quaried properties
func getPropertyValues(command string) string {
	var values string
	// If command is to get used space in dataset
	if strings.Contains(command, "used") {
		values = addToOutput(values, "69.5K")
	}
	// If command is to get available space in dataset
	if strings.Contains(command, "available") {
		values = addToOutput(values, "9.94G")
	}
	// If command is to get logicalused space in dataset
	if strings.Contains(command, "logicalused") {
		values = addToOutput(values, "70K")
	}
	// If command is to get compression value
	if strings.Contains(command, "compression") {
		values = addToOutput(values, "lz4")
	}
	return values
}

func addToOutput(output, value string) string {
	if output == "" {
		output = value
	} else {
		output = fmt.Sprintf("%s\n%s", output, value)
	}
	return output
}
