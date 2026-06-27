package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/state"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active engineering specification projects",
	Long:  `Scans the workspace directory and lists metadata, models, and confidence scores for all projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := state.ListProjects()
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No active projects found. Create one using 'synthspec init <project_name>'")
			return nil
		}

		const listFormat = "%-20s %-15s %-15s %-15s %-20s\n"

		// Print header
		fmt.Fprintf(cmd.OutOrStdout(), listFormat, "PROJECT NAME", "PROVIDER", "MODEL", "SCORES (F/S/S/C)", "LAST UPDATED")
		fmt.Fprintf(cmd.OutOrStdout(), listFormat, "------------", "--------", "-----", "----------------", "------------")

		for _, name := range projects {
			sess, err := state.LoadSession(name)
			if err != nil {
				// Skip corrupt sessions
				continue
			}
			scoreStr := fmt.Sprintf("%d/%d/%d/%d", sess.Scores.Functional, sess.Scores.Structural, sess.Scores.Security, sess.Scores.Compliance)
			updatedStr := sess.UpdatedAt.Format("2006-01-02 15:04:05")
			fmt.Fprintf(cmd.OutOrStdout(), listFormat, sess.ProjectName, sess.Provider, sess.Model, scoreStr, updatedStr)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
