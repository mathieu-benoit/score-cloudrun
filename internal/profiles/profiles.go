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

package profiles

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	Id                string `yaml:"id"`
	Description       string `yaml:"description,omitempty"`
	ManifestsTemplate string `yaml:"manifests,omitempty"`
}

const ProfilesFileSuffix = ".profiles.yaml"

// LoadProfilesFromDirectory loads all workload profiles we can find in files that end in the common suffix.
func LoadProfilesFromDirectory(path string) ([]Profile, error) {
	slog.Debug(fmt.Sprintf("Loading provisioners with suffix %s in directory '%s'", ProfilesFileSuffix, path))
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0)
	for _, item := range items {
		if !item.IsDir() && strings.HasSuffix(item.Name(), ProfilesFileSuffix) {
			raw, err := os.ReadFile(filepath.Join(path, item.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read '%s': %w", item.Name(), err)
			}
			p, err := LoadProfiles(raw)
			if err != nil {
				return nil, fmt.Errorf("failed to load '%s': %w", item.Name(), err)
			}
			out = append(out, p...)
		}
	}
	return out, nil
}

// LoadProfiles loads a list of provisioners from the raw contents from a yaml file.
func LoadProfiles(raw []byte) ([]Profile, error) {
	var intermediate []map[string]interface{}
	if err := yaml.NewDecoder(bytes.NewReader(raw)).Decode(&intermediate); err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}
	out := make([]Profile, 0, len(intermediate))
	for _, item := range intermediate {
		profile := Profile{}
		intermediate, _ := yaml.Marshal(item)
		dec := yaml.NewDecoder(bytes.NewReader(intermediate))
		dec.KnownFields(true)
		if err := dec.Decode(&profile); err != nil {
			slog.Info("YAML reader error", "error", err)
			return nil, err
		}
		if profile.Id == "" {
			return nil, fmt.Errorf("id not set")
		}

		slog.Debug(fmt.Sprintf("Loaded profile %s", profile.Id))
		out = append(out, profile)
	}
	return out, nil
}
