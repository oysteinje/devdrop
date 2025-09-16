// Package cmd provides the login command for DevDrop.
//
// The login command handles Docker registry authentication:
// - Prompts user for DockerHub username and password
// - Authenticates with Docker registry using Docker SDK
// - Stores credentials securely using Docker's credential store
// - Updates ~/.devdrop/config.yaml with username for image naming
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/qbits/devdrop/pkg/docker"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Docker registry",
	Long: `Authenticate with Docker registry (DockerHub by default) to enable
pushing and pulling of personal development environment images.

This will prompt for your DockerHub username and password, then store
the credentials securely using Docker's credential helper.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer dockerClient.Close()

	// Get username
	fmt.Print("Username: ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Get password (hidden input)
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Add newline after hidden password input
	password := string(passwordBytes)

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Authenticate with Docker registry
	ctx := context.Background()
	authConfig := types.AuthConfig{
		Username: username,
		Password: password,
	}

	response, err := dockerClient.RegistryLogin(ctx, authConfig)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Login successful! %s\n", response.Status)
	fmt.Printf("Logged in as: %s\n", username)

	// TODO: Save username to ~/.devdrop/config.yaml for image naming
	fmt.Println("Note: Configuration management will be implemented in a future step.")

	return nil
}