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
	Use:   "run [environment-name]",
	Short: "Run a development environment",
	Long: `Run a development environment in the current directory.

This command will:
1. Use the current environment or the specified environment
2. Check if the environment image exists locally
3. Pull from DockerHub if needed (requires 'devdrop login')
4. Start an interactive container with current directory mounted as /workspace
5. Drop you into your customized shell environment
6. Save the container for potential commit after session ends

The current directory will be available as /workspace inside the container,
and any changes you make to files will persist on your host system.
Container changes can be committed with 'devdrop commit' after the session.

Prerequisites:
- You must have run 'devdrop login' first
- The environment must exist locally or on DockerHub

Examples:
  cd ~/my-project
  devdrop run                    # Use current environment
  devdrop run myenv              # Use devdrop-myenv environment
  # Inside container: your tools are available, /workspace contains project files
  # Install additional tools, make changes
  exit
  devdrop commit                 # Save changes to current environment`,
	Args: cobra.MaximumNArgs(1),
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

	// Determine which environment to run
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

	// Get environment image name
	imageName := cfg.GetEnvironmentImageName(targetEnv)
	if imageName == "" {
		return fmt.Errorf("no username configured. Run 'devdrop login' first")
	}

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	// Check if committed image exists locally
	fmt.Printf("Using environment: %s\n", targetEnv)
	fmt.Printf("Checking for environment image: %s\n", imageName)

	var useImage string

	if dockerClient.ImageExists(imageName) {
		fmt.Println("Environment image found locally.")
		useImage = imageName
	} else {
		// Check if environment exists in config (might have uncommitted changes)
		if env, exists := cfg.Environments[targetEnv]; exists && env.BaseImage != "" {
			fmt.Printf("Environment image not found, using base image: %s\n", env.BaseImage)
			fmt.Println("Note: You'll be running the base environment. Run 'devdrop commit' after your session to save changes.")
			useImage = env.BaseImage
		} else {
			// Try pulling from DockerHub as last resort
			fmt.Printf("Environment image not found locally. Pulling from DockerHub...\n")
			if err := dockerClient.PullImage(imageName); err != nil {
				return fmt.Errorf("failed to pull environment image. Make sure the environment exists or run 'devdrop init' first: %w", err)
			}
			fmt.Println("Image pulled successfully!")
			useImage = imageName
		}
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
	containerID, err := dockerClient.CreateWorkspaceContainer(useImage, absPath)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start interactive container
	fmt.Println("Starting your development environment...")
	if err := dockerClient.StartInteractiveContainer(containerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Save container ID to environment config for potential commit
	fmt.Println()
	fmt.Println("Development session ended.")
	fmt.Printf("Environment: %s\n", targetEnv)
	fmt.Printf("Container ID: %s\n", containerID)

	if err := cfg.SetEnvironmentContainer(targetEnv, containerID); err != nil {
		fmt.Printf("Warning: failed to save container ID to config: %v\n", err)
	} else {
		fmt.Printf("Container saved for potential commit. Run 'devdrop commit %s' to save your changes.\n", targetEnv)
	}

	fmt.Printf("Note: Container will remain available for commit. Run 'devdrop commit %s' to save changes and clean up.\n", targetEnv)

	return nil
}
