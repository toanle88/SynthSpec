package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui"
)

var resumeCmd = &cobra.Command{
	Use:   "resume [project_name]",
	Short: "Resume an existing engineering specification session",
	Long:  `Loads a saved project session from disk and returns to the interactive TUI interrogation dashboard.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string

		if len(args) == 0 {
			// Auto-detect projects
			projects, err := state.ListProjects()
			if err != nil {
				return fmt.Errorf("failed to scan for projects: %w", err)
			}

			if len(projects) == 0 {
				return fmt.Errorf("no active projects found to resume. Start one using 'synthspec init <project_name>'")
			}

			if len(projects) > 1 {
				fmt.Println("Multiple active projects found. Please select one to resume:")
				for _, p := range projects {
					fmt.Printf(" - %s\n", p)
				}
				return fmt.Errorf("use 'synthspec resume [project_name]' to specify which project to load")
			}

			projectName = projects[0]
		} else {
			projectName = args[0]
		}

		// 1. Load session progress
		sess, err := state.LoadSession(projectName)
		if err != nil {
			return fmt.Errorf("failed to load project session '%s': %w", projectName, err)
		}

		// Allow overriding provider/model on resume if explicit global flags are passed
		if providerFlag != "" {
			sess.Provider = providerFlag
		}
		if modelFlag != "" {
			sess.Model = modelFlag
		}

		// 2. Setup Gateway
		gw, err := getGatewayForSession(sess, mockFlag)
		if err != nil {
			return err
		}

		// 3. Boot Dashboard
		settings, _ := config.LoadSettings()
		outDir := outputFlag
		if outDir == "" && settings != nil {
			outDir = settings.DefaultOutputFolder
		}
		// Default to project-specific output directory if not explicitly set
		if outDir == "" || outDir == "./output" || outDir == config.DefaultOutputFolderValue {
			outDir = filepath.Join(state.GetSessionDir(projectName), "output")
		}

		fmt.Printf("Resuming project '%s' using %s (%s)...\n", projectName, sess.Provider, sess.Model)
		m := tui.NewDashboardModel(sess, gw, outDir)
		if err := runTUI(m); err != nil {
			return fmt.Errorf("bubbletea execution failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
