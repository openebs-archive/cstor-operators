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

package apps

import (
	corebuilder "github.com/openebs/api/v3/pkg/kubernetes/core"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Deployment struct {
	*appsv1.Deployment
}

// NewDeployment returns an empty instance of deployment
func NewDeployment() *Deployment {
	return &Deployment{
		&appsv1.Deployment{},
	}
}

// WithName sets the Name field of deployment with provided value.
func (d *Deployment) WithName(name string) *Deployment {
	d.Name = name
	return d
}

// WithNamespace sets the Namespace field of deployment with provided value.
func (d *Deployment) WithNamespace(namespace string) *Deployment {
	d.Namespace = namespace
	return d
}

// WithAnnotations merges existing annotations if any
// with the ones that are provided here
func (d *Deployment) WithAnnotations(annotations map[string]string) *Deployment {
	if d.Annotations == nil {
		return d.WithAnnotationsNew(annotations)
	}

	for key, value := range annotations {
		d.Annotations[key] = value
	}
	return d
}

// WithAnnotationsNew resets existing annotaions if any with
// ones that are provided here
func (d *Deployment) WithAnnotationsNew(annotations map[string]string) *Deployment {
	newannotations := make(map[string]string)
	for key, value := range annotations {
		newannotations[key] = value
	}

	d.Annotations = newannotations
	return d
}

// WithNodeSelector Sets the node selector with the provided argument.
func (d *Deployment) WithNodeSelector(selector map[string]string) *Deployment {
	if d.Spec.Template.Spec.NodeSelector == nil {
		return d.WithNodeSelectorNew(selector)
	}

	for key, value := range selector {
		d.Spec.Template.Spec.NodeSelector[key] = value
	}
	return d
}

// WithNodeSelector Sets the node selector with the provided argument.
func (d *Deployment) WithNodeSelectorNew(selector map[string]string) *Deployment {
	d.Spec.Template.Spec.NodeSelector = selector
	return d
}

// WithOwnerReferenceNew sets ownerreference if any with
// ones that are provided here
func (d *Deployment) WithOwnerReferenceNew(ownerRefernce []metav1.OwnerReference) *Deployment {
	d.OwnerReferences = ownerRefernce
	return d
}

// WithLabels merges existing labels if any
// with the ones that are provided here
func (d *Deployment) WithLabels(labels map[string]string) *Deployment {
	if d.Labels == nil {
		return d.WithLabelsNew(labels)
	}

	for key, value := range labels {
		d.Labels[key] = value
	}
	return d
}

// WithLabelsNew resets existing labels if any with
// ones that are provided here
func (d *Deployment) WithLabelsNew(labels map[string]string) *Deployment {
	newLabels := make(map[string]string)
	for key, value := range labels {
		newLabels[key] = value
	}

	d.Labels = newLabels
	return d
}

// WithSelectorMatchLabels merges existing matchlabels if any
// with the ones that are provided here
func (d *Deployment) WithSelectorMatchLabels(matchlabels map[string]string) *Deployment {
	if d.Spec.Selector == nil {
		return d.WithSelectorMatchLabelsNew(matchlabels)
	}

	for key, value := range matchlabels {
		d.Spec.Selector.MatchLabels[key] = value
	}
	return d
}

// WithSelectorMatchLabelsNew resets existing matchlabels if any with
// ones that are provided here
func (d *Deployment) WithSelectorMatchLabelsNew(matchlabels map[string]string) *Deployment {
	newmatchlabels := make(map[string]string)
	for key, value := range matchlabels {
		newmatchlabels[key] = value
	}

	newselector := &metav1.LabelSelector{
		MatchLabels: newmatchlabels,
	}

	d.Spec.Selector = newselector
	return d
}

// WithReplicas sets the replica field of deployment.
// Caller should not pass nil value.
func (d *Deployment) WithReplicas(replicas *int32) *Deployment {
	newreplicas := *replicas
	d.Spec.Replicas = &newreplicas
	return d
}

//WithStrategyType sets the strategy field of the deployment
func (d *Deployment) WithStrategyType(strategytype appsv1.DeploymentStrategyType) *Deployment {
	d.Spec.Strategy.Type = strategytype
	return d
}

// WithPodTemplateSpecBuilder sets the template field of the deployment
func (d *Deployment) WithPodTemplateSpec(pts *corebuilder.PodTemplateSpec) *Deployment {
	templatespecObj := pts.Build()
	d.Spec.Template = *templatespecObj
	return d
}

// Build returns a deployment instance
func (d *Deployment) Build() *appsv1.Deployment {
	return d.Deployment
}
