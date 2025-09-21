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

package provisioners

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"maps"
	"slices"

	"github.com/score-spec/score-go/framework"

	"github.com/score-spec/score-cloudrun/internal/state"
)

type Provisioner struct {
	Uri               string  `yaml:"uri"`
	ResType           string  `yaml:"type"`
	Class             *string `yaml:"class,omitempty"`
	Description       string  `yaml:"description,omitempty"`
	ManifestsTemplate string  `yaml:"manifests,omitempty"`
	// Inputs is a list of inputs that the provisioner expects to be passed in.
	Inputs map[string]interface{} `yaml:"inputs,omitempty"`
	// Outputs is a list of outputs that the provisioner should return.
	Outputs map[string]interface{} `yaml:"outputs,omitempty"`
}

func ProvisionResources(currentState *state.State, provisioners []Provisioner, location string) (string, *state.State, error) {
	out := currentState
	manifests := ""

	// provision in sorted order
	orderedResources, err := currentState.GetSortedResourceUids()
	if err != nil {
		return "", nil, fmt.Errorf("failed to determine sort order for provisioning: %w", err)
	}

	out.Resources = maps.Clone(out.Resources)
	for _, resUid := range orderedResources {
		resState := out.Resources[resUid]

		provisionerIndex := slices.IndexFunc(provisioners, func(provisioner Provisioner) bool {
			return true
			//return provisioner.Id == resUid
		})
		if provisionerIndex < 0 {
			return "", nil, fmt.Errorf("resource '%s' is not supported by any provisioner. "+
				"Please implement a custom resource provisioner to support this resource type.", resUid)
		}

		var params map[string]interface{}
		if len(resState.Params) > 0 {
			resOutputs, err := out.GetResourceOutputForWorkload(resState.SourceWorkload)
			if err != nil {
				return "", nil, fmt.Errorf("%s: failed to find resource params for resource: %w", resUid, err)
			}
			sf := framework.BuildSubstitutionFunction(out.Workloads[resState.SourceWorkload].Spec.Metadata, resOutputs)
			rawParams, err := framework.Substitute(resState.Params, sf)
			if err != nil {
				return "", nil, fmt.Errorf("%s: failed to substitute params for resource: %w", resUid, err)
			}
			params = rawParams.(map[string]interface{})
		}
		resState.Params = params

		resState.ProvisionerUri = provisioners[provisionerIndex].Uri

		if provisioners[provisionerIndex].Outputs != nil {
			resState.Outputs = provisioners[provisionerIndex].Outputs
		} else {
			resState.Outputs = make(map[string]interface{})
		}

		var resourceManifest string
		resourceManifest, err = generateResourceManifest(provisioners[provisionerIndex].ManifestsTemplate, location)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate resource manifest %s: %w", resUid.Type(), err)
		}
		slog.Info(fmt.Sprintf("Resource %s's manifests generated", resUid.Type()))

		out.Resources[resUid] = resState
		resourceManifest = "\n" + resourceManifest
		manifests = manifests + resourceManifest
	}

	return manifests, out, nil
}

func generateResourceManifest(resourceTypeTemplate string, location string) (string, error) {
	t, err := template.New("template").Parse(resourceTypeTemplate)
	if err != nil {
		return "", err
	}
	// Create a buffer to hold the generated cloud run service
	var buf bytes.Buffer

	data := struct {
		Location string
	}{
		Location: location,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
