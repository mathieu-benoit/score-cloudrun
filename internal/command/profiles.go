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
	"fmt"
	"log/slog"
	"sort"

	"github.com/spf13/cobra"

	"github.com/score-spec/score-cloudrun/internal/profiles"
	"github.com/score-spec/score-cloudrun/internal/state"
	"github.com/score-spec/score-go/formatter"
)

var (
	profilesGroup = &cobra.Command{
		Use:   "profiles",
		Short: "Subcommands related to workload profiles",
	}
	profilesList = &cobra.Command{
		Use:   "list [--format table|json]",
		Short: "List the workload profiles",
		Long: `The list command will list out the workload profiles. This requires an active score-cloudrun state
after 'init' has been run. The list of workload profiles will be empty if no profiles are defined.
`,
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		RunE:          listProfiles,
	}
)

func listProfiles(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	sd, ok, err := state.LoadStateDirectory(".")
	if err != nil {
		return fmt.Errorf("failed to load existing state directory: %w", err)
	} else if !ok {
		return fmt.Errorf("no state directory found, run 'score-cloudrun init' first")
	}
	slog.Debug(fmt.Sprintln("Listing workload profiles"))

	profiles, err := profiles.LoadProfilesFromDirectory(sd.Path, "")
	if err != nil {
		return fmt.Errorf("failed to load workload profiles in %s: %w", sd.Path, err)
	}

	if len(profiles) == 0 {
		slog.Info("No workload profiles found")
		return nil
	}

	outputFormat := cmd.Flag("format").Value.String()
	return displayProfiles(profiles, outputFormat)
}

func displayProfiles(profiles []profiles.Profile, outputFormat string) error {
	var outputFormatter formatter.OutputFormatter
	sortedProfiles := sortProfilesByName(profiles)

	switch outputFormat {
	case "json":
		type jsonData struct {
			Name        string
			Format      string
			Description string
		}
		var outputs []jsonData
		for _, provisioner := range sortedProfiles {
			outputs = append(outputs, jsonData{
				Name:        provisioner.Name,
				Format:      provisioner.Format,
				Description: provisioner.Description,
			})
		}
		outputFormatter = &formatter.JSONOutputFormatter[[]jsonData]{Data: outputs}
	default:
		rows := [][]string{}
		for _, provisioner := range sortedProfiles {
			rows = append(rows, []string{provisioner.Name, provisioner.Format, provisioner.Description})
		}
		headers := []string{"Name", "Format", "Description"}
		outputFormatter = &formatter.TableOutputFormatter{
			Headers: headers,
			Rows:    rows,
		}
	}
	return outputFormatter.Display()
}

func sortProfilesByName(profiles []profiles.Profile) []profiles.Profile {
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles
}

func init() {
	profilesList.Flags().StringP("format", "f", "table", "Format of the output: table (default), json")
	profilesGroup.AddCommand(profilesList)
	rootCmd.AddCommand(profilesGroup)
}
