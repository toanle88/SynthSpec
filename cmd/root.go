package cmd

import (
	"github.com/spf13/cobra"
)

var (
	providerFlag string
	modelFlag    string
	mockFlag     bool
	outputFlag   string
)

var rootCmd = &cobra.Command{
	Use:   "synthspec",
	Short: "SynthSpec: Open-Source BYOK AI Solution Architect CLI",
	Long:  `SynthSpec is a privacy-first, open-source command-line utility that transforms vague application ideas into production-ready, enterprise-grade engineering specifications.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
}
