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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Service holds the api's service objects
type Service struct {
	*corev1.Service
}

// NewService returns new instance of service
func NewService() *Service {
	return &Service{
		&corev1.Service{},
	}
}

// WithName sets the Name field of service with provided value.
func (s *Service) WithName(name string) *Service {
	s.Name = name
	return s
}

// WithNamespace sets the Namespace field of Service with provided value.
func (s *Service) WithNamespace(namespace string) *Service {

	s.Namespace = namespace
	return s
}

// WithAnnotations merges existing annotations if any
// with the ones that are provided here
func (s *Service) WithAnnotations(annotations map[string]string) *Service {
	for key, value := range annotations {
		s.Annotations[key] = value
	}
	return s
}

// WithAnnotationsNew resets the annotation field of service
// with provided arguments
func (s *Service) WithAnnotationsNew(annotations map[string]string) *Service {
	newannotations := make(map[string]string)
	for key, value := range annotations {
		newannotations[key] = value
	}
	s.Annotations = newannotations
	return s
}

// WithLabels merges existing labels if any
// with the ones that are provided here
func (s *Service) WithLabels(labels map[string]string) *Service {
	if s.Labels == nil {
		return s.WithLabelsNew(labels)
	}

	for key, value := range labels {
		s.Labels[key] = value
	}
	return s
}

// WithLabelsNew resets the labels field of service
// with provided arguments
func (s *Service) WithLabelsNew(labels map[string]string) *Service {
	newLabels := make(map[string]string)
	for key, value := range labels {
		newLabels[key] = value
	}

	s.Labels = newLabels
	return s
}

// WithSelectors merges existing selectors if any
// with the ones that are provided here
func (s *Service) WithSelectors(selectors map[string]string) *Service {
	if s.Spec.Selector == nil {
		return s.WithSelectorsNew(selectors)
	}

	for key, value := range selectors {
		s.Spec.Selector[key] = value
	}
	return s
}

// WithSelectorsNew resets existing selectors if any with
// ones that are provided here
func (s *Service) WithSelectorsNew(selectors map[string]string) *Service {
	// copy of original map
	newslctrs := map[string]string{}
	for key, value := range selectors {
		newslctrs[key] = value
	}

	// override
	s.Spec.Selector = newslctrs
	return s
}

// WithPorts sets the Ports field of Service with provided arguments
func (s *Service) WithPorts(ports []corev1.ServicePort) *Service {
	// copy of original slice
	newports := []corev1.ServicePort{}
	newports = append(newports, ports...)

	// override
	s.Spec.Ports = newports
	return s
}

// WithType sets the Type field of Service with provided arguments
func (s *Service) WithType(svcType corev1.ServiceType) *Service {
	s.Spec.Type = svcType
	return s
}

// WithOwnerReferenceNew sets ownerrefernce if any with
// ones that are provided here
func (s *Service) WithOwnerReferenceNew(ownerRefernce []metav1.OwnerReference) *Service {

	s.OwnerReferences = ownerRefernce
	return s
}

// Build returns the Service API instance
func (s *Service) Build() *corev1.Service {
	return s.Service
}
