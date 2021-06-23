// Copyright 2020 The OpenEBS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import "testing"

func TestIsCurrentLessThanNewVersion(t *testing.T) {
	type args struct {
		old string
		new string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "old is less than new",
			args: args{
				old: "1.12.0",
				new: "2.8.0",
			},
			want: true,
		},
		{
			name: "old is greater than new",
			args: args{
				old: "2.10.0-RC2",
				new: "2.8.0",
			},
			want: false,
		},
		{
			name: "old is same as new",
			args: args{
				old: "2.8.0",
				new: "2.8.0",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCurrentLessThanNewVersion(tt.args.old, tt.args.new); got != tt.want {
				t.Errorf("IsCurrentLessThanNewVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
