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
	"time"

	. "github.com/onsi/gomega"
	cstor "github.com/openebs/api/v3/pkg/apis/cstor/v1"
	"github.com/openebs/api/v3/pkg/apis/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// GetBDReplacmentStatusOnCSPI gets the status of block device replacement
func (client *Client) GetBDReplacmentStatusOnCSPI(cspcName, cspcNamespace, hostNameValue string, expectedStatus bool) bool {
	gotStatus := false
	for i := 0; i < (maxRetry + 100); i++ {
		ls := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				types.CStorPoolClusterLabelKey: cspcName,
				types.HostNameLabelKey:         hostNameValue,
			},
		}
		cspiList, err := client.OpenEBSClientSet.CstorV1().
			CStorPoolInstances(cspcNamespace).
			List(context.TODO(), metav1.ListOptions{
				LabelSelector: labels.Set(ls.MatchLabels).String(),
			})
		Expect(err).To(BeNil())
		Expect(len(cspiList.Items)).To(BeNumerically("==", 1))
		for _, v := range cspiList.Items[0].Status.Conditions {
			if v.Type == cstor.CSPIDiskReplacement && v.Reason == "BlockDeviceReplacementSucceess" {
				gotStatus = true
			}
		}

		if expectedStatus == gotStatus {
			return gotStatus
		}

		time.Sleep(3 * time.Second)
	}

	return gotStatus
}
