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
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/score-spec/score-go/framework"
	scoretypes "github.com/score-spec/score-go/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-cloudrun/internal/profiles"
	"github.com/score-spec/score-cloudrun/internal/provisioners/loader"
	"github.com/score-spec/score-cloudrun/internal/state"
)

const (
	initCmdFileFlag             = "file"
	initCmdNoDefaultProfileFlag = "no-default-profile"
	initCmdNoSampleFlag         = "no-sample"
	initCmdProvisionersFlag     = "provisioners"
	initCmdProfilesFlag         = "profiles"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise the local state directory and sample score file",
	Args:  cobra.NoArgs,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		sd, ok, err := state.LoadStateDirectory(".")
		if err != nil {
			return fmt.Errorf("failed to load existing state directory: %w", err)
		} else if ok {
			slog.Info("Found existing state directory", "dir", sd.Path)
		} else {
			slog.Info("Writing new state directory", "dir", state.DefaultRelativeStateDirectory)
			sd = &state.StateDirectory{
				Path: state.DefaultRelativeStateDirectory,
				State: state.State{
					Workloads:   map[string]framework.ScoreWorkloadState[state.WorkloadExtras]{},
					Resources:   map[framework.ResourceUid]framework.ScoreResourceState[state.ResourceExtras]{},
					SharedState: map[string]interface{}{},
				},
			}
			slog.Info("Writing new state directory", "dir", sd.Path)
			if err := sd.Persist(); err != nil {
				return fmt.Errorf("failed to persist new state directory: %w", err)
			}
		}

		// create and write the default profile file if it doesn't already exist
		disableDefaultProfile, err := cmd.Flags().GetBool(initCmdNoDefaultProfileFlag)
		if err != nil {
			return fmt.Errorf("failed to parse --%s flag: %w", initCmdNoDefaultProfileFlag, err)
		}

		provisionerSources, err := cmd.Flags().GetStringArray(initCmdProvisionersFlag)
		if err != nil {
			return fmt.Errorf("failed to parse --%s flag: %w", initCmdProvisionersFlag, err)
		}
		for _, src := range provisionerSources {
			raw, err := loadFileOrURL(cmd.Context(), src)
			if err != nil {
				return fmt.Errorf("failed to load provisioner '%s': %w", src, err)
			}
			if err := loader.SaveProvisionerToDirectory(sd.Path, src, raw); err != nil {
				return fmt.Errorf("failed to save provisioner '%s': %w", src, err)
			}
		}

		profileSources, err := cmd.Flags().GetStringArray(initCmdProfilesFlag)
		if err != nil {
			return fmt.Errorf("failed to parse --%s flag: %w", initCmdProfilesFlag, err)
		}
		for _, src := range profileSources {
			raw, err := loadFileOrURL(cmd.Context(), src)
			if err != nil {
				return fmt.Errorf("failed to load profile '%s': %w", src, err)
			}
			if err := profiles.SaveProfileToDirectory(sd.Path, src, raw); err != nil {
				return fmt.Errorf("failed to save profile '%s': %w", src, err)
			}
			if profiles.HasDefaultProfile(raw) {
				disableDefaultProfile = true
			}
		}

		if !disableDefaultProfile {
			defaultProfile := filepath.Join(sd.Path, profiles.DefaultProfileFileName)

			if _, err := os.Stat(defaultProfile); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("failed to check for existing default workload profile file: %w", err)
				}

				f, err := os.OpenFile(defaultProfile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
				if err != nil {
					return fmt.Errorf("failed to create default workload profile file: %w", err)
				}

				defer func(f *os.File) {
					if f == nil {
						return
					}
					err = errors.Join(err, f.Close())
				}(f)

				slog.Info("Writing default workload profile file", "path", defaultProfile)
				if _, err := f.WriteString(profiles.GetDefaultProfileContent()); err != nil {
					return fmt.Errorf("failed to write default workload profile content: %w", err)
				}
			} else {
				slog.Info("Default workload profile file already exists, skipping", "path", defaultProfile)
			}
		} else {
			slog.Info("Skipping default workload profile due to --no-default-profile flag")
		}

		noSample, err := cmd.Flags().GetBool(initCmdNoSampleFlag)
		if err != nil {
			return fmt.Errorf("failed to parse --%s flag: %w", initCmdNoSampleFlag, err)
		}

		initCmdScoreFile, _ := cmd.Flags().GetString(initCmdFileFlag)
		if noSample {
			slog.Info("Skipping creation of initial Score file due to --no-sample flag")
		} else if _, err := os.Stat(initCmdScoreFile); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to check for existing Score file: %w", err)
			}
			workload := &scoretypes.Workload{
				ApiVersion: "score.dev/v1b1",
				Metadata: map[string]interface{}{
					"name": "example",
				},
				Containers: map[string]scoretypes.Container{
					"main": {
						Image: "stefanprodan/podinfo",
					},
				},
				Service: &scoretypes.WorkloadService{
					Ports: map[string]scoretypes.ServicePort{
						"web": {Port: 8080},
					},
				},
			}
			rawScore, _ := yaml.Marshal(workload)
			if err := os.WriteFile(initCmdScoreFile, rawScore, 0755); err != nil {
				return fmt.Errorf("failed to write Score file: %w", err)
			}
			slog.Info("Created initial Score file", "file", initCmdScoreFile)
		} else {
			slog.Info("Skipping creation of initial Score file since it already exists", "file", initCmdScoreFile)
		}

		return nil
	},
}

func loadFileOrURL(ctx context.Context, src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected HTTP status %s", resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(src)
}

func init() {
	initCmd.Flags().StringP(initCmdFileFlag, "f", "score.yaml", "The score file to initialize")
	initCmd.Flags().Bool(initCmdNoDefaultProfileFlag, false, "Disable generation of the default workload profile file")
	initCmd.Flags().Bool(initCmdNoSampleFlag, false, "Disable generation of the sample score file")
	initCmd.Flags().StringArray(initCmdProvisionersFlag, nil, "Provisioner files or URLs to load into the state directory")
	initCmd.Flags().StringArray(initCmdProfilesFlag, nil, "Workload profile files or URLs to load into the state directory")
	rootCmd.AddCommand(initCmd)
}
