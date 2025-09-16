# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevDrop is a CLI tool written in Go that allows developers to create, customize, and share personal development environments using Docker containers. Think "dotfiles for entire environments" - portable, version-controlled, and instantly available anywhere Docker runs.

## Development Commands

This is a Go project using standard Go toolchain:

```bash
# Initialize Go module (if not already done)
go mod init github.com/yourusername/devdrop

# Install dependencies
go mod tidy

# Build the CLI
go build -o devdrop ./cmd/devdrop

# Run the CLI
./devdrop [command]
```

## Building and Running Locally

To build and test DevDrop locally:

```bash
# Clone and enter the repository
git clone <repository-url>
cd devdrop

# Install dependencies
go mod tidy

# Build the binary
go build -o devdrop ./cmd/devdrop

# Test the CLI
./devdrop --help
./devdrop login --help

# Run commands (requires Docker to be running)
./devdrop login    # Authenticate with DockerHub
```

**Prerequisites for local development:**
- Go 1.18+ installed
- Docker installed and running
- DockerHub account (for testing login functionality)

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o devdrop-linux ./cmd/devdrop
GOOS=darwin GOARCH=amd64 go build -o devdrop-macos ./cmd/devdrop
GOOS=windows GOARCH=amd64 go build -o devdrop-windows.exe ./cmd/devdrop
```

## Architecture Overview

DevDrop follows a standard Go CLI architecture using Cobra:

### Core Components
- **CLI Commands** (`cmd/`): Cobra-based command implementations
  - `init.go` - Pull base image, start interactive container
  - `run.go` - Start container with current directory mounted
  - `commit.go` - Commit container state to personal image
  - `pull.go` - Pull latest personal image from registry
- **Docker Operations** (`pkg/docker/`): Docker SDK wrapper and container/image management
- **Configuration** (`pkg/config/`): User config management (~/.devdrop/config.yaml)
- **Authentication** (`pkg/auth/`): DockerHub/registry authentication

### Key Technical Decisions
- **Language**: Go for single binary distribution and excellent Docker SDK support
- **CLI Framework**: Cobra for command structure and argument parsing
- **Container Registry**: Initially DockerHub, designed for pluggable backends
- **Base Image**: Ubuntu 22.04 LTS with essential dev tools

### MVP Workflow Implementation
```bash
devdrop init     # Pull devdrop/base:latest, run interactive container
devdrop commit   # Commit container to username/devdrop-env:latest, push to registry
devdrop run      # Run personal image with current directory mounted as /workspace
devdrop pull     # Pull latest personal image from registry
```

## Docker Integration

The CLI wraps these core Docker operations:
```go
// Key Docker SDK operations
docker.ImagePull()           // Pull base/personal images
docker.ContainerCreate()     // Create containers with proper mounts
docker.ContainerStart()      // Start interactive containers
docker.ContainerCommit()     // Save container state to image
docker.ImagePush()           // Push personal images to registry
```

Container configuration:
- Interactive mode with TTY allocation
- Current directory mounted as `/workspace`
- Working directory set to `/workspace`
- Proper signal handling for graceful shutdown

## Expected Project Structure

```
devdrop/
├── cmd/devdrop/            # CLI entry point
│   └── main.go
├── cmd/                    # Command implementations
│   ├── root.go             # Cobra root command setup
│   ├── login.go            # devdrop login - authenticate with registry
│   ├── init.go             # devdrop init - pull base image, start interactive container
│   ├── run.go              # devdrop run - start container with current directory mounted
│   ├── commit.go           # devdrop commit - commit container state, push to registry
│   └── pull.go             # devdrop pull - pull latest personal image
├── pkg/
│   ├── docker/             # Docker SDK wrapper
│   │   ├── client.go       # Docker client connection and configuration
│   │   ├── images.go       # Image pull/push/management operations
│   │   └── containers.go   # Container create/start/stop/commit operations
│   ├── config/             # Configuration management
│   │   └── config.go       # User config (~/.devdrop/config.yaml)
│   └── auth/               # Registry authentication
│       └── auth.go         # DockerHub/registry auth handling
├── internal/               # Private application code
│   └── version/
│       └── version.go      # Application version info
├── go.mod
├── go.sum
└── Makefile               # Build automation
```

## Dependencies

Key Go modules this project uses:
- `github.com/spf13/cobra` - CLI framework and command structure
- `github.com/docker/docker` - Docker Engine SDK for container operations
- `github.com/docker/docker/api/types` - Docker API type definitions
- Standard library: `context`, `os`, `path/filepath`, `fmt`

## Configuration Format

User configuration stored at `~/.devdrop/config.yaml`:
```yaml
username: dockerhub-username
base_image: devdrop/base:latest
default_shell: /bin/bash
environments:
  default:
    image: username/devdrop-env:latest
    created: "2025-01-15T10:30:00Z"
    last_updated: "2025-01-20T14:45:00Z"
```

## Command Details

### `devdrop login`
```bash
# What it does:
# 1. Prompt for DockerHub username and password
# 2. Authenticate with Docker registry using Docker SDK
# 3. Store credentials securely using Docker's credential store
# 4. Update ~/.devdrop/config.yaml with username
```

### `devdrop init`
```bash
# What it does:
# 1. Pull base image (devdrop/base:latest)
# 2. Run interactive container: docker run -it devdrop/base:latest /bin/bash
# 3. User customizes environment
# 4. User exits
# 5. Prompt user to run 'devdrop commit' to save changes
```

### `devdrop run`
```bash
# What it does:
# 1. Check if personal image exists locally
# 2. If not, pull it: docker pull username/devdrop-env:latest
# 3. Run with current directory mounted:
#    docker run -it -v $(pwd):/workspace -w /workspace username/devdrop-env:latest /bin/bash
```

### `devdrop commit`
```bash
# What it does:
# 1. Find the most recently exited devdrop container
# 2. Commit it: docker commit <container-id> username/devdrop-env:latest
# 3. Push to DockerHub: docker push username/devdrop-env:latest
# 4. Clean up old containers
```

### `devdrop pull`
```bash
# What it does:
# 1. Pull latest personal image: docker pull username/devdrop-env:latest
# 2. Update local image cache
# 3. Display success message with image details
```

## Development Prerequisites

- Docker installed and running locally
- DockerHub account (for image registry)
- Go 1.21+ for development
- `golangci-lint` for code linting (optional but recommended)

## Implementation Priority

When implementing features, follow this order:
1. Basic Cobra CLI structure with root command
2. Docker client connection and error handling
3. `devdrop login` command (authentication setup)
4. `devdrop init` command (simplest - pull and run base image)
5. `devdrop commit` command (container commit and registry push)
6. `devdrop run` command (run personal image with directory mounting)
7. Configuration file management
8. `devdrop pull` command
9. Enhanced error handling and user experience
