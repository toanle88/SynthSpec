package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
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
		var gw gateway.Gateway
		switch cfg.Provider {
		case config.ProviderMock:
			gw = gateway.NewMockGateway()
		case config.ProviderGemini:
			gw = gateway.NewGeminiGateway(cfg.APIKey, cfg.Model)
		case config.ProviderOpenAI:
			gw = gateway.NewOpenAIGateway(cfg.APIKey, cfg.Model)
		case config.ProviderAnthropic:
			gw = gateway.NewAnthropicGateway(cfg.APIKey, cfg.Model)
		case config.ProviderOpenRouter:
			gw = gateway.NewOpenRouterGateway(cfg.APIKey, cfg.Model)
		default:
			return fmt.Errorf("unrecognized provider: %s", cfg.Provider)
		}

		// 3. Create Session File
		sessionPath := state.GetSessionPath(projectName)
		if _, err := os.Stat(sessionPath); err == nil {
			return fmt.Errorf("project '%s' already exists. Use 'synthspec resume %s' to continue", projectName, projectName)
		}

		sess := state.Session{
			ProjectName: projectName,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := sess.Save(); err != nil {
			return fmt.Errorf("failed to save initial session: %w", err)
		}

		// 4. Run TUI Dashboard
		fmt.Printf("Initializing project '%s' using %s (%s)...\n", projectName, cfg.Provider, cfg.Model)
		m := tui.NewDashboardModel(&sess, gw, outputFlag)
		if err := runTUI(m); err != nil {
			return fmt.Errorf("bubbletea execution failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// Helper to initialize gateway based on session state
func getGatewayForSession(sess *state.Session, forceMock bool) (gateway.Gateway, error) {
	if forceMock || sess.Provider == config.ProviderMock {
		return gateway.NewMockGateway(), nil
	}

	cfg, err := config.LoadConfig(sess.Provider, sess.Model, false)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve credentials: %w", err)
	}

	switch cfg.Provider {
	case config.ProviderGemini:
		return gateway.NewGeminiGateway(cfg.APIKey, cfg.Model), nil
	case config.ProviderOpenAI:
		return gateway.NewOpenAIGateway(cfg.APIKey, cfg.Model), nil
	case config.ProviderAnthropic:
		return gateway.NewAnthropicGateway(cfg.APIKey, cfg.Model), nil
	case config.ProviderOpenRouter:
		return gateway.NewOpenRouterGateway(cfg.APIKey, cfg.Model), nil
	default:
		return nil, errors.New("unsupported provider in session")
	}
}
