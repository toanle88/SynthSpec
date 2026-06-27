package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/state"
)

var forceFlag bool

var deleteCmd = &cobra.Command{
	Use:   "delete [project_name]",
	Short: "Delete an existing engineering specification project",
	Long:  `Removes a project's session file and all of its generated output files recursively.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		dir := state.GetSessionDir(projectName)

		// Check if directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("project '%s' does not exist", projectName)
		}

		if !forceFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete project '%s' and all generated files? (y/N): ", projectName)
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read confirmation input: %w", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Deletion cancelled.")
				return nil
			}
		}

		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to delete project directory: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Project '%s' deleted successfully.\n", projectName)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force deletion without prompting for confirmation")
	rootCmd.AddCommand(deleteCmd)
}
