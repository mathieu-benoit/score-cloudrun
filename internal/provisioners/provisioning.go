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

	"github.com/score-spec/score-go/framework"

	"github.com/score-spec/score-cloudrun/internal/state"
)

func ProvisionResources(currentState *state.State, location string) (string, *state.State, error) {
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

		// ==========================================================================================
		// TODO: HERE IS WHERE YOU WOULD USE THE RESOURCE TYPE, CLASS, ID, AND PARAMS TO PROVISION IT
		// ==========================================================================================

		resState.Outputs = map[string]interface{}{}
		var resourceTemplate = ""
		var resourceTemplateName = ""

		if resUid.Type() == "redis" {
			resState.Outputs["host"] = "${google_redis_instance.memorystore.host}"
			resState.Outputs["port"] = "${google_redis_instance.memorystore.port}"
			resState.Outputs["username"] = ""
			resState.Outputs["password"] = "${google_redis_instance.memorystore.auth_string}"
			resourceTemplateName = "redisTemplate"
			resourceTemplate = redisTemplate
		} else if resUid.Type() == "postgres" {
			resState.Outputs["host"] = "${google_sql_database_instance.instance.name}"
			resState.Outputs["database"] = "${google_sql_database.database.name}"
			resState.Outputs["port"] = 5432
			resState.Outputs["username"] = "${google_sql_user.user.name}"
			resState.Outputs["password"] = "${google_sql_user.user.password}"
			resourceTemplateName = "postgresTemplate"
			resourceTemplate = postgresTemplate
		} else if resUid.Type() == "dns" {
			resState.Outputs["host"] = "${google_cloud_run_v2_service.example.uri}"
			resourceTemplateName = "dnsTemplate"
			resourceTemplate = dnsTemplate
		}

		var resourceManifest = ""
		resourceManifest, err = generateResource(resourceTemplateName, resourceTemplate, location)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate resource %s: %w", resUid.Type(), err)
		}
		slog.Info(fmt.Sprintf("Resource %s's manifests generated", resUid.Type()))

		out.Resources[resUid] = resState
		manifests = manifests + resourceManifest
	}

	return manifests, out, nil
}

func generateResource(resourceTypeTemplateName string, resourceTypeTemplate string, location string) (string, error) {
	t, err := template.New(resourceTypeTemplateName).Parse(resourceTypeTemplate)
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
