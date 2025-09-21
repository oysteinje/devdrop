// Package cmd provides the status command for DevDrop.
//
// The status command shows current environment status and recent containers.
package cmd

import (
	"fmt"

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/oysteinje/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current environment status",
	Long: `Display the current DevDrop environment status including:
- Current active environment
- Recent containers for the current environment
- Environment configuration details
- Local vs remote sync status

Example:
  devdrop status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Username == "" {
		fmt.Println("Status: Not logged in")
		fmt.Println("Run 'devdrop login' to authenticate with DockerHub")
		return nil
	}

	fmt.Printf("User: %s\n", cfg.Username)

	if !cfg.HasEnvironments() {
		fmt.Println("Status: No environments configured")
		fmt.Println("Run 'devdrop init' to create your first environment")
		return nil
	}

	currentEnv := cfg.GetCurrentEnvironment()
	if currentEnv == "" {
		fmt.Println("Status: No active environment")
		fmt.Println("Run 'devdrop switch' to select an environment")
		return nil
	}

	env := cfg.Environments[currentEnv]
	fmt.Printf("Current Environment: %s\n", currentEnv)
	fmt.Printf("Base Image: %s\n", env.BaseImage)
	fmt.Printf("Created: %s\n", env.Created.Format("2006-01-02 15:04:05"))
	if !env.LastUpdated.IsZero() {
		fmt.Printf("Last Updated: %s\n", env.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	if env.Description != "" {
		fmt.Printf("Description: %s\n", env.Description)
	}

	// Show container status
	if env.LastContainer != "" {
		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Printf("Last Container: %s (Docker connection failed)\n", env.LastContainer)
		} else {
			defer dockerClient.Close()
			fmt.Printf("Last Container: %s\n", env.LastContainer)
			// TODO: Add container status check (running, stopped, etc.)
		}
	}

	// Show image status
	expectedImage := cfg.GetEnvironmentImageName(currentEnv)
	fmt.Printf("Expected Image: %s\n", expectedImage)

	// Show total environments
	fmt.Printf("\nTotal Environments: %d\n", len(cfg.Environments))

	if len(cfg.Environments) > 1 {
		fmt.Println("Other Environments:")
		for name := range cfg.Environments {
			if name != currentEnv {
				fmt.Printf("  %s\n", name)
			}
		}
	}

	return nil
}