package cmd

import (
	"fmt"

	"github.com/toanle/synthspec/config"
	"github.com/toanle/synthspec/gateway"
	"github.com/toanle/synthspec/state"
)

// NewGatewayForSession creates a Gateway from a saved session state, using the session's stored provider/model.
func NewGatewayForSession(sess *state.Session, forceMock bool) (gateway.Gateway, error) {
	if forceMock || sess.Provider == config.ProviderMock {
		return gateway.NewMockGateway(), nil
	}

	cfg, err := config.LoadConfig(sess.Provider, sess.Model, false)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve credentials: %w", err)
	}

	return gateway.NewGateway(cfg.Provider, cfg.APIKey, cfg.Model)
}

// resolveProjectName auto-detects a single project from args, or returns an error
// if zero or multiple projects match. Works for resume/update/export commands.
func resolveProjectName(args []string, action string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	// Auto-detect projects
	projects, err := state.ListProjects()
	if err != nil {
		return "", fmt.Errorf("failed to scan for projects: %w", err)
	}

	if len(projects) == 0 {
		return "", fmt.Errorf("no active projects found to %s. Start one using 'synthspec init <project_name>'", action)
	}

	if len(projects) > 1 {
		fmt.Printf("Multiple active projects found. Please select one to %s:\n", action)
		for _, p := range projects {
			fmt.Printf(" - %s\n", p)
		}
		return "", fmt.Errorf("use 'synthspec %s [project_name]' to specify which project to load", action)
	}

	return projects[0], nil
}
