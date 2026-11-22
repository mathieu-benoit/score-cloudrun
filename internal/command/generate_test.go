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

package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-cloudrun/internal/state"
)

func changeToDir(t *testing.T, dir string) string {
	t.Helper()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})
	return dir
}

func changeToTempDir(t *testing.T) string {
	return changeToDir(t, t.TempDir())
}

func TestGenerateWithoutInit(t *testing.T) {
	_ = changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"generate"})
	assert.EqualError(t, err, "state directory does not exist, please run \"init\" first")
	assert.Equal(t, "", stdout)
}

func TestGenerateWithoutScoreFiles(t *testing.T) {
	_ = changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate"})
	assert.EqualError(t, err, "project is empty, please add a score file")
	assert.Equal(t, "", stdout)
}

func TestGenerateWithoutLocation(t *testing.T) {
	_ = changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "score.yaml",
	})
	assert.EqualError(t, err, "no Google Cloud location specified, please use --location")
	assert.Equal(t, "", stdout)
}

func TestGenerateWithoutProjectId(t *testing.T) {
	_ = changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "score.yaml", "--location", "location-test",
	})
	assert.EqualError(t, err, "no Google Cloud Project ID specified, please use --project-id")
	assert.Equal(t, "", stdout)
}

func TestInitAndGenerateWithBadFile(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	assert.NoError(t, os.WriteFile(filepath.Join(td, "thing"), []byte(`"blah"`), 0644))

	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "thing"})
	assert.EqualError(t, err, "failed to decode input score file: thing: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `blah` into map[string]interface {}")
	assert.Equal(t, "", stdout)
}

func TestInitAndGenerateWithBadScore(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	assert.NoError(t, os.WriteFile(filepath.Join(td, "thing"), []byte(`{}`), 0644))

	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "thing"})
	assert.EqualError(t, err, "invalid score file: thing: jsonschema: '' does not validate with https://score.dev/schemas/score#/required: missing properties: 'apiVersion', 'metadata', 'containers'")
	assert.Equal(t, "", stdout)
}

func TestInitAndGenerate_with_sample(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	require.NoError(t, err)
	assert.Equal(t, "", stdout)
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "manifests.tf", "-p", "project-test", "-l", "location-test", "--", "score.yaml",
	})
	require.NoError(t, err)
	assert.Equal(t, "", stdout)
	raw, err := os.ReadFile(filepath.Join(td, "manifests.tf"))
	assert.NoError(t, err)
	assert.Equal(t, `
locals {
  network = "default"
}
resource "google_cloud_run_v2_service" "example" {
  provider = google-beta
  project = "project-test"
  location = "location-test"
  name     = "example"
  deletion_protection = false
  ingress = "INGRESS_TRAFFIC_ALL"
  default_uri_disabled = true

  template {
    service_account = google_service_account.example.email
    containers {
      name  = "main"
      image = "stefanprodan/podinfo"
      ports {
        container_port = 8080
      }
    }
    vpc_access {
      network_interfaces {
        network    = local.network
        subnetwork = local.network
      }
    }
  }
}

resource "google_service_account" "example" {
  project = "project-test"
  account_id = "example"
}`, string(raw))

	// check that state was persisted
	sd, ok, err := state.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "score.yaml", *sd.State.Workloads["example"].File)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 0)
}

