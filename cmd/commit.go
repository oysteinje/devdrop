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

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/oysteinje/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit [environment-name]",
	Short: "Commit your customized environment to a personal image",
	Long: `Commit your customized development environment to a personal Docker image
and push it to DockerHub for later use.

This command will:
1. Use the current environment or the specified environment
2. Find the most recent container for that environment
3. Commit all your customizations to a new image
4. Push the image to DockerHub as username/devdrop-envname:latest
5. Update your configuration with the new environment

Prerequisites:
- You must have run 'devdrop login' to authenticate
- You must have a container from 'devdrop init' or 'devdrop run'

Examples:
  devdrop commit              # Commit current environment
  devdrop commit myenv        # Commit devdrop-myenv environment
  devdrop init
  # customize environment, install tools, etc.
  exit
  devdrop commit              # Save changes to current environment`,
	Args: cobra.MaximumNArgs(1),
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

	// Determine which environment to commit
	var targetEnv string
	if len(args) == 0 {
		// Use current environment
		if !cfg.HasEnvironments() {
			return fmt.Errorf("no environments configured. Run 'devdrop init' to create one")
		}
		targetEnv = cfg.GetCurrentEnvironment()
		if targetEnv == "" {
			return fmt.Errorf("no current environment set. Run 'devdrop switch' to select one")
		}
	} else {
		targetEnv = config.EnsureDevDropPrefix(args[0])
	}

	// Check if environment exists
	env, exists := cfg.Environments[targetEnv]
	if !exists {
		return fmt.Errorf("environment '%s' not found. Run 'devdrop ls' to see available environments", targetEnv)
	}

	// Check if there's a container to commit for this environment
	containerID := env.LastContainer
	if containerID == "" {
		return fmt.Errorf("no container to commit for environment '%s'. Run 'devdrop init' or 'devdrop run' first", targetEnv)
	}

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	// Generate environment image name
	imageName := cfg.GetEnvironmentImageName(targetEnv)

	fmt.Printf("Committing environment: %s\n", targetEnv)
	fmt.Printf("Container: %s\n", containerID[:12])
	fmt.Printf("Image: %s\n", imageName)

	// Commit container to image
	if err := dockerClient.CommitContainer(containerID, imageName); err != nil {
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
	env.Image = imageName
	env.LastUpdated = time.Now()
	env.LastContainer = "" // Clear since we're cleaning up the container

	if err := cfg.AddEnvironment(targetEnv, env); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	// Clean up the container
	fmt.Printf("Cleaning up container %s...\n", containerID[:12])
	if err := dockerClient.RemoveContainer(containerID); err != nil {
		// Don't fail the whole operation if cleanup fails
		fmt.Printf("Warning: failed to remove container: %v\n", err)
	} else {
		fmt.Println("Container cleaned up successfully!")
	}

	fmt.Println()
	fmt.Printf("✅ Environment '%s' successfully committed and pushed as %s\n", targetEnv, imageName)
	fmt.Printf("You can now run 'devdrop run %s' to use your customized environment in any project!\n", targetEnv)

	return nil
}
