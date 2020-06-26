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
package zpool

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// GetProperty mocks the zpool get command and returns the error based on the output
func (poolMocker *PoolMocker) GetProperty(cmd string) ([]byte, error) {
	var isProperty bool
	var output string

	// If configuration expects error then return error
	if poolMocker.TestConfig.ZpoolCommand.ZpoolGetError {
		return getPropertyError(cmd)
	}

	values := strings.Split(cmd, " ")
	if poolMocker.PoolName == "" {
		return []byte(fmt.Sprintf("cannot open '%s': no such pool", values[len(values)-1])), errors.Errorf("exit statu 1")
	}
	if !strings.Contains(cmd, poolMocker.PoolName) {
		return []byte(fmt.Sprintf("cannot open '%s': no such pool", values[len(values)-1])), errors.Errorf("exit statu 1")
	}

	//TODO: Imporve below return values based on topology
	// Command: zpool get  -H  -o value, health,io.openebs:readonly,  cstor-1234
	for _, val := range values {
		if val == " " {
			continue
		}
		if val == "value," {
			isProperty = true
		}
		// If command is to get pool name
		if isProperty && strings.Contains(val, "name") {
			// We are fetching the pool only during starting reconciliation
			// So here good reduce ResilveringProgress count
			output = addToOutput(output, poolMocker.PoolName)
			if poolMocker.IsReplacementInProgress && poolMocker.TestConfig.ResilveringProgress > 0 {
				poolMocker.TestConfig.ResilveringProgress--
			}
		}

		if isProperty {
			output = addToOutput(output, poolMocker.TestConfig.getPropertyValues(val))
		}
	}
	return []byte(output), nil
}

// getPropertyValues returns the values for quaried properties
func (tc *TestConfig) getPropertyValues(command string) string {
	var values string
	// If command is to get free space in pool
	if strings.Contains(command, "free") {
		values = addToOutput(values, "9.94G")
	}
	// If command is to get allocated space in pool
	if strings.Contains(command, "allocated") {
		values = addToOutput(values, "69.5K")
	}
	// If command is to get size space in pool
	if strings.Contains(command, "size") {
		values = addToOutput(values, "9.94G")
	}

	if strings.Contains(command, "health") {
		values = addToOutput(values, "ONLINE")
	}

	if strings.Contains(command, "io.openebs:readonly") {
		values = addToOutput(values, "off")
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

func getPropertyError(cmd string) ([]byte, error) {
	return []byte("fake error to get values"), errors.Errorf("exit status 1")
}
