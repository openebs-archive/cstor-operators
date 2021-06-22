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

package vsnapshot

import (
	"fmt"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	"github.com/openebs/cstor-operators/pkg/zcmd/bin"
	"github.com/pkg/errors"
)

const (
	// Operation defines type of zfs operation
	Operation = "snapshot"
)

//VolumeSnapshot defines structure for volume 'Snapshot' operation
type VolumeSnapshot struct {
	//list of property
	Property []string

	//name of snapshot
	Snapshot string

	//name of dataset on which snapshot should be taken
	Dataset string

	//Recursively create snapshots of all descendent datasets
	Recursive bool

	// command string
	Command string

	// checks is list of predicate function used for validating object
	checks []PredicateFunc

	// error
	err error
}

// NewVolumeSnapshot returns new instance of object VolumeSnapshot
func NewVolumeSnapshot() *VolumeSnapshot {
	return &VolumeSnapshot{}
}

// WithCheck add given check to checks list
func (v *VolumeSnapshot) WithCheck(check ...PredicateFunc) *VolumeSnapshot {
	v.checks = append(v.checks, check...)
	return v
}

// WithProperty method fills the Property field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithProperty(key, value string) *VolumeSnapshot {
	v.Property = append(v.Property, fmt.Sprintf("%s=%s", key, value))
	return v
}

// WithRecursive method fills the Recursive field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithRecursive(Recursive bool) *VolumeSnapshot {
	v.Recursive = Recursive
	return v
}

// WithSnapshot method fills the Snapshot field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithSnapshot(Snapshot string) *VolumeSnapshot {
	v.Snapshot = Snapshot
	return v
}

// WithDataset method fills the Dataset field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithDataset(Dataset string) *VolumeSnapshot {
	v.Dataset = Dataset
	return v
}

// WithCommand method fills the Command field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithCommand(Command string) *VolumeSnapshot {
	v.Command = Command
	return v
}

// Validate is to validate generated VolumeSnapshot object by builder
func (v *VolumeSnapshot) Validate() *VolumeSnapshot {
	for _, check := range v.checks {
		if !check(v) {
			v.err = errors.Wrapf(v.err, "validation failed {%v}", runtime.FuncForPC(reflect.ValueOf(check).Pointer()).Name())
		}
	}
	return v
}

// Execute is to execute generated VolumeSnapshot object
func (v *VolumeSnapshot) Execute() ([]byte, error) {
	v, err := v.Build()
	if err != nil {
		return nil, err
	}
	// execute command here
	// #nosec
	return exec.Command(bin.BASH, "-c", v.Command).CombinedOutput()
}

// Build returns the VolumeSnapshot object generated by builder
func (v *VolumeSnapshot) Build() (*VolumeSnapshot, error) {
	var c strings.Builder
	v = v.Validate()
	v.appendCommand(&c, bin.ZFS)

	v.appendCommand(&c, fmt.Sprintf(" %s ", Operation))
	if IsRecursiveSet()(v) {
		v.appendCommand(&c, " -r ")
	}

	if IsPropertySet()(v) {
		for _, p := range v.Property {
			v.appendCommand(&c, fmt.Sprintf(" -o %s", p))
		}
	}

	v.appendCommand(&c, fmt.Sprintf(" %s@%s ", v.Dataset, v.Snapshot))

	v.Command = c.String()
	return v, v.err
}

// appendCommand append string to given string builder
func (v *VolumeSnapshot) appendCommand(c *strings.Builder, cmd string) {
	_, err := c.WriteString(cmd)
	if err != nil {
		v.err = errors.Wrapf(v.err, "Failed to append cmd{%s} : %s", cmd, err.Error())
	}
}
