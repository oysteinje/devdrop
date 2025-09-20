// Package cmd provides the commit command for DevDrop.
//
// The commit command handles saving container customizations:
// - Finds the most recent container from devdrop init
// - Commits container changes to a personal Docker image
// - Pushes the image to DockerHub using stored credentials
// - Updates configuration with environment metadata
// - Optionally cleans up the committed container
package cmd

import (
	"fmt"
	"time"

	"github.com/qbits/devdrop/pkg/config"
	"github.com/qbits/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit your customized environment to a personal image",
	Long: `Commit your customized development environment to a personal Docker image
and push it to DockerHub for later use.

This command will:
1. Find the container from your last 'devdrop init'
2. Commit all your customizations to a new image
3. Push the image to DockerHub as username/devdrop-env:latest
4. Update your configuration with the new environment

Prerequisites:
- You must have run 'devdrop login' to authenticate
- You must have run 'devdrop init' and customized the environment

Example:
  devdrop init
  # customize environment, install tools, etc.
  exit
  devdrop commit`,
	RunE: runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if user is logged in
	if cfg.Username == "" {
		return fmt.Errorf("you must run 'devdrop login' first to authenticate with DockerHub")
	}

	// Check if we have auth token
	if cfg.AuthToken == "" {
		return fmt.Errorf("missing authentication token. Please run 'devdrop login' again")
	}

	// Check if there's a container to commit
	if cfg.LastContainer == "" {
		return fmt.Errorf("no container to commit. Run 'devdrop init' first to create an environment")
	}

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	// Generate personal image name
	imageName := cfg.GetPersonalImageName()

	fmt.Printf("Committing container %s to image %s...\n", cfg.LastContainer[:12], imageName)

	// Commit container to image
	if err := dockerClient.CommitContainer(cfg.LastContainer, imageName); err != nil {
		return fmt.Errorf("failed to commit container: %w", err)
	}

	fmt.Println("Container committed successfully!")

	// Push image to DockerHub
	fmt.Printf("Pushing image %s to DockerHub...\n", imageName)
	if err := dockerClient.PushImage(imageName, cfg.AuthToken); err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	fmt.Println("Image pushed successfully!")

	// Update environment in configuration
	env := config.Environment{
		Image:       imageName,
		Created:     time.Now(),
		LastUpdated: time.Now(),
		Description: "DevDrop environment created from ubuntu:24.04",
	}

	if err := cfg.AddEnvironment("default", env); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	// Clean up the container
	fmt.Printf("Cleaning up container %s...\n", cfg.LastContainer[:12])
	if err := dockerClient.RemoveContainer(cfg.LastContainer); err != nil {
		// Don't fail the whole operation if cleanup fails
		fmt.Printf("Warning: failed to remove container: %v\n", err)
	} else {
		fmt.Println("Container cleaned up successfully!")
	}

	// Clear the last container ID since it's been committed and removed
	cfg.LastContainer = ""
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: failed to clear container ID from config: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("✅ Environment successfully committed and pushed as %s\n", imageName)
	fmt.Println("You can now run 'devdrop run' to use your customized environment in any project!")

	return nil
}
