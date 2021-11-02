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

// PodTemplateSpec holds the api's podtemplatespec objects
type PodTemplateSpec struct {
	*corev1.PodTemplateSpec
}

// NewPodTemplateSpec returns new instance of PodTemplateSpec
func NewPodTemplateSpec() *PodTemplateSpec {
	return &PodTemplateSpec{
		&corev1.PodTemplateSpec{},
	}
}

// WithName sets the Name field of podtemplatespec with provided value.
func (p *PodTemplateSpec) WithName(name string) *PodTemplateSpec {
	p.Name = name
	return p
}

// WithNamespace sets the Namespace field of PodTemplateSpec with provided value.
func (p *PodTemplateSpec) WithNamespace(namespace string) *PodTemplateSpec {

	p.Namespace = namespace
	return p
}

// WithAnnotations merges existing annotations if any
// with the ones that are provided here
func (p *PodTemplateSpec) WithAnnotations(annotations map[string]string) *PodTemplateSpec {
	for key, value := range annotations {
		p.Annotations[key] = value
	}
	return p
}

// WithAnnotationsNew resets the annotation field of podtemplatespec
// with provided arguments
func (p *PodTemplateSpec) WithAnnotationsNew(annotations map[string]string) *PodTemplateSpec {
	newannotations := make(map[string]string)
	for key, value := range annotations {
		newannotations[key] = value
	}
	p.Annotations = newannotations
	return p
}

// WithLabels merges existing labels if any
// with the ones that are provided here
func (p *PodTemplateSpec) WithLabels(labels map[string]string) *PodTemplateSpec {
	if p.Labels == nil {
		return p.WithLabelsNew(labels)
	}

	for key, value := range labels {
		p.Labels[key] = value
	}
	return p
}

// WithLabelsNew resets the labels field of podtemplatespec
// with provided arguments
func (p *PodTemplateSpec) WithLabelsNew(labels map[string]string) *PodTemplateSpec {
	newLabels := make(map[string]string)
	for key, value := range labels {
		newLabels[key] = value
	}

	p.Labels = newLabels
	return p
}

// WithNodeSelector merges the nodeselectors if present
// with the provided arguments
func (p *PodTemplateSpec) WithNodeSelector(nodeselectors map[string]string) *PodTemplateSpec {
	if p.Spec.NodeSelector == nil {
		return p.WithNodeSelectorNew(nodeselectors)
	}

	for key, value := range nodeselectors {
		p.Spec.NodeSelector[key] = value
	}
	return p
}

// WithPriorityClassName sets the PriorityClassName field of podtemplatespec
func (p *PodTemplateSpec) WithPriorityClassName(prorityClassName string) *PodTemplateSpec {
	p.Spec.PriorityClassName = prorityClassName
	return p
}

// WithNodeSelectorNew resets the nodeselector field of podtemplatespec
// with provided arguments
func (p *PodTemplateSpec) WithNodeSelectorNew(nodeselectors map[string]string) *PodTemplateSpec {
	newnodeselectors := make(map[string]string)
	for key, value := range nodeselectors {
		newnodeselectors[key] = value
	}

	p.Spec.NodeSelector = newnodeselectors
	return p
}

func (p *PodTemplateSpec) WithNodeSelectorByValue(nodeselectors map[string]string) *PodTemplateSpec {
	newnodeselectors := make(map[string]string)
	for key, value := range nodeselectors {
		newnodeselectors[key] = value
	}

	p.Spec.NodeSelector = newnodeselectors
	return p
}

// WithServiceAccountName sets the ServiceAccountnNme field of podtemplatespec
func (p *PodTemplateSpec) WithServiceAccountName(serviceAccountnNme string) *PodTemplateSpec {
	p.Spec.ServiceAccountName = serviceAccountnNme
	return p
}

// WithAffinity sets the affinity field of podtemplatespec
func (p *PodTemplateSpec) WithAffinity(affinity *corev1.Affinity) *PodTemplateSpec {
	p.Spec.Affinity = affinity
	return p
}

// WithTolerationsByValue sets pod toleration.
// If provided tolerations argument is empty it does not complain.
func (p *PodTemplateSpec) WithTolerations(tolerations ...corev1.Toleration) *PodTemplateSpec {
	p.Spec.Tolerations = tolerations
	return p
}

// WithTolerationsNew sets the tolerations field of podtemplatespec
func (p *PodTemplateSpec) WithTolerationsNew(tolerations ...corev1.Toleration) *PodTemplateSpec {
	if len(tolerations) == 0 {
		return p.WithTolerations(tolerations...)
	}

	newtolerations := []corev1.Toleration{}
	newtolerations = append(newtolerations, tolerations...)

	p.Spec.Tolerations = newtolerations

	return p
}

// WithRestartPolicy sets the restart-policy for pod-spec
func (p *PodTemplateSpec) WithRestartPolicy(policy corev1.RestartPolicy) *PodTemplateSpec {
	p.Spec.RestartPolicy = policy
	return p
}

// WithContainers builds the list of containerbuilder
// provided and merges it to the containers field of the podtemplatespec
func (p *PodTemplateSpec) WithContainers(containersList ...*Container) *PodTemplateSpec {
	for _, container := range containersList {
		containerObj := container.Build()
		p.Spec.Containers = append(
			p.Spec.Containers,
			*containerObj,
		)
	}
	return p
}

// WithVolumes builds the list of volumebuilders provided
// and merges it to the volumes field of podtemplatespec.
func (p *PodTemplateSpec) WithVolumes(volumerList ...*Volume) *PodTemplateSpec {
	for _, volume := range volumerList {
		vol := volume.Build()
		p.Spec.Volumes = append(p.Spec.Volumes, *vol)
	}
	return p
}

// WithImagePullSecrets sets the pod image pull secrets
// if the length is zero then no secret is needed to pull the image
func (p *PodTemplateSpec) WithImagePullSecrets(secrets []corev1.LocalObjectReference) *PodTemplateSpec {
	if len(secrets) == 0 {
		return p
	}
	p.Spec.ImagePullSecrets = secrets
	return p
}

func (p *PodTemplateSpec) Build() *corev1.PodTemplateSpec {
	return p.PodTemplateSpec
}
