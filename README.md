# DevDrop

> Portable development environments using Docker

DevDrop lets you create, customize, and share complete development environments. Think "dotfiles for entire environments" – install your tools once, use them anywhere.

## What it solves

Setting up dev environments is tedious. DevDrop packages your tools, configs, and shell setup into a Docker image you can use on any machine. Install tools and dependencies in your environment without affecting your host system.

## Who it's for

- Developers working across multiple machines
- Anyone tired of environment setup

## How it works

1. `devdrop init` - Start with Ubuntu
2. Install tools, configure shell
3. `devdrop commit` - Save to DockerHub
4. `devdrop run` - Use anywhere

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/oysteinje/devdrop/main/install.sh | bash
```

**Prerequisites**: Docker + DockerHub account

## Quick start

```bash
devdrop login     # Authenticate
devdrop init      # Create environment
# Install tools, configure shell
exit
devdrop commit    # Save changes
devdrop run       # Use in any project
```

## Commands

- `devdrop login` - Authenticate with DockerHub
- `devdrop init` - Create new environment
- `devdrop run` - Use environment in current directory
- `devdrop commit` - Save changes
- `devdrop pull` - Pull latest version
