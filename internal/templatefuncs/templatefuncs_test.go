// Copyright 2025 The Score Authors.
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

package templatefuncs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHCLEscape(t *testing.T) {
	for _, tc := range []struct {
		name, in, out string
	}{
		{"plain", "nginx:latest", "nginx:latest"},
		{"html chars untouched", `a<b && c>d`, `a<b && c>d`},
		{"double quote", `say "hi"`, `say \"hi\"`},
		{"backslash", `\d+`, `\\d+`},
		{"newline and tab", "a\nb\tc\r", `a\nb\tc\r`},
		{"control char", "a\x00b\x7f", `a\u0000b\u007F`},
		{"terraform interpolation preserved", "${google_sql_user.user.name}", "${google_sql_user.user.name}"},
		{"breakout attempt", `x" } resource "evil" "e" { a = "`, `x\" } resource \"evil\" \"e\" { a = \"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.out, HCLEscape(tc.in))
		})
	}
}
