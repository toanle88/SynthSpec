package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/logger"
	"github.com/toanle/synthspec/tui"
)

var (
	providerFlag string
	modelFlag    string
	mockFlag     bool
	outputFlag   string
	debugFlag    bool

	runTUI = func(m tui.DashboardModel) error {
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		_, err := p.Run()
		return err
	}
)

var rootCmd = &cobra.Command{
	Use:   "synthspec",
	Short: "SynthSpec: Open-Source BYOK AI Solution Architect CLI",
	Long:  `SynthSpec is a privacy-first, open-source command-line utility that transforms vague application ideas into production-ready, enterprise-grade engineering specifications.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		s, _ := config.LoadSettings()
		settingsDebug := false
		if s != nil {
			settingsDebug = s.Debug
		}
		if err := logger.Init(debugFlag, settingsDebug); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logger.Close()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.NewWelcomeModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		resModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("welcome menu execution failed: %w", err)
		}

		welcomeModel := resModel.(tui.WelcomeModel)
		switch welcomeModel.Action {
		case tui.ActionCreate:
			if welcomeModel.SelectedBlueprint != "" {
				_ = initCmd.Flags().Set("blueprint", welcomeModel.SelectedBlueprint)
			}
			return initCmd.RunE(initCmd, []string{welcomeModel.ProjectName})
		case tui.ActionResume:
			return resumeCmd.RunE(resumeCmd, []string{welcomeModel.ProjectName})
		case tui.ActionExport:
			return exportCmd.RunE(exportCmd, []string{welcomeModel.ProjectName})
		case tui.ActionExit:
			return nil
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&providerFlag, "provider", "p", "", "Explicitly override LLM provider (gemini, openai, anthropic, openrouter)")
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "", "Explicitly override LLM model")
	rootCmd.PersistentFlags().BoolVar(&mockFlag, "mock", false, "Use mock LLM provider for local testing and development")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Override output directory for generated assets")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Write sanitized execution traces to .synthspec/crash.log")
}
