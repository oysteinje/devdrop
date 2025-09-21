// Package cmd provides the init command for DevDrop.
//
// The init command handles initial environment setup:
// - Pulls the base Ubuntu 24.04 image
// - Creates and starts an interactive container
// - Allows user to customize their development environment
// - Provides instructions for committing changes after customization
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

var starterImages = map[string]string{
	"ubuntu": "ubuntu:24.04",
	"go":     "golang:latest",
	"node":   "node:latest",
	"python": "python:latest",
}

var (
	envName         string
	starterImage    string
	customBaseImage string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new development environment",
	Long: `Initialize a new development environment by selecting a starter image
and creating a named environment where you can customize your setup.

This command will:
1. Let you choose from starter images (ubuntu, go, node, python) or provide a custom image
2. Create a named environment (automatically prefixed with 'devdrop-')
3. Start an interactive container with bash
4. Allow you to install tools, configure dotfiles, etc.
5. After you exit, run 'devdrop commit <env-name>' to save your changes

Examples:
  devdrop init                           # Interactive prompts for image and name
  devdrop init --name myenv              # Use 'devdrop-myenv' as environment name
  devdrop init --name myenv --image go   # Use Go starter image
  devdrop init --image custom --base-image myimage:latest  # Use custom image`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&envName, "name", "n", "", "Environment name (will be prefixed with 'devdrop-')")
	initCmd.Flags().StringVarP(&starterImage, "image", "i", "", "Starter image (ubuntu, go, node, python, or 'custom' for --base-image)")
	initCmd.Flags().StringVar(&customBaseImage, "base-image", "", "Custom base image URL (use with --image=custom)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get base image first (we need it for smart defaults)
	finalBaseImage := ""
	if starterImage == "" {
		finalBaseImage, err = promptForStarterImage()
		if err != nil {
			return err
		}
	} else {
		finalBaseImage, err = resolveBaseImage(starterImage, customBaseImage)
		if err != nil {
			return err
		}
	}

	// Get environment name with smart defaults
	finalEnvName := envName
	if finalEnvName == "" {
		// Generate smart default based on base image
		suggestedName := generateSmartDefault(finalBaseImage)

		// Prompt user with suggestion
		finalEnvName, err = promptForEnvironmentNameWithDefault(suggestedName)
		if err != nil {
			return err
		}

		// If user didn't provide a name, use the smart default
		if finalEnvName == "" {
			finalEnvName = suggestedName
		}
	}
	finalEnvName = config.EnsureDevDropPrefix(finalEnvName)

	fmt.Printf("Initializing environment '%s' with base image: %s\n", finalEnvName, finalBaseImage)

	// Pull base image
	fmt.Println("Pulling base image...")
	if err := dockerClient.PullImage(finalBaseImage); err != nil {
		return fmt.Errorf("failed to pull base image: %w", err)
	}

	// Create and start interactive container
	fmt.Println("Starting interactive container...")
	fmt.Println("You can now customize your development environment.")
	fmt.Printf("When finished, type 'exit' and then run 'devdrop commit %s' to save your changes.\n", finalEnvName)
	fmt.Println()

	containerID, err := dockerClient.CreateContainer(finalBaseImage)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := dockerClient.StartInteractiveContainer(containerID); err != nil {
		return fmt.Errorf("failed to start interactive container: %w", err)
	}

	// Create environment entry in config
	env := config.Environment{
		BaseImage:     finalBaseImage,
		Created:       time.Now(),
		LastUpdated:   time.Now(),
		LastContainer: containerID,
		Description:   fmt.Sprintf("Environment based on %s", finalBaseImage),
	}

	if err := cfg.AddEnvironment(finalEnvName, env); err != nil {
		return fmt.Errorf("failed to save environment to config: %w", err)
	}

	// Set this as the current environment
	if err := cfg.SetCurrentEnvironment(finalEnvName); err != nil {
		return fmt.Errorf("failed to set current environment: %w", err)
	}

	fmt.Println()
	fmt.Println("Container exited successfully!")
	fmt.Printf("Environment: %s\n", finalEnvName)
	fmt.Printf("Container ID: %s\n", containerID)
	fmt.Printf("Run 'devdrop commit %s' to save your customizations.\n", finalEnvName)

	return nil
}

func promptForEnvironmentName() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter environment name (will be prefixed with 'devdrop-'): ")
	name, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read environment name: %w", err)
	}
	name = strings.TrimSpace(name)
	// Don't provide a default here - let the caller handle it
	return name, nil
}

func promptForEnvironmentNameWithDefault(defaultName string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter environment name [%s]: ", defaultName)
	name, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read environment name: %w", err)
	}
	name = strings.TrimSpace(name)
	return name, nil
}

func promptForStarterImage() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Available starter images:")
	options := []string{"ubuntu", "go", "node", "python", "custom"}
	for i, option := range options {
		if option == "custom" {
			fmt.Printf("%d. %s (provide your own image URL)\n", i+1, option)
		} else {
			fmt.Printf("%d. %s (%s)\n", i+1, option, starterImages[option])
		}
	}

	fmt.Print("Select starter image (1-5): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read starter image selection: %w", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > 5 {
		return "", fmt.Errorf("invalid selection. Please choose 1-5")
	}

	selectedOption := options[choice-1]

	if selectedOption == "custom" {
		fmt.Print("Enter custom image URL: ")
		customImage, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read custom image URL: %w", err)
		}
		customImage = strings.TrimSpace(customImage)
		if customImage == "" {
			return "", fmt.Errorf("custom image URL cannot be empty")
		}
		return customImage, nil
	}

	return starterImages[selectedOption], nil
}

func resolveBaseImage(starter, custom string) (string, error) {
	if starter == "custom" {
		if custom == "" {
			return "", fmt.Errorf("--base-image is required when using --image=custom")
		}
		return custom, nil
	}

	if image, exists := starterImages[starter]; exists {
		return image, nil
	}

	return "", fmt.Errorf("unknown starter image: %s. Available options: ubuntu, go, node, python, custom", starter)
}

func generateSmartDefault(baseImage string) string {
	// Extract the base name from common image patterns
	if strings.HasPrefix(baseImage, "ubuntu") {
		return "ubuntu"
	}
	if strings.HasPrefix(baseImage, "golang") {
		return "go"
	}
	if strings.HasPrefix(baseImage, "node") {
		return "node"
	}
	if strings.HasPrefix(baseImage, "python") {
		return "python"
	}

	// For custom images, try to extract a meaningful name
	// e.g., "myregistry/myimage:tag" -> "myimage"
	parts := strings.Split(baseImage, "/")
	lastPart := parts[len(parts)-1]
	imageName := strings.Split(lastPart, ":")[0]

	// Remove common suffixes that aren't descriptive
	imageName = strings.TrimSuffix(imageName, "-latest")
	imageName = strings.TrimSuffix(imageName, "-dev")

	return imageName
}
