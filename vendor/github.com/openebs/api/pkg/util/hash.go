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

package util

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

// Hash constructs the hash value for any type of object
func Hash(obj interface{}) (string, error) {

	// Convert the given object into json encoded bytes
	jsonEncodedValues, err := json.Marshal(obj)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert the object to json encoded format")
	}
	hashBytes := md5.Sum(jsonEncodedValues)
	return hex.EncodeToString(hashBytes[:]), nil
}
