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

package cstorvolumeconfig

import (
	apis "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// defaultQueueDepth represents the queue size at iSCSI target which limits
	// the ongoing IO count from client.
	defaultQueueDepth = "32"
	// defaultIOWorkers represents default (luWorker) number of threads that are
	// working on queue
	defaultIOWorkers = int64(6)
)

type policyOptFuncs func(*apis.CStorVolumePolicySpec, apis.CStorVolumePolicySpec)

// validatePolicySpec validates the provided policy created by the user and
// otherwise sets the defaults policy spec for cstor volumes.
func validatePolicySpec(policy *apis.CStorVolumePolicySpec) {
	defaultPolicy := getDefaultPolicySpec()
	optFuncs := []policyOptFuncs{
		defaultTargetPolicy, defaultReplicaPolicy,
	}
	for _, o := range optFuncs {
		o(policy, defaultPolicy)
	}
}

// defaultTargetPolicy configure the default volume target deployment related policies
func defaultTargetPolicy(policy *apis.CStorVolumePolicySpec, defaultPolicy apis.CStorVolumePolicySpec) {
	if policy.Target.Resources == nil {
		policy.Target.Resources = defaultPolicy.Target.Resources
	}
	if policy.Target.AuxResources == nil {
		policy.Target.AuxResources = defaultPolicy.Target.AuxResources
	}
	if policy.Target.IOWorkers == 0 {
		policy.Target.IOWorkers = defaultPolicy.Target.IOWorkers
	}
	if policy.Target.QueueDepth == "" {
		policy.Target.QueueDepth = defaultPolicy.Target.QueueDepth
	}

}

// defaultReplicaPolicy configure the default volume replica related policies
func defaultReplicaPolicy(policy *apis.CStorVolumePolicySpec, defaultPolicy apis.CStorVolumePolicySpec) {
}

// getDefaultPolicySpec generate default cstor volume policy spec.
func getDefaultPolicySpec() apis.CStorVolumePolicySpec {
	return apis.CStorVolumePolicySpec{
		Target: apis.TargetSpec{
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
			},
			AuxResources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
			},
			QueueDepth: defaultQueueDepth,
			IOWorkers:  defaultIOWorkers,
		},
		Replica: apis.ReplicaSpec{},
	}
}
