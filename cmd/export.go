package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/generator/export"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
)

var exportDestFlag string
var exportFormatFlag string

var exportCmd = &cobra.Command{
	Use:     "export [project_name]",
	Aliases: []string{"compile"},
	Short:   "Export generated specifications into HTML, Excalidraw, or Structurizr DSL formats",
	Long:    `Compiles generated markdown documents and metadata from the specified project output directory into HTML, Excalidraw JSON, or Structurizr DSL diagram layouts.`,
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

		// 3. Export based on format
		var outputPath string
		switch exportFormatFlag {
		case "excalidraw":
			fmt.Fprintf(cmd.OutOrStdout(), "Exporting specifications to Excalidraw for project '%s'...\n", projectName)
			outputPath, err = export.ExportToExcalidraw(projectName, outputDir, distDir)
		case "dsl":
			fmt.Fprintf(cmd.OutOrStdout(), "Exporting specifications to Structurizr DSL for project '%s'...\n", projectName)
			outputPath, err = export.ExportToStructurizr(projectName, outputDir, distDir)
		default:
			fmt.Fprintf(cmd.OutOrStdout(), "Exporting specifications to HTML for project '%s'...\n", projectName)
			outputPath, err = export.ExportToHTML(projectName, outputDir, distDir)
		}

		if err != nil {
			logger.LogError(projectName, "export", "Export", err)
			return fmt.Errorf("export failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully exported specifications to: %s\n", outputPath)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportDestFlag, "dest", "d", "", "Custom destination directory (defaults to synthspec/<project_name>/dist)")
	exportCmd.Flags().StringVarP(&exportFormatFlag, "format", "f", "html", "Export format: html, excalidraw, dsl")
	rootCmd.AddCommand(exportCmd)
}