func TestInitAndGenerate_with_full_example(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	require.NoError(t, err)
	assert.Equal(t, "", stdout)

	_ = os.Remove(filepath.Join(td, "score.yaml"))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: stefanprodan/podinfo
    command:
      - "node"
    args:
      - packages/backend
      - "--config"
      - app-config.yaml
    variables:
      key: value
      dynamic: ${metadata.name}
    resources:
      limits:
        memory: "128Mi"
        cpu: "500m"
      requests:
        memory: "64Mi"
        cpu: "250m"
    livenessProbe:
      httpGet:
        scheme: HTTP
        path: /alive
        port: 8080
    readinessProbe:
      httpGet:
        scheme: HTTP
        path: /ready
        port: 8080
    files:
      /somefile:
        content: |
          ${metadata.name}
service:
  ports:
    tcp:
      port: 8080
      targetPort: 8080
resources:
  thing:
    type: something
    params:
      x: ${metadata.name}
`), 0755))

	assert.NoError(t, os.WriteFile(filepath.Join(td, ".score-cloudrun", "something.provisioners.yaml"), []byte(`
- uri: default://something
  type: something
  class: default
  format: terraform
`), 0644))

	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "manifests.tf", "-p", "project-test", "-l", "location-test", "--", "score.yaml",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(td, "manifests.tf"))
	assert.NoError(t, err)
	assert.Equal(t, `
locals {
  network = "default"
}
resource "google_cloud_run_v2_service" "example" {
  provider = google-beta
  project = "project-test"
  location = "location-test"
  name     = "example"
  deletion_protection = false
  ingress = "INGRESS_TRAFFIC_ALL"
  default_uri_disabled = true

  template {
    service_account = google_service_account.example.email
    containers {
      name  = "main"
      image = "stefanprodan/podinfo"
      ports {
        container_port = 8080
      }
      env {
        name = "dynamic"
        value = "example"
      }
      env {
        name = "key"
        value = "value"
      }
      command = [
        "node",
      ]
      args = [
        "packages/backend",
        "--config",
        "app-config.yaml",
      ]
      liveness_probe {
        http_get {
          path = "/alive"
          port = 8080
        }
      }
    }
    vpc_access {
      network_interfaces {
        network    = local.network
        subnetwork = local.network
      }
    }
  }
}

resource "google_service_account" "example" {
  project = "project-test"
  account_id = "example"
}
`, string(raw))
}

func TestGenerate_does_not_html_escape_variables_or_templates(t *testing.T) {
	td := changeToTempDir(t)
	_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	require.NoError(t, err)

	_ = os.Remove(filepath.Join(td, "score.yaml"))
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: nginx:latest
    variables:
      DATABASE_URL: "postgres://user:pass@10.0.0.2:5432/app?sslmode=require&connect_timeout=5"
      LOG_FILTER: "severity>=WARNING && latency<500"
resources:
  custom:
    type: custom
`), 0644))

	require.NoError(t, os.WriteFile(filepath.Join(td, ".score-cloudrun", "custom.provisioners.yaml"), []byte(`
- uri: template://test/custom
  type: custom
  class: default
  format: terraform
  manifests: |
    # Special characters like <, >, &, and " must not be HTML-escaped in HCL.
    locals {
      condition = (1 > 0 && 2 < 3) ? "yes" : "no"
    }
`), 0644))

	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "manifests.tf", "-p", "project-test", "-l", "us-east4", "--", "score.yaml",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(td, "manifests.tf"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, `value = "postgres://user:pass@10.0.0.2:5432/app?sslmode=require&connect_timeout=5"`)
	assert.Contains(t, content, `value = "severity>=WARNING && latency<500"`)
	assert.Contains(t, content, `condition = (1 > 0 && 2 < 3) ? "yes" : "no"`)
	assert.NotContains(t, content, "&lt;")
	assert.NotContains(t, content, "&gt;")
	assert.NotContains(t, content, "&amp;")
	assert.NotContains(t, content, "&#34;")
}

func TestGenerate_escapes_hcl_string_values(t *testing.T) {
	td := changeToTempDir(t)
	_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	require.NoError(t, err)

	_ = os.Remove(filepath.Join(td, "score.yaml"))
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  main:
    image: nginx:latest
    command: ["sh", "-c"]
    args: ["echo \"hi\" && exit 0"]
    variables:
      QUOTED: 'say "hi"'
      BREAKOUT: 'x" } } resource "google_project_iam_member" "pwn" { role = "'
      PATTERN: '\d+'
      MULTILINE: "line1\nline2"
      INTERPOLATED: '$${google_sql_user.user.name}'
`), 0644))

	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "manifests.tf", "-p", "project-test", "-l", "us-east4", "--", "score.yaml",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(td, "manifests.tf"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, `value = "say \"hi\""`)
	assert.Contains(t, content, `value = "x\" } } resource \"google_project_iam_member\" \"pwn\" { role = \""`)
	assert.NotContains(t, content, `resource "google_project_iam_member"`)
	assert.Contains(t, content, `value = "\\d+"`)
	assert.Contains(t, content, `value = "line1\nline2"`)
	assert.Contains(t, content, `value = "${google_sql_user.user.name}"`)
	assert.Contains(t, content, `"echo \"hi\" && exit 0"`)
}

func TestGenerate_skips_reserved_port_env_and_uses_target_port(t *testing.T) {
	td := changeToTempDir(t)
	_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	require.NoError(t, err)

	_ = os.Remove(filepath.Join(td, "score.yaml"))
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: web
containers:
  app:
    image: nginx:latest
    variables:
      PORT: "3000"
      APP_MODE: production
service:
  ports:
    web:
      port: 80
      targetPort: 3000
`), 0644))

	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "manifests.tf", "-p", "project-test", "-l", "us-east4", "--", "score.yaml",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(td, "manifests.tf"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "container_port = 3000")
	assert.Contains(t, content, `name = "APP_MODE"`)
	assert.NotContains(t, content, `name = "PORT"`)
}

func TestGenerate_route_resource_enables_default_uri(t *testing.T) {
	routeProvisioners, err := filepath.Abs("../../examples/provisioners/route-terraform.provisioners.yaml")
	require.NoError(t, err)
	td := changeToTempDir(t)
	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--no-sample", "--provisioners", routeProvisioners})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: web
containers:
  app:
    image: nginx:latest
resources:
  dns:
    type: route
    params:
      host: example.com
      path: /
      port: 8080
`), 0644))

	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "manifests.tf", "-p", "project-test", "-l", "us-east4", "--", "score.yaml",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(td, "manifests.tf"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "provider = google\n")
	assert.NotContains(t, content, "default_uri_disabled")
}
