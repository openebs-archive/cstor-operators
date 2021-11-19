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
	"os"
	"strings"
)

const (

	// OpenEBSNamespace is the environment variable to get openebs namespace
	//
	// This environment variable is set via kubernetes downward API
	OpenEBSNamespace = "OPENEBS_NAMESPACE"

	// OpenEBSBaseDir is the environment variable to get base directory of
	// openebs
	OpenEBSBaseDir = "OPENEBS_IO_BASE_DIR"

	// Namespace is the environment variable to get openebs namespace
	//
	// This environment variable is set via kubernetes downward API
	Namespace = "NAMESPACE"

	// DefaultOpenEBSServiceAccount name of the default openebs service account with
	// required permissions
	DefaultOpenEBSServiceAccount = "openebs-maya-operator"

	// OpenEBSServiceAccount is the environment variable to get operator service
	// account name
	//
	// This environment variable is set via kubernetes downward API in cvc and
	// cspc operators deployments
	OpenEBSServiceAccount = "OPENEBS_SERVICEACCOUNT_NAME"

	// OpenEBSImagePullSecret is the environment variable that provides the image pull secrets
	OpenEBSImagePullSecret = "OPENEBS_IO_IMAGE_PULL_SECRETS"
)

// LookupOrFalse looks up an environment variable and returns a string "false"
// if environment variable is not present. It returns appropriate values for
// other cases.
func LookupOrFalse(envKey string) string {
	val, present := lookupEnv(envKey)
	if !present {
		return "false"
	}
	return strings.TrimSpace(val)
}

// GetEnv fetches the provided environment variable's value
func GetEnv(envKey string) (value string) {
	return strings.TrimSpace(os.Getenv(envKey))
}

// lookupEnv looks up the provided environment variable
func lookupEnv(envKey string) (value string, present bool) {
	value, present = os.LookupEnv(envKey)
	value = strings.TrimSpace(value)
	return
}

// GetOpenebsBaseDirPath returns the base path to store openebs related files on
// host machine
func GetOpenebsBaseDirPath() string {
	baseDir, isPresent := lookupEnv(string(OpenEBSBaseDir))
	if !isPresent {
		return "/var/openebs"
	}
	return baseDir
}

// GetNamespace gets the namespace OPENEBS_NAMESPACE env value which is set by the
// downward API where CVC-Operator has been deployed
func GetNamespace() string {
	return GetEnv(OpenEBSNamespace)
}

// GetServiceAccountName gets the name of OPENEBS_SERVICEACCOUNT_NAME env value which is set by the
// downward API of cvc and cspc operator deployments
func GetServiceAccountName() string {
	name, present := os.LookupEnv(OpenEBSServiceAccount)
	if !present {
		name = DefaultOpenEBSServiceAccount
	}
	return name
}

// GetOpenEBSImagePullSecrets gets the image pull secrets as string from the environment variable
func GetOpenEBSImagePullSecrets() string {
	return os.Getenv(OpenEBSImagePullSecret)
}
