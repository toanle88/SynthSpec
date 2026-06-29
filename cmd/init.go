package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/state"
	"github.com/toanle/synthspec/tui"
)

var initCmd = &cobra.Command{
	Use:   "init [project_name]",
	Short: "Initialize a new engineering specification project",
	Long:  `Sets up a new project directory, validates your API keys, and launches the interactive TUI interrogation loop.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		// 1. Resolve Configuration
		cfg, err := config.LoadConfig(providerFlag, modelFlag, mockFlag)
		if err != nil {
			return err
		}

		// 2. Setup Gateway
		gw, err := gateway.NewGateway(cfg.Provider, cfg.APIKey, cfg.Model)
		if err != nil {
			return err
		}

		// 3. Create Session File
		sessionPath := state.GetSessionPath(projectName)
		if _, err := os.Stat(sessionPath); err == nil {
			return fmt.Errorf("project '%s' already exists. Use 'synthspec resume %s' to continue", projectName, projectName)
		}

		var initialFacts gateway.Facts
		if blueprintFlag != "" {
			blueprints, err := config.LoadBlueprints()
			if err != nil {
				return fmt.Errorf("failed to load blueprints: %w", err)
			}
			var found bool
			for _, bp := range blueprints {
				if bp.ID == blueprintFlag {
					initialFacts = gateway.Facts{
						Functional: bp.Facts.Functional,
						Structural: bp.Facts.Structural,
						Security:   bp.Facts.Security,
						Compliance: bp.Facts.Compliance,
					}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("blueprint %q not found", blueprintFlag)
			}
		}

		sess := state.Session{
			ProjectName: projectName,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			Facts:       initialFacts,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := sess.Save(); err != nil {
			return fmt.Errorf("failed to save initial session: %w", err)
		}

		// 4. Run TUI Dashboard
		loadSettings, err := config.LoadSettings()
		if err != nil {
			logger.Log("WARN: failed to load settings: %v", err)
		}
		settings := loadSettings
		outDir := outputFlag
		if outDir == "" && settings != nil {
			outDir = settings.DefaultOutputFolder
		}
		// Default to project-specific output directory if not explicitly set via flag
		if outputFlag == "" {
			outDir = filepath.Join(state.GetSessionDir(projectName), "output")
		}

		fmt.Printf("Initializing project '%s' using %s (%s)...\n", projectName, cfg.Provider, cfg.Model)
		m := tui.NewDashboardModel(&sess, gw, outDir)
		if err := runTUI(m); err != nil {
			return fmt.Errorf("bubbletea execution failed: %w", err)
		}

		return nil
	},
}

var blueprintFlag string

func init() {
	initCmd.Flags().StringVarP(&blueprintFlag, "blueprint", "b", "", "Starting template/blueprint for project context (e.g. fintech-saas, internal-crud)")
	rootCmd.AddCommand(initCmd)
}
