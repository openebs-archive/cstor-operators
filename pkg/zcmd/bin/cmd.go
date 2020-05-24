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

package bin

import (
	"os/exec"
)

const (
	// ZPOOL is zpool command name
	ZPOOL = "zpool"

	// BASH is bash command name
	BASH = "bash"

	// ZFS is zfs command name
	ZFS = "zfs"
)

// Executor is an interface for executing ZPOOL/ZFS operations
type Executor interface {
	Execute(command string) ([]byte, error)
}

// Zcmd is structure which is responsible for executing ZPOOl/ZFS
// commands
type Zcmd struct{}

// NewZcmd is new instance of Zcmd
func NewZcmd() *Zcmd {
	return &Zcmd{}
}

// Execute is to execute zpool/zfs commands in bash
func (z *Zcmd) Execute(command string) ([]byte, error) {
	// execute command here
	// #nosec
	return exec.Command(BASH, "-c", command).CombinedOutput()
}
