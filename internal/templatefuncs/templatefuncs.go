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

// Package templatefuncs holds the functions available to profile and provisioner manifest templates.
package templatefuncs

import (
	"fmt"
	"strings"
	"text/template"
)

// FuncMap returns the functions registered on every manifest template.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"hclescape": HCLEscape,
	}
}

// HCLEscape escapes s so it can be placed between double quotes in an HCL string literal.
// It escapes backslashes, double quotes, and control characters. It intentionally leaves
// "${" and "%{" sequences untouched, because resource outputs rely on Terraform interpolation
// (for example "${google_sql_user.user.name}").
func HCLEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
