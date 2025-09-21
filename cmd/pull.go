// Package cmd provides the pull command for DevDrop.
//
// The pull command handles downloading the latest personal environment:
// - Checks authentication and personal image configuration
// - Pulls the latest version of the user's personal image from DockerHub
// - Provides feedback on success/failure and image details
// - Handles cases where the personal image doesn't exist on the registry
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oysteinje/devdrop/pkg/config"
	"github.com/oysteinje/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [environment-name]",
	Short: "Pull the latest version of a development environment",
	Long: `Pull the latest version of a development environment from DockerHub.

This command will:
1. Check your authentication and configuration
2. Prompt you to select an environment (if not specified)
3. Pull the latest version of the selected environment from DockerHub
4. Update your local image cache
5. Display information about the updated environment

Prerequisites:
- You must have run 'devdrop login' to authenticate
- The environment must exist on DockerHub

Examples:
  devdrop pull              # Interactive prompt to select environment
  devdrop pull myenv        # Pull devdrop-myenv environment
  devdrop pull devdrop-go   # Pull devdrop-go environment`,
	Args: cobra.MaximumNArgs(1),
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

	var targetEnv string

	// Determine which environment to pull
	if len(args) == 0 {
		// Interactive selection
		if !cfg.HasEnvironments() {
			return fmt.Errorf("no environments configured. Run 'devdrop init' to create one")
		}
		targetEnv, err = promptForEnvironmentToPull(cfg)
		if err != nil {
			return err
		}
	} else {
		targetEnv = config.EnsureDevDropPrefix(args[0])
	}

	// Get image name
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

	fmt.Printf("Pulling environment '%s': %s\n", targetEnv, imageName)

	// Pull the image
	if err := dockerClient.PullImage(imageName); err != nil {
		// Check if this is a "not found" error
		if isImageNotFoundError(err) {
			return fmt.Errorf(`environment '%s' not found on DockerHub.

This usually means:
1. The environment hasn't been committed yet - run 'devdrop commit %s'
2. The environment name is incorrect - run 'devdrop ls' to see available environments
3. You don't have access to this image

Image name: %s`, targetEnv, targetEnv, imageName)
		}
		return fmt.Errorf("failed to pull environment image: %w", err)
	}

	// Update environment in config if it exists
	if env, exists := cfg.Environments[targetEnv]; exists {
		env.Image = imageName
		env.LastUpdated = time.Now()
		cfg.Environments[targetEnv] = env
		cfg.Save()
	}

	fmt.Println("✅ Environment pulled successfully!")
	fmt.Printf("Environment: %s\n", targetEnv)
	fmt.Printf("Image: %s\n", imageName)
	fmt.Println()
	fmt.Printf("Run 'devdrop run %s' to use this environment in any project.\n", targetEnv)

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

func promptForEnvironmentToPull(cfg *config.Config) (string, error) {
	reader := bufio.NewReader(os.Stdin)

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

	fmt.Print("Select environment to pull (1-" + fmt.Sprintf("%d", len(envNames)) + "): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read selection: %w", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(envNames) {
		return "", fmt.Errorf("invalid selection. Please choose 1-%d", len(envNames))
	}

	return envNames[choice-1], nil
}
