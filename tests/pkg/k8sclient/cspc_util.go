/*
Copyright 2020 The OpenEBS Authors

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

package k8sclient

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/gomega"
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/openebs.io/v1alpha1"
	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const maxRetry = 30

// GetHealthyCSPICountEventually gets online cspi(s) based on cspc name
func (client *Client) GetOnlineCSPICountEventually(cspcName, cspcNamespace string, expectedCSPICount int) int {
	var cspiCount int
	// as cspi deletion takes more time now for cleanup of its resources
	// for reconciled cspi to come up it can take additional time.
	for i := 0; i < (maxRetry + 100); i++ {
		cspiList, err := client.GetCSPIList(cspcName, cspcNamespace)
		Expect(err).To(BeNil())
		filteredList := cspiList.Filter(cstor.IsOnline())
		cspiCount = len(filteredList.Items)
		if cspiCount == expectedCSPICount {
			return cspiCount
		}
		time.Sleep(3 * time.Second)
	}
	return cspiCount
}

// GetCSPICountEventually gets cspi(s) based on cspc name
func (client *Client) GetCSPICountEventually(cspcName, cspcNamespace string, expectedCSPICount int) int {
	var cspiCount int
	for i := 0; i < (maxRetry + 100); i++ {
		cspiList, err := client.GetCSPIList(cspcName, cspcNamespace)
		Expect(err).To(BeNil())
		cspiCount = len(cspiList.Items)
		if cspiCount == expectedCSPICount {
			return cspiCount
		}
		time.Sleep(3 * time.Second)
	}
	return cspiCount
}

// GetPoolManagerCountEventually gets the pool-manager deployment count based on cspc name
func (client *Client) GetPoolManagerCountEventually(cspcName, cspcNamespace string, expectedCSPICount int) int {
	var pmCount int
	for i := 0; i < (maxRetry + 100); i++ {
		pmList := client.GetPoolManagerList(cspcName, cspcNamespace)
		pmCount = len(pmList.Items)
		if pmCount == expectedCSPICount {
			return pmCount
		}
		time.Sleep(3 * time.Second)
	}
	return pmCount
}

// GetBDCCountEventually gets the bdc count based on cspc name and namespace
func (client *Client) GetBDCCountEventually(cspcName, cspcNamespace string, expectedBDCCount int) int {
	var bdcCount int
	for i := 0; i < (maxRetry + 100); i++ {
		bdcList := client.GetBDCList(cspcName, cspcNamespace)
		bdcCount = len(bdcList.Items)
		if bdcCount == expectedBDCCount {
			return bdcCount
		}
		time.Sleep(3 * time.Second)
	}
	return bdcCount
}

// GetProvisionedInstancesStatusOnCSPC gets provisioned instances count based on cspc name
// and namespace.
func (client *Client) GetProvisionedInstancesStatusOnCSPC(cspcName, cspcNamespace string,
	expectedProvisionedInstancesStatus int32) int32 {
	var gotProvisionedInstances int32
	for i := 0; i < (maxRetry + 100); i++ {
		cspc := client.GetCSPC(cspcName, cspcNamespace)
		gotProvisionedInstances = cspc.Status.ProvisionedInstances
		if gotProvisionedInstances == expectedProvisionedInstancesStatus {
			return gotProvisionedInstances
		}
		time.Sleep(3 * time.Second)
	}
	return gotProvisionedInstances
}

// GetCStorPoolInstanceNames will return list of cStor pool instance names belongs to provided CSPC
func (client *Client) GetCStorPoolInstanceNames(cspcName, cspcNamespace string) ([]string, error) {
	cspiList, err := client.GetCSPIList(cspcName, cspcNamespace)
	if err != nil {
		return []string{}, errors.Wrapf(err, "failed to list pool instances belongs to %s", cspcName)
	}
	poolNames := make([]string, len(cspiList.Items))
	for i, cspiObj := range cspiList.Items {
		poolNames[i] = cspiObj.Name
	}
	return poolNames, nil
}

// GetHealthyInstancesStatusOnCSPC gets healthy instances count based on cspc name
// and namespace.
func (client *Client) GetHealthyInstancesStatusOnCSPC(cspcName, cspcNamespace string,
	expectedHealthyInstancesStatus int32) int32 {
	var gotHealthyInstances int32
	for i := 0; i < (maxRetry + 100); i++ {
		cspc := client.GetCSPC(cspcName, cspcNamespace)
		gotHealthyInstances = cspc.Status.HealthyInstances
		if gotHealthyInstances == expectedHealthyInstancesStatus {
			return gotHealthyInstances
		}
		time.Sleep(3 * time.Second)
	}
	return gotHealthyInstances
}

// GetDesiredInstancesStatusOnCSPC gets desired instances count based on cspc name
// and namespace.
func (client *Client) GetDesiredInstancesStatusOnCSPC(cspcName, cspcNamespace string,
	expectedDesiredInstancesStatus int32) int32 {
	var gotDesiredInstances int32
	for i := 0; i < (maxRetry + 100); i++ {
		cspc := client.GetCSPC(cspcName, cspcNamespace)
		gotDesiredInstances = cspc.Status.DesiredInstances
		if gotDesiredInstances == expectedDesiredInstancesStatus {
			return gotDesiredInstances
		}
		time.Sleep(3 * time.Second)
	}
	return gotDesiredInstances
}

// GetCSPIList gets the list of all cspi(s) based on cspc name and namespace.
func (client *Client) GetCSPIList(cspcName, cspcNamespace string) (*cstor.CStorPoolInstanceList, error) {
	return client.OpenEBSClientSet.CstorV1().
		CStorPoolInstances(cspcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspcName})
}

// GetPoolManagerList gets the list of all pool-manger deployments based on cspc name and namespace.
func (client *Client) GetPoolManagerList(cspcName, cspcNamespace string) *v1.DeploymentList {
	pmList, err := client.KubeClientSet.AppsV1().
		Deployments(cspcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspcName})
	Expect(err).To(BeNil())
	return pmList
}

// GetBDCList gets the list of all bdc(s) based on cspc name and namespace.
func (client *Client) GetBDCList(cspcName, cspcNamespace string) *v1alpha1.BlockDeviceClaimList {
	bdcList, err := client.OpenEBSClientSet.OpenebsV1alpha1().
		BlockDeviceClaims(cspcNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: types.CStorPoolClusterLabelKey + "=" + cspcName})
	Expect(err).To(BeNil())
	return bdcList
}

// GetPoolManagerList gets the list of all pool-manger deployments based on cspc name and namespace.
func (client *Client) GetCSPC(cspcName, cspcNamespace string) *cstor.CStorPoolCluster {
	cspc, err := client.OpenEBSClientSet.CstorV1().CStorPoolClusters(cspcNamespace).Get(context.TODO(), cspcName, metav1.GetOptions{})
	Expect(err).To(BeNil())
	return cspc
}

// HasResourceLimitOnCSPIEventually ...
func (client *Client) HasResourceLimitOnCSPIEventually(
	cspcName, cspcNamespace string, expectedResource *corev1.ResourceRequirements) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		resourceLimitMatches := true
		cspiList, err := client.GetCSPIList(cspcName, cspcNamespace)
		Expect(err).To(BeNil())
		for _, v := range cspiList.Items {
			if !reflect.DeepEqual(v.Spec.PoolConfig.Resources, expectedResource) {
				resourceLimitMatches = false
			}
		}
		if resourceLimitMatches {
			return resourceLimitMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasResourceLimitOnPoolManagerEventually ...
func (client *Client) HasResourceLimitOnPoolManagerEventually(
	cspcName, cspcNamespace string, expectedResource *corev1.ResourceRequirements) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		resourceLimitMatches := true
		pmList := client.GetPoolManagerList(cspcName, cspcNamespace)
		for _, pm := range pmList.Items {
			for _, container := range pm.Spec.Template.Spec.Containers {
				if container.Name == "cstor-pool" {
					if !reflect.DeepEqual(container.Resources, *expectedResource) {
						resourceLimitMatches = false
					}
				}
			}
		}
		if resourceLimitMatches {
			return resourceLimitMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasTolerationsOnCSPIEventually ...
func (client *Client) HasTolerationsOnCSPIEventually(
	cspcName, cspcNamespace string, tolerations []corev1.Toleration) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		tolerationsMatches := true
		cspiList, _ := client.GetCSPIList(cspcName, cspcNamespace)
		for _, v := range cspiList.Items {
			if !reflect.DeepEqual(v.Spec.PoolConfig.Tolerations, tolerations) {
				tolerationsMatches = false
			}
		}
		if tolerationsMatches {
			return tolerationsMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasTolerationsOnPoolManagerEventually ...
func (client *Client) HasTolerationsOnPoolManagerEventually(
	cspcName, cspcNamespace string, tolerations []corev1.Toleration) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		tolerationsMatches := true
		pmList := client.GetPoolManagerList(cspcName, cspcNamespace)
		for _, pm := range pmList.Items {
			if !reflect.DeepEqual(pm.Spec.Template.Spec.Tolerations, tolerations) {
				tolerationsMatches = false
			}
		}
		if tolerationsMatches {
			return tolerationsMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasPriorityClassOnCSPIEventually ...
func (client *Client) HasPriorityClassOnCSPIEventually(
	cspcName, cspcNamespace string, priorityClass *string) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		priorityClassMatches := true
		cspiList, _ := client.GetCSPIList(cspcName, cspcNamespace)
		for _, v := range cspiList.Items {
			if !reflect.DeepEqual(v.Spec.PoolConfig.PriorityClassName, priorityClass) {
				priorityClassMatches = false
			}
		}
		if priorityClassMatches {
			return priorityClassMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasPriorityClassOnPoolManagerEventually ...
func (client *Client) HasPriorityClassOnPoolManagerEventually(
	cspcName, cspcNamespace string, priorityClass *string) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		priorityClassMatches := true
		pmList := client.GetPoolManagerList(cspcName, cspcNamespace)
		for _, pm := range pmList.Items {
			if !reflect.DeepEqual(pm.Spec.Template.Spec.PriorityClassName, *priorityClass) {
				priorityClassMatches = false
			}
		}
		if priorityClassMatches {
			return priorityClassMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasCompressionOnCSPIEventually ...
func (client *Client) HasCompressionOnCSPIEventually(
	cspcName, cspcNamespace string, compression string) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		compressionMatches := true
		cspiList, _ := client.GetCSPIList(cspcName, cspcNamespace)
		for _, v := range cspiList.Items {
			if v.Spec.PoolConfig.Compression != compression {
				compressionMatches = false
			}
		}
		if compressionMatches {
			return compressionMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// HasROThresholdOnCSPIEventually ...
func (client *Client) HasROThresholdOnCSPIEventually(
	cspcName, cspcNamespace string, roThreshold *int) bool {
	for i := 0; i < (maxRetry + 100); i++ {
		roThresholdMatches := true
		cspiList, _ := client.GetCSPIList(cspcName, cspcNamespace)
		for _, v := range cspiList.Items {
			if !reflect.DeepEqual(v.Spec.PoolConfig.ROThresholdLimit, roThreshold) {
				roThresholdMatches = false
			}
		}
		if roThresholdMatches {
			return roThresholdMatches
		}
		time.Sleep(3 * time.Second)
	}
	return false
}
