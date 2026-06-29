package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui"
)

var updateCmd = &cobra.Command{
	Use:   "update [project_name]",
	Short: "Add new requirements or modify an existing specification",
	Long:  `Loads a saved project session and prompts directly in the TUI to input new requirements or modifications.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := resolveProjectName(args, "update")
		if err != nil {
			return err
		}

		// 1. Load session progress
		sess, err := state.LoadSession(projectName)
		if err != nil {
			return fmt.Errorf("failed to load project session '%s': %w", projectName, err)
		}

		// Allow overriding provider/model on update if explicit global flags are passed
		if providerFlag != "" {
			sess.Provider = providerFlag
		}
		if modelFlag != "" {
			sess.Model = modelFlag
		}

		// 2. Setup Gateway
		gw, err := NewGatewayForSession(sess, mockFlag)
		if err != nil {
			return err
		}

		// 3. Boot Dashboard
		fmt.Printf("Loading project '%s' for updates...\n", projectName)
		m := tui.NewDashboardModel(sess, gw, outputFlag)
		m.StartWithUpdatePrompt()
		if err := runTUI(m); err != nil {
			return fmt.Errorf("bubbletea execution failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
