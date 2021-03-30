/*
Copyright 2021 The OpenEBS Authors.

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

package webhook

import (
	"context"
	"fmt"

	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func (wh *webhook) validateNamespace(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request
	response := &v1.AdmissionResponse{}
	response.Allowed = true
	openebsNamespace, err := getOpenebsNamespace()
	if err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("error getting OPENEBS_NAMESPACE env %s: %v", req.Name, err.Error()),
		}
		return response
	}
	// validates only if requested operation is DELETE
	if openebsNamespace == req.Name && req.Operation == v1.Delete {
		return wh.validateNamespaceDeleteRequest(req)
	}
	klog.V(2).Info("Admission wehbook for Namespace module not " +
		"configured for operations other than DELETE")
	return response
}

func (wh *webhook) validateNamespaceDeleteRequest(req *v1.AdmissionRequest) *v1.AdmissionResponse {
	response := &v1.AdmissionResponse{}
	response.Allowed = true

	// ignore the Delete request of Namespace if resource name is empty
	if req.Name == "" {
		return response
	}
	// Delete the validatingWebhookConfiguration only if its a delete request to
	// delete openebs namespace
	err := wh.kubeClient.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Delete(context.TODO(), validatorWebhook, metav1.DeleteOptions{})
	if err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{
			Message: err.Error(),
		}
		return response
	}
	return response
}
