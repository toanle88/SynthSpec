package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/generator"
	"github.com/toanle/synthspec/state"
)

var ingestProjectFlag string

var ingestCmd = &cobra.Command{
	Use:   "ingest <path>",
	Short: "Ingest local codebases, database schemas, and documentation (RAG)",
	Long:  `Scans the target directory, chunks text files, generates vector embeddings, and stores them in the project's local knowledge base.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		var projectName string
		var err error
		if ingestProjectFlag != "" {
			projectName = ingestProjectFlag
		} else {
			projectName, err = resolveProjectName(nil, "ingest")
			if err != nil {
				return fmt.Errorf("could not resolve project: %w. Provide --project <name>", err)
			}
		}

		// Load session to construct the corresponding gateway
		sess, err := state.LoadSession(projectName)
		if err != nil {
			return fmt.Errorf("failed to load project session '%s': %w", projectName, err)
		}

		gw, err := NewGatewayForSession(sess, mockFlag)
		if err != nil {
			return fmt.Errorf("failed to initialize gateway: %w", err)
		}

		projDir := state.GetSessionDir(projectName)
		kbPath := filepath.Join(projDir, "kb.json")

		fmt.Fprintf(cmd.OutOrStdout(), "Ingesting context from '%s' into project '%s'...\n", path, projectName)

		ing := generator.NewIngester(gw)
		count, err := ing.IngestDirectory(cmd.Context(), path, kbPath)
		if err != nil {
			return fmt.Errorf("ingestion failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully ingested %d context chunks. Vector KB stored at: %s\n", count, kbPath)
		return nil
	},
}

func init() {
	ingestCmd.Flags().StringVar(&ingestProjectFlag, "project", "", "Project name to associate with this knowledge base")
	rootCmd.AddCommand(ingestCmd)
}
