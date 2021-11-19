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

package volumemgmt

import (
	"os"
	"testing"
	"time"

	openebsFakeClientset "github.com/openebs/api/v3/pkg/client/clientset/versioned/fake"
)

func TestGetSyncInterval(t *testing.T) {
	tests := map[string]struct {
		resyncInterval string
		expectedResult time.Duration
	}{
		"resync environment variable is missing": {
			resyncInterval: "",
			expectedResult: 30 * time.Second,
		},
		"resync environment variable is non numeric": {
			resyncInterval: "sfdgg",
			expectedResult: 30 * time.Second,
		},
		"resync interval is set to zero(0)": {
			resyncInterval: "0",
			expectedResult: 30 * time.Second,
		},
		"resync interval is correct": {
			resyncInterval: "13",
			expectedResult: 13 * time.Second,
		},
	}

	for name, mock := range tests {
		os.Setenv("RESYNC_INTERVAL", mock.resyncInterval)
		defer os.Unsetenv("RESYNC_INTERVAL")
		t.Run(name, func(t *testing.T) {
			interval := getSyncInterval()
			if interval != mock.expectedResult {
				t.Errorf("unable to get correct resync interval, expected: %v got %v", mock.expectedResult, interval)
			}
		})
	}
}

// TestCheckForCStorVolumeCRD validates if CStorVolume CRD operations
// can be done.
func TestCheckForCStorVolumeCRD(t *testing.T) {
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()
	done := make(chan bool)
	defer close(done)
	go func(done chan bool) {
		//CheckForCStorVolumeCR tries to find the volume CR and if is is not found
		// it will wait for 10 seconds and continue trying in the loop.
		// as we are already passing the fake CR, it has to find it immediately
		// if not, it means the code is not working properly
		CheckForCStorVolumeCRD(fakeOpenebsClient)
		//this below line will get executed only when CheckForCStorVolumeCR has
		//found the CR. Otherwise, the function will not return and we timeout
		// in the below select block and fail the testcase.
		done <- true
	}(done)

	select {
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout - CStorVolume is unknown")
	case <-done:
	}
}
