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

package core

import (
	corev1 "k8s.io/api/core/v1"
)

// container encapsulates kubernetes container type
type Container struct {
	*corev1.Container
}

// NewContainer returns a new instance of container
func NewContainer() *Container {
	return &Container{
		&corev1.Container{},
	}
}

// WithName sets the name of the container
func (c *Container) WithName(name string) *Container {
	c.Name = name
	return c
}

// WithImage sets the image of the container
func (c *Container) WithImage(img string) *Container {
	c.Image = img
	return c
}

// WithCommandNew sets the command of the container
func (c *Container) WithCommandNew(cmd []string) *Container {

	newcmd := []string{}
	newcmd = append(newcmd, cmd...)
	c.Command = newcmd
	return c
}

// WithArgumentsNew sets the command arguments of the container
func (c *Container) WithArgumentsNew(args []string) *Container {
	newargs := []string{}
	newargs = append(newargs, args...)
	c.Args = newargs
	return c
}

// WithVolumeMountsNew sets the command arguments of the container
func (c *Container) WithVolumeMountsNew(volumeMounts []corev1.VolumeMount) *Container {
	newvolumeMounts := []corev1.VolumeMount{}
	newvolumeMounts = append(newvolumeMounts, volumeMounts...)
	c.VolumeMounts = newvolumeMounts
	return c
}

// WithVolumeDevicesNew sets the containers with the appropriate volumeDevices
func (c *Container) WithVolumeDevicesNew(volumeDevices []corev1.VolumeDevice) *Container {
	newVolumeDevices := []corev1.VolumeDevice{}
	newVolumeDevices = append(newVolumeDevices, volumeDevices...)
	c.VolumeDevices = newVolumeDevices
	return c
}

// WithImagePullPolicy sets the image pull policy of the container
func (c *Container) WithImagePullPolicy(policy corev1.PullPolicy) *Container {
	c.ImagePullPolicy = policy
	return c
}

// WithPrivilegedSecurityContext sets securitycontext of the container
func (c *Container) WithPrivilegedSecurityContext(privileged *bool) *Container {
	newprivileged := *privileged
	newsecuritycontext := &corev1.SecurityContext{
		Privileged: &newprivileged,
	}

	c.SecurityContext = newsecuritycontext
	return c
}

// WithResources sets resources of the container
func (c *Container) WithResources(resources corev1.ResourceRequirements) *Container {
	c.Resources = resources
	return c
}

func (c *Container) WithResourcesByRef(resources *corev1.ResourceRequirements) *Container {
	newresources := *resources
	c.Resources = newresources
	return c
}

// WithPortsNew sets ports of the container
func (c *Container) WithPortsNew(ports []corev1.ContainerPort) *Container {
	newports := []corev1.ContainerPort{}
	newports = append(newports, ports...)
	c.Ports = newports
	return c
}

// WithEnvsNew sets the envs of the container
func (c *Container) WithEnvsNew(envs []corev1.EnvVar) *Container {
	newenvs := []corev1.EnvVar{}
	newenvs = append(newenvs, envs...)
	c.Env = newenvs
	return c
}

// WithTTY sets the tty boolean for container
func (c *Container) WithTTY(tty bool) *Container {
	c.TTY = tty
	return c
}

// WithEnvs sets the envs of the container
func (c *Container) WithEnvs(envs []corev1.EnvVar) *Container {
	c.Env = append(c.Env, envs...)
	return c
}

// WithLivenessProbe sets the liveness probe of the container
func (c *Container) WithLivenessProbe(liveness *corev1.Probe) *Container {
	c.LivenessProbe = liveness
	return c
}

// WithLifeCycle sets the life cycle of the container
func (c *Container) WithLifeCycle(lc *corev1.Lifecycle) *Container {
	c.Lifecycle = lc
	return c
}

func (c *Container) Build() *corev1.Container {
	return c.Container
}
