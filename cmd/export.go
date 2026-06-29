package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/generator/export"
	"github.com/toanle/synthspec/state"
)

var exportDestFlag string

var exportCmd = &cobra.Command{
	Use:     "export [project_name]",
	Aliases: []string{"compile"},
	Short:   "Export generated specifications into a standalone searchable HTML site",
	Long:    `Compiles all markdown documents and metadata from the specified project output directory into a responsive, self-contained HTML page with fuzzy search and Mermaid layout.`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := resolveProjectName(args, "export")
		if err != nil {
			return err
		}

		// 1. Verify session exists
		sess, err := state.LoadSession(projectName)
		if err != nil {
			return fmt.Errorf("failed to load project session '%s': %w", projectName, err)
		}

		// 2. Determine target directories
		projDir := state.GetSessionDir(sess.ProjectName)
		outputDir := filepath.Join(projDir, "output")

		distDir := exportDestFlag
		if distDir == "" {
			distDir = filepath.Join(projDir, "dist")
		}

		// 3. Export to HTML
		fmt.Fprintf(cmd.OutOrStdout(), "Exporting specifications for project '%s'...\n", projectName)
		indexPath, err := export.ExportToHTML(projectName, outputDir, distDir)
		if err != nil {
			state.LogError(projectName, err)
			return fmt.Errorf("export failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully exported specifications to: %s\n", indexPath)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportDestFlag, "dest", "d", "", "Custom destination directory to save the index.html file (defaults to synthspec/<project_name>/dist)")
	rootCmd.AddCommand(exportCmd)
}
