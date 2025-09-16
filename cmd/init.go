// Package cmd provides the init command for DevDrop.
//
// The init command handles initial environment setup:
// - Pulls the base Ubuntu 24.04 image
// - Creates and starts an interactive container
// - Allows user to customize their development environment
// - Provides instructions for committing changes after customization
package cmd

import (
	"fmt"

	"github.com/qbits/devdrop/pkg/docker"
	"github.com/spf13/cobra"
)

const (
	baseImage = "ubuntu:24.04"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new development environment",
	Long: `Initialize a new development environment by pulling the base Ubuntu 24.04 image
and starting an interactive container where you can customize your environment.

This command will:
1. Pull the Ubuntu 24.04 base image
2. Start an interactive container with bash
3. Allow you to install tools, configure dotfiles, etc.
4. After you exit, run 'devdrop commit' to save your changes

Example:
  devdrop init
  # Inside container: install vim, git, configure shell, etc.
  exit
  devdrop commit`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	fmt.Printf("Initializing DevDrop environment with base image: %s\n", baseImage)

	// Pull base image
	fmt.Println("Pulling base image...")
	if err := dockerClient.PullImage(baseImage); err != nil {
		return fmt.Errorf("failed to pull base image: %w", err)
	}

	// Create and start interactive container
	fmt.Println("Starting interactive container...")
	fmt.Println("You can now customize your development environment.")
	fmt.Println("When finished, type 'exit' and then run 'devdrop commit' to save your changes.")
	fmt.Println()

	containerID, err := dockerClient.CreateContainer(baseImage)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := dockerClient.StartInteractiveContainer(containerID); err != nil {
		return fmt.Errorf("failed to start interactive container: %w", err)
	}

	fmt.Println()
	fmt.Println("Container exited successfully!")
	fmt.Printf("Container ID: %s\n", containerID)
	fmt.Println("Run 'devdrop commit' to save your customizations to your personal image.")

	return nil
}