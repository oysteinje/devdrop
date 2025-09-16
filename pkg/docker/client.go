// Package docker provides a wrapper around the Docker SDK for DevDrop.
//
// This client serves as the foundation for all Docker operations:
// - Establishes connection to Docker daemon and tests connectivity
// - Provides single entry point for Docker operations
// - Handles connection errors gracefully
// - Will be extended for image ops (pull/push), container ops (create/start/stop/commit),
//   authentication (registry login), and volume mounting (workspace attachment)
package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test the connection
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

func (c *Client) RegistryLogin(ctx context.Context, authConfig types.AuthConfig) (registry.AuthenticateOKBody, error) {
	return c.cli.RegistryLogin(ctx, authConfig)
}

func (c *Client) PullImage(imageName string) error {
	ctx := context.Background()
	reader, err := c.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read the pull output to completion (required for pull to finish)
	// In a real implementation, you might want to display progress
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	return nil
}

func (c *Client) CreateContainer(imageName string) (string, error) {
	ctx := context.Background()

	config := &container.Config{
		Image:        imageName,
		Cmd:          []string{"/bin/bash"},
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := c.cli.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) StartInteractiveContainer(containerID string) error {
	// Use docker exec to run the container interactively
	// This is simpler and more reliable than trying to handle TTY attachment through the Go API
	cmd := exec.Command("docker", "start", "-i", containerID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start interactive container: %w", err)
	}

	return nil
}