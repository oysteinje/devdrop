// Package cmd provides the switch command for DevDrop.
//
// The switch command changes the current active environment context.
package cmd

import (
	"fmt"

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <environment-name>",
	Short: "Switch to a different development environment",
	Long: `Switch the current active environment context. This affects which
environment is used by default for run, commit, and other commands.

The environment name will be automatically prefixed with 'devdrop-' if needed.

Examples:
  devdrop switch myenv          # Switch to devdrop-myenv
  devdrop switch devdrop-go     # Switch to devdrop-go
  devdrop switch                # Interactive prompt to choose environment`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.HasEnvironments() {
		return fmt.Errorf("no environments configured. Run 'devdrop init' to create one")
	}

	var targetEnv string

	if len(args) == 0 {
		// Interactive selection
		targetEnv, err = promptForEnvironmentSelection(cfg)
		if err != nil {
			return err
		}
	} else {
		targetEnv = config.EnsureDevDropPrefix(args[0])
	}

	// Verify environment exists
	if _, exists := cfg.Environments[targetEnv]; !exists {
		return fmt.Errorf("environment '%s' not found. Run 'devdrop ls' to see available environments", targetEnv)
	}

	// Switch to the environment
	if err := cfg.SetCurrentEnvironment(targetEnv); err != nil {
		return fmt.Errorf("failed to switch environment: %w", err)
	}

	fmt.Printf("Switched to environment: %s\n", targetEnv)
	return nil
}

func promptForEnvironmentSelection(cfg *config.Config) (string, error) {
	fmt.Println("Available environments:")

	envNames := make([]string, 0, len(cfg.Environments))
	for name := range cfg.Environments {
		envNames = append(envNames, name)
	}

	for i, name := range envNames {
		marker := " "
		if name == cfg.GetCurrentEnvironment() {
			marker = "*"
		}
		fmt.Printf("%d.%s %s\n", i+1, marker, name)
	}

	fmt.Print("Select environment (1-" + fmt.Sprintf("%d", len(envNames)) + "): ")

	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if choice < 1 || choice > len(envNames) {
		return "", fmt.Errorf("invalid selection. Please choose 1-%d", len(envNames))
	}

	return envNames[choice-1], nil
}