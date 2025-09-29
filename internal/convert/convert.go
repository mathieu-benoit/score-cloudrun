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

package convert

import (
	"bytes"
	"fmt"
	"html/template"
	"maps"
	"os"
	"path/filepath"

	"github.com/score-spec/score-go/framework"
	scoretypes "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-cloudrun/internal/profiles"
	"github.com/score-spec/score-cloudrun/internal/state"
)

func Workload(workloadProfile profiles.Profile, currentState *state.State, workloadName string, location string, projectId string) (string, error) {
	resOutputs, err := currentState.GetResourceOutputForWorkload(workloadName)
	if err != nil {
		return "", fmt.Errorf("failed to generate outputs: %w", err)
	}
	sf := framework.BuildSubstitutionFunction(currentState.Workloads[workloadName].Spec.Metadata, resOutputs)

	spec := currentState.Workloads[workloadName].Spec
	containers := maps.Clone(spec.Containers)
	for containerName, container := range containers {
		if container.Variables, err = convertContainerVariables(container.Variables, sf); err != nil {
			return "", fmt.Errorf("workload: %s: container: %s: variables: %w", workloadName, containerName, err)
		}

		if container.Files, err = convertContainerFiles(container.Files, currentState.Workloads[workloadName].File, sf); err != nil {
			return "", fmt.Errorf("workload: %s: container: %s: files: %w", workloadName, containerName, err)
		}
		containers[containerName] = container
	}
	spec.Containers = containers
	resources := maps.Clone(spec.Resources)
	for resName, res := range resources {
		resUid := framework.NewResourceUid(workloadName, resName, res.Type, res.Class, res.Id)
		resState, ok := currentState.Resources[resUid]
		if !ok {
			return "", fmt.Errorf("workload '%s': resource '%s' (%s) is not primed", workloadName, resName, resUid)
		}
		res.Params = resState.Params
		resources[resName] = res
	}
	spec.Resources = resources

	convertedWorkloadManifests, err := convertWithWorkloadProfile(workloadProfile, spec, workloadName, location, projectId)
	if err != nil {
		return "", fmt.Errorf("workload: %s: failed to convert to Terraform: %w", workloadName, err)
	}

	return convertedWorkloadManifests, nil
}

// convertWithWorkloadProfile converts the Score workload with the associated Workload Profile
func convertWithWorkloadProfile(profile profiles.Profile, spec scoretypes.Workload, workloadName string, location string, projectId string) (string, error) {
	fileContent, err := generateWorkloadManifests(profile, spec, workloadName, location, projectId)
	if err != nil {
		return "", fmt.Errorf("failed to generate cloud run service: %w", err)
	}

	return fileContent, nil
}

func generateWorkloadManifests(profile profiles.Profile, spec scoretypes.Workload, workloadName string, location string, projectId string) (string, error) {
	t, err := template.New("template").Parse(profile.ManifestsTemplate)
	if err != nil {
		return "", err
	}
	// Create a buffer to hold the generated cloud run service
	var buf bytes.Buffer

	data := struct {
		WorkloadName string
		Spec         scoretypes.Workload
		Location     string
		ProjectId    string
	}{
		WorkloadName: workloadName,
		Spec:         spec,
		Location:     location,
		ProjectId:    projectId,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func convertContainerVariables(input scoretypes.ContainerVariables, sf func(string) (string, error)) (map[string]string, error) {
	outMap := make(map[string]string, len(input))
	for key, value := range input {
		out, err := framework.SubstituteString(value, sf)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		outMap[key] = out
	}
	return outMap, nil
}

func convertContainerFiles(input map[string]scoretypes.ContainerFile, scoreFile *string, sf func(string) (string, error)) (map[string]scoretypes.ContainerFile, error) {
	output := make(map[string]scoretypes.ContainerFile, len(input))
	for target, file := range input {
		var content string
		if file.Content != nil {
			content = *file.Content
		} else if file.Source != nil {
			sourcePath := *file.Source
			if !filepath.IsAbs(sourcePath) && scoreFile != nil {
				sourcePath = filepath.Join(filepath.Dir(*scoreFile), sourcePath)
			}
			if rawContent, err := os.ReadFile(sourcePath); err != nil {
				return nil, fmt.Errorf("%s: source: failed to read file '%s': %w", target, sourcePath, err)
			} else {
				content = string(rawContent)
			}
		} else {
			return nil, fmt.Errorf("%s: missing 'content' or 'source'", target)
		}

		var err error
		if file.NoExpand == nil || !*file.NoExpand {
			content, err = framework.SubstituteString(string(content), sf)
			if err != nil {
				return nil, fmt.Errorf("%s: failed to substitute in content: %w", target, err)
			}
		}
		file.Source = nil
		file.Content = &content
		bTrue := true
		file.NoExpand = &bTrue
		output[target] = file
	}
	return output, nil
}
