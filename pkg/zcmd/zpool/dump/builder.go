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

package pstatus

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	vdump "github.com/openebs/api/v2/pkg/internalapis/apis/cstor"
	"github.com/openebs/cstor-operators/pkg/zcmd/bin"
	"github.com/pkg/errors"
)

const (
	// Operation defines type of zfs operation
	Operation = "dump"
)

//PoolDump defines structure for pool 'Status' operation
type PoolDump struct {
	//pool name
	Pool string

	// command string
	Command string

	// checks is list of predicate function used for validating object
	checks []PredicateFunc

	// StripVdevPath to stip partition path if whole disk is used for pool
	StripVdevPath bool

	// error
	err error

	// Executor is to execute the commands
	Executor bin.Executor
}

// NewPoolDump returns new instance of object PoolDump
func NewPoolDump() *PoolDump {
	return &PoolDump{}
}

// WithCheck add given check to checks list
func (p *PoolDump) WithCheck(check ...PredicateFunc) *PoolDump {
	p.checks = append(p.checks, check...)
	return p
}

// WithPool method fills the Pool field of PoolDump object.
func (p *PoolDump) WithPool(Pool string) *PoolDump {
	p.Pool = Pool
	return p
}

// WithCommand method fills the Command field of PoolDump object.
func (p *PoolDump) WithCommand(Command string) *PoolDump {
	p.Command = Command
	return p
}

// WithStripVdevPath method will set StripVdevPath for PoolDump object
func (p *PoolDump) WithStripVdevPath() *PoolDump {
	p.StripVdevPath = true
	return p
}

// WithExecutor method fills the Executor field of PoolDump object.
func (p *PoolDump) WithExecutor(executor bin.Executor) *PoolDump {
	p.Executor = executor
	return p
}

// Validate is to validate generated PoolDump object by builder
func (p *PoolDump) Validate() *PoolDump {
	for _, check := range p.checks {
		if !check(p) {
			p.err = errors.Wrapf(p.err, "validation failed {%v}", runtime.FuncForPC(reflect.ValueOf(check).Pointer()).Name())
		}
	}
	return p
}

// Execute is to execute generated PoolDump object
func (p *PoolDump) Execute() (vdump.Topology, error) {
	var t vdump.Topology
	var out []byte
	var err error

	p, err = p.Build()
	if err != nil {
		return t, err
	}

	if IsExecutorSet()(p) {
		out, err = p.Executor.Execute(p.Command)
	} else {
		// execute command here
		// #nosec
		out, err = exec.Command(bin.BASH, "-c", p.Command).CombinedOutput()
	}
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(out, &t)

	if p.StripVdevPath {
		stripDiskPath(t.VdevTree.Topvdev)
		stripDiskPath(t.VdevTree.Spares)
		stripDiskPath(t.VdevTree.Readcache)
	}
	return t, err
}

// Build returns the PoolDump object generated by builder
func (p *PoolDump) Build() (*PoolDump, error) {
	var c strings.Builder
	p = p.Validate()
	p.appendCommand(&c, bin.ZPOOL)
	p.appendCommand(&c, fmt.Sprintf(" %s ", Operation))

	p.appendCommand(&c, p.Pool)

	p.Command = c.String()
	return p, p.err
}

// appendCommand append string to given string builder
func (p *PoolDump) appendCommand(c *strings.Builder, cmd string) {
	_, err := c.WriteString(cmd)
	if err != nil {
		p.err = errors.Wrapf(p.err, "Failed to append cmd{%s} : %s", cmd, err.Error())
	}
}
