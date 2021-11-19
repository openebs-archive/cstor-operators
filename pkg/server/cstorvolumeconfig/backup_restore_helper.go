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

package cstorvolumeconfig

import (
	"context"

	cstortypes "github.com/openebs/api/v3/pkg/apis/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// checkIfPoolManagerNodeDown will check if CSPI pool manager is in running or not
func checkIfPoolManagerNodeDown(k8sclient kubernetes.Interface, cspiName, namespace string) bool {
	var nodeDown = true
	var pod *corev1.Pod
	var err error

	// If cspiName is not empty then fetch the CStor pool pod using CSPI name
	if cspiName == "" {
		klog.Errorf("failed to find pool manager, empty CSPI is provided")
		return nodeDown
	}
	pod, err = getPoolManager(k8sclient, cspiName, namespace)
	if err != nil {
		klog.Errorf("Failed to find pool manager for CSPI:%s err:%s", cspiName, err.Error())
		return nodeDown
	}

	if pod.Spec.NodeName == "" {
		klog.Errorf("node name is empty in pool manager %s", pod.Name)
		return nodeDown
	}

	node, err := k8sclient.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		klog.Infof("Failed to fetch node info for CSPI:%s: %v", cspiName, err)
		return nodeDown
	}
	for _, nodestat := range node.Status.Conditions {
		if nodestat.Type == corev1.NodeReady && nodestat.Status != corev1.ConditionTrue {
			klog.Infof("Node:%v is not in ready state", node.Name)
			return nodeDown
		}
	}
	return !nodeDown
}

// checkIfPoolManagerDown will check if pool pod is running or not
func checkIfPoolManagerDown(k8sclient kubernetes.Interface, cspiName, namespace string) bool {
	var podDown = true
	var pod *corev1.Pod
	var err error

	// If cspiName is not empty then fetch the CStor pool pod using CSPI name
	if cspiName == "" {
		klog.Errorf("failed to find pool manager, empty CSPI is provided")
		return podDown
	}
	pod, err = getPoolManager(k8sclient, cspiName, namespace)
	if err != nil {
		klog.Errorf("Failed to find pool manager for CSPI:%s err:%s", cspiName, err.Error())
		return podDown
	}

	for _, containerstatus := range pod.Status.ContainerStatuses {
		if containerstatus.Name == "cstor-pool-mgmt" {
			return !containerstatus.Ready
		}
	}

	return podDown
}

// getPoolManager returns pool manager pod for provided CSPI
func getPoolManager(k8sclientset kubernetes.Interface, cspiName, openebsNs string) (*corev1.Pod, error) {
	cstorPodLabel := "app=cstor-pool"
	cspiPoolName := cstortypes.CStorPoolInstanceLabelKey + "=" + cspiName
	podlistops := metav1.ListOptions{
		LabelSelector: cstorPodLabel + "," + cspiPoolName,
	}

	if openebsNs == "" {
		return nil, errors.Errorf("Failed to fetch operator namespace")
	}

	podList, err := k8sclientset.CoreV1().Pods(openebsNs).List(context.TODO(), podlistops)
	if err != nil {
		klog.Errorf("Failed to fetch pod list :%v", err)
		return nil, err
	}

	if len(podList.Items) != 1 {
		return nil, errors.Errorf("expected 1 pool manager but got %d pool managers", len(podList.Items))
	}
	return &podList.Items[0], nil
}
