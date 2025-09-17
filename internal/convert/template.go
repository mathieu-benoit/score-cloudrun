// Copyright 2024 Score
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


const cloudRunServiceTemplate = `{{ define "cloudRunServiceTemplate" }}{{ $workloadName := .WorkloadName }}
resource "google_cloud_run_v2_service" "{{ $workloadName }}" {
  name     = "{{ $workloadName }}"
  location = "us-central1"
  deletion_protection = false
  ingress = "INGRESS_TRAFFIC_ALL"

  template {
    {{- range $containerName, $container := .Spec.Containers }}
    containers {
      name  = "{{ $containerName }}"
      image = "{{ $container.Image }}"
      
      {{- range $variableName, $variableValue := $container.Variables }}
      env {
        name: "{{ $variableName }}"
        value: "{{ $variableValue }}"
      }{{- end }}

      {{- if (gt (len $container.Command) 0) }}
      command = [
        {{- range $i, $cmd := $container.Command }}
        '{{ $cmd }}'{{ if (gt (len $container.Command) $i) }},{{ end }}
        {{- end }}
      ]
      {{- end }}

      {{- if (gt (len $container.Args) 0) }}
      args = [
        {{- range $i, $arg := $container.Args }}
        '{{ $arg }}'{{ if (gt (len $container.Args) $i) }},{{ end }}
        {{- end }}
      ]{{- end }}

      {{- if and (ne $container.LivenessProbe nil)  (ne $container.LivenessProbe.HttpGet nil) }}
      liveness_probe {
        http_get {
          {{- if (ne $container.LivenessProbe.HttpGet.Path "") }}
          path: '{{ $container.LivenessProbe.HttpGet.Path }}'
          {{- end }}
          {{- if (ne $container.LivenessProbe.HttpGet.Port 0) }}
          port: {{ $container.LivenessProbe.HttpGet.Port }}
          {{- end}}
        }
      }
      {{- end }}
    }
    {{- end }}
  }
}{{- end }}
`