// Package cmd provides the ls command for DevDrop.
//
// The ls command lists available environments:
// - Local environments from config
// - Remote devdrop-* images from DockerHub registry
package cmd

import (
	"fmt"

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/oysteinje/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List available development environments",
	Long: `List all available development environments, showing both local
configurations and remote images available on DockerHub.

This command displays:
- Local environments (configured in ~/.devdrop/config.yaml)
- Remote devdrop-* images available for pull from DockerHub
- Current active environment (marked with *)

Examples:
  devdrop ls                    # List all environments
  devdrop ls --remote-only      # Show only remote images
  devdrop ls --local-only       # Show only local environments`,
	RunE: runLs,
}

var (
	remoteOnly bool
	localOnly  bool
)

func init() {
	rootCmd.AddCommand(lsCmd)
	lsCmd.Flags().BoolVar(&remoteOnly, "remote-only", false, "Show only remote images")
	lsCmd.Flags().BoolVar(&localOnly, "local-only", false, "Show only local environments")
}

func runLs(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Username == "" {
		return fmt.Errorf("not logged in. Please run 'devdrop login' first")
	}

	currentEnv := cfg.GetCurrentEnvironment()

	// Show local environments
	if !remoteOnly {
		fmt.Println("Local Environments:")
		if len(cfg.Environments) == 0 {
			fmt.Println("  (none configured)")
		} else {
			for name, env := range cfg.Environments {
				marker := " "
				if name == currentEnv {
					marker = "*"
				}
				fmt.Printf("  %s %s\n", marker, name)
				fmt.Printf("    Base: %s\n", env.BaseImage)
				fmt.Printf("    Created: %s\n", env.Created.Format("2006-01-02 15:04"))
				if !env.LastUpdated.IsZero() {
					fmt.Printf("    Updated: %s\n", env.LastUpdated.Format("2006-01-02 15:04"))
				}
				fmt.Println()
			}
		}
	}

	// Show remote environments
	if !localOnly {
		dockerClient, err := docker.NewClient()
		if err != nil {
			return fmt.Errorf("failed to connect to Docker: %w", err)
		}
		defer dockerClient.Close()

		fmt.Println("Remote Environments (DockerHub):")
		remoteImages, err := dockerClient.ListDevDropRepositories(cfg.Username)
		if err != nil {
			fmt.Printf("  Error fetching remote images: %v\n", err)
		} else if len(remoteImages) == 0 {
			fmt.Println("  (no devdrop- images found)")
		} else {
			for _, image := range remoteImages {
				localStatus := "not pulled"
				if _, exists := cfg.Environments[image]; exists {
					localStatus = "configured locally"
				}
				fmt.Printf("  %s (%s)\n", image, localStatus)
			}
		}
	}

	if currentEnv != "" && !remoteOnly {
		fmt.Printf("\nCurrent environment: %s\n", currentEnv)
	}

	return nil
}