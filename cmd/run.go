// Package cmd provides the run command for DevDrop.
//
// The run command starts your personal development environment:
// - Checks if personal image exists locally, pulls from DockerHub if not
// - Creates and starts container with current directory mounted as /workspace
// - Provides interactive shell in your customized environment
// - Automatically cleans up container when session ends
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/oysteinje/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run your personal development environment",
	Long: `Run your personal development environment in the current directory.

This command will:
1. Check if your personal image exists locally
2. Pull from DockerHub if needed (requires 'devdrop login')
3. Start an interactive container with current directory mounted as /workspace
4. Drop you into your customized shell environment
5. Save the container for potential commit after session ends

The current directory will be available as /workspace inside the container,
and any changes you make to files will persist on your host system.
Container changes can be committed with 'devdrop commit' after the session.

Prerequisites:
- You must have run 'devdrop login' and 'devdrop commit' first
- Your personal image must exist on DockerHub

Example:
  cd ~/my-project
  devdrop run
  # Inside container: your tools are available, /workspace contains project files
  # Install additional tools, make changes
  exit
  devdrop commit  # Save changes to your personal image`,
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
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

	// Check if image exists locally, pull if not
	fmt.Printf("Checking for personal environment: %s\n", imageName)

	if !dockerClient.ImageExists(imageName) {
		fmt.Printf("Personal image not found locally. Pulling from DockerHub...\n")
		if err := dockerClient.PullImage(imageName); err != nil {
			return fmt.Errorf("failed to pull personal image. Make sure you've run 'devdrop commit' first: %w", err)
		}
		fmt.Println("Image pulled successfully!")
	} else {
		fmt.Println("Personal environment found locally.")
	}

	// Get current directory to mount as workspace
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(currentDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fmt.Printf("Starting environment in: %s\n", absPath)
	fmt.Printf("Current directory will be available as /workspace inside the container.\n")
	fmt.Println()

	// Create and start container with volume mount
	containerID, err := dockerClient.CreateWorkspaceContainer(imageName, absPath)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start interactive container
	fmt.Println("Starting your development environment...")
	if err := dockerClient.StartInteractiveContainer(containerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Save container ID to config for potential commit
	fmt.Println()
	fmt.Println("Development session ended.")
	fmt.Printf("Container ID: %s\n", containerID)

	if err := cfg.SetLastContainer(containerID); err != nil {
		fmt.Printf("Warning: failed to save container ID to config: %v\n", err)
	} else {
		fmt.Println("Container saved for potential commit. Run 'devdrop commit' to save your changes.")
	}

	fmt.Println("Note: Container will remain available for commit. Run 'devdrop commit' to save changes and clean up.")

	return nil
}
