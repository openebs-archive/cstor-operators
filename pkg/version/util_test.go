/*
Copyright 2021 The OpenEBS Authors
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
package version

import "testing"

func TestIsCurrentVersionValid(t *testing.T) {
	// setting the variable for test
	validDesiredVersion = "2.9.0"
	type args struct {
		v string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Valid Current Version",
			args: args{
				v: "1.12.0",
			},
			want: true,
		},
		{
			name: "Less than Min Current Version",
			args: args{
				v: "1.9.0",
			},
			want: false,
		},
		{
			name: "More than Valid Desired Version",
			args: args{
				v: "2.13.0",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCurrentVersionValid(tt.args.v); got != tt.want {
				t.Errorf("IsCurrentVersionValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
