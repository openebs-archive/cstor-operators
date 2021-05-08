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

package hash

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/cespare/xxhash"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

// Hash constructs the hash value for any type of object
func Hash(obj interface{}) (string, error) {

	// Convert the given object into json encoded bytes
	jsonEncodedValues, err := json.Marshal(obj)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert the object to json encoded format")
	}
	hashBytes := xxhash.Sum64(jsonEncodedValues)
	return strconv.FormatUint(hashBytes, 10), nil
}

const (
	// TemplateHashLabelName is a label to annotate a Kubernetes resource
	// with the hash of its initial template before creation.
	TemplateHashLabelName = "cstor.openebs.io/template-hash"
)

// SetTemplateHashLabel adds a label containing the hash of the given template into the
// given labels. This label can then be used for template or resource comparisons.
func SetTemplateHashLabel(labels map[string]string, template interface{}) map[string]string {
	return setHashLabel(TemplateHashLabelName, labels, template)
}

func setHashLabel(labelName string, labels map[string]string, template interface{}) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}
	labels[labelName] = HashObject(template)
	return labels
}

// GetTemplateHashLabel returns the template hash label value if set, or an empty string.
func GetTemplateHashLabel(labels map[string]string) string {
	return labels[TemplateHashLabelName]
}

// HashObject writes the specified object to a hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
// The returned hash can be used for object comparisons.
//
// This is inspired by controller revisions in StatefulSets:
// https://github.com/kubernetes/kubernetes/blob/8de1569ddae62e8fab559fe6bd210a5d6100a277/pkg/controller/history/controller_history.go#L89-L101
func HashObject(object interface{}) string {
	hf := fnv.New32()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, _ = printer.Fprintf(hf, "%#v", object)
	return fmt.Sprint(hf.Sum32())
}
