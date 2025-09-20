// Package cmd provides the pull command for DevDrop.
//
// The pull command handles downloading the latest personal environment:
// - Checks authentication and personal image configuration
// - Pulls the latest version of the user's personal image from DockerHub
// - Provides feedback on success/failure and image details
// - Handles cases where the personal image doesn't exist on the registry
package cmd

import (
	"fmt"

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/oysteinje/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull the latest version of your personal environment",
	Long: `Pull the latest version of your personal development environment from DockerHub.

This command will:
1. Check your authentication and configuration
2. Pull the latest version of your personal image from DockerHub
3. Update your local image cache
4. Display information about the updated environment

Prerequisites:
- You must have run 'devdrop login' to authenticate
- You must have previously committed an environment with 'devdrop commit'

Example:
  devdrop pull    # Pull latest version of your environment`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if user is logged in
	if cfg.Username == "" {
		return fmt.Errorf("you must run 'devdrop login' first to authenticate with DockerHub")
	}

	// Get personal image name
	imageName := cfg.GetPersonalImageName()
	if imageName == "" {
		return fmt.Errorf("no username configured. Run 'devdrop login' first")
	}

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	fmt.Printf("Pulling latest version of your personal environment: %s\n", imageName)

	// Pull the image
	if err := dockerClient.PullImage(imageName); err != nil {
		// Check if this is a "not found" error
		if isImageNotFoundError(err) {
			return fmt.Errorf(`personal environment not found on DockerHub.

This usually means you haven't created an environment yet. To get started:
1. Run 'devdrop init' to create a new environment
2. Customize your environment (install tools, configure shell, etc.)
3. Run 'devdrop commit' to save and push your environment

Your personal image name: %s`, imageName)
		}
		return fmt.Errorf("failed to pull personal image: %w", err)
	}

	fmt.Println("✅ Personal environment pulled successfully!")
	fmt.Printf("Image: %s\n", imageName)
	fmt.Println()
	fmt.Println("Your environment is now up to date. Run 'devdrop run' to use it in any project.")

	return nil
}

// isImageNotFoundError checks if the error indicates the image was not found
func isImageNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "not found") ||
		contains(errStr, "404") ||
		contains(errStr, "does not exist") ||
		contains(errStr, "pull access denied")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 (len(s) > len(substr) &&
		  (s[:len(substr)] == substr ||
		   s[len(s)-len(substr):] == substr ||
		   indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring in a string
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}