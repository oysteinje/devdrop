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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

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

	err := cmd.Run()
	if err != nil {
		// Check if it's just a normal exit (exit status 0, 1, or 2 are normal for bash)
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			// Exit codes 0, 1, 2 are normal bash exits, don't treat as errors
			if exitCode >= 0 && exitCode <= 2 {
				return nil
			}
		}
		return fmt.Errorf("failed to start interactive container: %w", err)
	}

	return nil
}

func (c *Client) ImageExists(imageName string) bool {
	ctx := context.Background()
	_, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	return err == nil
}

func (c *Client) CreateWorkspaceContainer(imageName, workspaceDir string) (string, error) {
	ctx := context.Background()

	config := &container.Config{
		Image:        imageName,
		Cmd:          []string{"/bin/bash"},
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	}

	hostConfig := &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/workspace", workspaceDir)},
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create workspace container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) CommitContainer(containerID, imageName string) error {
	ctx := context.Background()

	options := types.ContainerCommitOptions{
		Reference: imageName,
		Comment:   "DevDrop environment commit",
		Author:    "DevDrop CLI",
	}

	_, err := c.cli.ContainerCommit(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to commit container %s to %s: %w", containerID, imageName, err)
	}

	return nil
}

func (c *Client) PushImage(imageName, authToken string) error {
	ctx := context.Background()

	// Use the stored auth token for authentication
	reader, err := c.cli.ImagePush(ctx, imageName, types.ImagePushOptions{
		RegistryAuth: authToken,
	})
	if err != nil {
		return fmt.Errorf("failed to push image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read the push output to completion (required for push to finish)
	buf, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read push output: %w", err)
	}

	// Parse the output to check for errors
	output := string(buf)
	if strings.Contains(output, `"error":"`) {
		return fmt.Errorf("push failed: %s", output)
	}

	// Show push progress to user (optional)
	if len(output) > 0 {
		fmt.Print(output)
	}

	return nil
}

func (c *Client) RemoveContainer(containerID string) error {
	ctx := context.Background()

	err := c.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true, // Remove even if container is running
	})
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// Docker Hub API structs
type DockerHubRepository struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	LastUpdated string `json:"last_updated"`
}

type DockerHubRepositoriesResponse struct {
	Count    int                   `json:"count"`
	Next     string                `json:"next"`
	Previous string                `json:"previous"`
	Results  []DockerHubRepository `json:"results"`
}

// ListDevDropRepositories lists all devdrop-* repositories for a user on Docker Hub
func (c *Client) ListDevDropRepositories(username string) ([]string, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/?page_size=100", username)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query Docker Hub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var hubResp DockerHubRepositoriesResponse
	if err := json.Unmarshal(body, &hubResp); err != nil {
		return nil, fmt.Errorf("failed to parse Docker Hub response: %w", err)
	}

	var devdropRepos []string
	for _, repo := range hubResp.Results {
		if strings.HasPrefix(repo.Name, "devdrop-") {
			devdropRepos = append(devdropRepos, repo.Name)
		}
	}

	return devdropRepos, nil
}
