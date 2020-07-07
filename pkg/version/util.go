package version

import (
	"strings"
)

var (
	validCurrentVersions = map[string]bool{
		"1.9.0": true, "1.10.0": true,
	}
	validDesiredVersion = strings.Split(GetVersion(), "-")[0]
)

// IsCurrentVersionValid verifies if the  current version is valid or not
func IsCurrentVersionValid(v string) bool {
	currentVersion := strings.Split(v, "-")[0]
	return validCurrentVersions[currentVersion]
}

// IsDesiredVersionValid verifies the desired version is valid or not
func IsDesiredVersionValid(v string) bool {
	desiredVersion := strings.Split(v, "-")[0]
	return validDesiredVersion == desiredVersion
}
