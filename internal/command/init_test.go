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
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/score-spec/score-go/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-cloudrun/internal/profiles"
	"github.com/score-spec/score-cloudrun/internal/provisioners/loader"
	"github.com/score-spec/score-cloudrun/internal/state"
)

func TestInitNominal(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	stdout, stderr, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "score.yaml", "-p", "project-test", "-l", "location-test",
	})
	assert.NoError(t, err)
	assert.Equal(t, ``, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	sd, ok, err := state.LoadStateDirectory(".")
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, state.DefaultRelativeStateDirectory, sd.Path)
		assert.Len(t, sd.State.Workloads, 1)
		assert.Equal(t, map[framework.ResourceUid]framework.ScoreResourceState[state.ResourceExtras]{}, sd.State.Resources)
		assert.Equal(t, map[string]interface{}{}, sd.State.SharedState)
	}
}

func TestInitNominal_run_twice(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	// first init
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--file", "score2.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	// init again
	stdout, stderr, err = executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	_, err = os.Stat("score.yaml")
	assert.NoError(t, err)
	_, err = os.Stat("score2.yaml")
	assert.NoError(t, err)

	sd, ok, err := state.LoadStateDirectory(".")
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, state.DefaultRelativeStateDirectory, sd.Path)
		assert.Equal(t, map[string]framework.ScoreWorkloadState[state.WorkloadExtras]{}, sd.State.Workloads)
		assert.Equal(t, map[framework.ResourceUid]framework.ScoreResourceState[state.ResourceExtras]{}, sd.State.Resources)
		assert.Equal(t, map[string]interface{}{}, sd.State.SharedState)
	}
}

func TestInit_with_no_sample_provisioners_and_profiles(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	require.NoError(t, os.WriteFile(filepath.Join(td, "custom.provisioners.yaml"), []byte(`
- uri: template://test/env
  type: environment
  class: default
  format: terraform
  outputs:
    TOKEN: "ok"
`), 0644))

	require.NoError(t, os.WriteFile(filepath.Join(td, "custom.profiles.yaml"), []byte(`
- id: custom-default
  name: default
  format: terraform
  description: Custom default profile
  manifests: |
    # custom {{ .WorkloadName }}
`), 0644))

	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{
		"init", "--no-sample", "--provisioners", "custom.provisioners.yaml", "--profiles", "custom.profiles.yaml",
	})
	require.NoError(t, err)
	assert.Equal(t, "", stdout)

	_, err = os.Stat(filepath.Join(td, "score.yaml"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	_, err = os.Stat(filepath.Join(td, state.DefaultRelativeStateDirectory, profiles.DefaultProfileFileName))
	assert.ErrorIs(t, err, os.ErrNotExist)

	loadedProvisioners, err := loader.LoadProvisionersFromDirectory(state.DefaultRelativeStateDirectory, loader.ProvisionersFileSuffix, "terraform")
	require.NoError(t, err)
	require.Len(t, loadedProvisioners, 1)
	assert.Equal(t, "environment", loadedProvisioners[0].ResType)

	loadedProfiles, err := profiles.LoadProfilesFromDirectory(state.DefaultRelativeStateDirectory, "terraform")
	require.NoError(t, err)
	require.Len(t, loadedProfiles, 1)
	assert.Equal(t, "custom-default", loadedProfiles[0].Id)
}

func TestInit_custom_default_profile_for_other_format_keeps_builtin_default(t *testing.T) {
	td := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	require.NoError(t, os.WriteFile(filepath.Join(td, "kcc.profiles.yaml"), []byte(`
- id: kcc-default
  name: default
  format: kcc
  manifests: |
    # kcc {{ .WorkloadName }}
`), 0644))

	_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{
		"init", "--no-sample", "--profiles", "kcc.profiles.yaml",
	})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(td, state.DefaultRelativeStateDirectory, profiles.DefaultProfileFileName))
	assert.NoError(t, err, "built-in terraform default profile should still be written")
}

func TestInit_rerun_without_profiles_keeps_custom_default_profile(t *testing.T) {
	td := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	require.NoError(t, os.WriteFile(filepath.Join(td, "custom.profiles.yaml"), []byte(`
- id: custom-default
  name: default
  format: terraform
  manifests: |
    # custom {{ .WorkloadName }}
`), 0644))

	_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{
		"init", "--no-sample", "--profiles", "custom.profiles.yaml",
	})
	require.NoError(t, err)
	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--no-sample"})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(td, state.DefaultRelativeStateDirectory, profiles.DefaultProfileFileName))
	assert.ErrorIs(t, err, os.ErrNotExist)

	loadedProfiles, err := profiles.LoadProfilesFromDirectory(state.DefaultRelativeStateDirectory, "terraform")
	require.NoError(t, err)
	require.Len(t, loadedProfiles, 1)
	assert.Equal(t, "custom-default", loadedProfiles[0].Id)
}

func TestLoadFileOrURL_rejects_oversized_http_response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxSourceFileBytes+1))
	}))
	defer srv.Close()

	_, err := loadFileOrURL(context.Background(), srv.URL)
	assert.ErrorContains(t, err, "byte limit")
}
