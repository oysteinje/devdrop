#!/bin/bash

# DevDrop installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/oysteinje/devdrop/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository
REPO="oysteinje/devdrop"
GITHUB_API="https://api.github.com/repos/${REPO}"

# Installation directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        CYGWIN*|MINGW*|MSYS*) os="windows";;
        *)
            echo -e "${RED}Error: Unsupported operating system: $(uname -s)${NC}" >&2
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64)     arch="amd64";;
        arm64|aarch64) arch="arm64";;
        *)
            echo -e "${RED}Error: Unsupported architecture: $(uname -m)${NC}" >&2
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Get the latest release URL
get_latest_release() {
    local platform="$1"
    local binary_name="devdrop-${platform}"

    if [[ "$platform" == *"windows"* ]]; then
        binary_name="${binary_name}.exe"
    fi

    # Get latest release info from GitHub API
    local release_url
    release_url=$(curl -s "${GITHUB_API}/releases/latest" | \
        grep "browser_download_url.*${binary_name}\"" | \
        cut -d '"' -f 4)

    if [ -z "$release_url" ]; then
        echo -e "${RED}Error: Could not find release for platform: ${platform}${NC}" >&2
        exit 1
    fi

    echo "$release_url"
}

# Check if running as root for system-wide install
check_permissions() {
    if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$EUID" -ne 0 ]; then
        echo -e "${YELLOW}Note: Installing to system directory requires sudo${NC}"
        echo -e "You can also install to your home directory with:"
        echo -e "${BLUE}  export INSTALL_DIR=\$HOME/.local/bin${NC}"
        echo -e "${BLUE}  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash${NC}"
        echo
        echo -e "Continuing with sudo for system-wide installation..."
        return 0
    fi
}

# Create installation directory
create_install_dir() {
    if [ ! -d "$INSTALL_DIR" ]; then
        echo -e "${BLUE}Creating installation directory: ${INSTALL_DIR}${NC}"
        if [ "$INSTALL_DIR" = "/usr/local/bin" ]; then
            sudo mkdir -p "$INSTALL_DIR"
        else
            mkdir -p "$INSTALL_DIR"
        fi
    fi
}

# Download and install binary
install_binary() {
    local platform="$1"
    local download_url="$2"
    local temp_file="/tmp/devdrop-${platform}"

    echo -e "${BLUE}Downloading DevDrop for ${platform}...${NC}"
    echo -e "URL: ${download_url}"

    # Download binary
    curl -fsSL "$download_url" -o "$temp_file"

    # Make executable
    chmod +x "$temp_file"

    # Install binary
    echo -e "${BLUE}Installing to ${INSTALL_DIR}/devdrop...${NC}"
    if [ "$INSTALL_DIR" = "/usr/local/bin" ]; then
        sudo mv "$temp_file" "${INSTALL_DIR}/devdrop"
    else
        mv "$temp_file" "${INSTALL_DIR}/devdrop"
    fi
}

# Verify installation
verify_installation() {
    if command -v devdrop >/dev/null 2>&1; then
        echo -e "${GREEN}✅ DevDrop installed successfully!${NC}"
        echo -e "${GREEN}Version: $(devdrop --version 2>/dev/null || echo 'unknown')${NC}"
    else
        echo -e "${YELLOW}⚠️  DevDrop installed but not in PATH${NC}"
        echo -e "Add ${INSTALL_DIR} to your PATH:"
        echo -e "${BLUE}  export PATH=\"${INSTALL_DIR}:\$PATH\"${NC}"
        echo -e "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
    fi
}

# Check prerequisites
check_prerequisites() {
    # Check if Docker is installed
    if ! command -v docker >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Docker not found. DevDrop requires Docker to be installed.${NC}"
        echo -e "Install Docker from: https://docs.docker.com/get-docker/"
        echo
    fi

    # Check if curl is available
    if ! command -v curl >/dev/null 2>&1; then
        echo -e "${RED}Error: curl is required but not installed${NC}" >&2
        exit 1
    fi
}

# Main installation function
main() {
    echo -e "${GREEN}🚀 DevDrop Installer${NC}"
    echo -e "Installing DevDrop - Docker-based development environments"
    echo

    # Check prerequisites
    check_prerequisites

    # Detect platform
    local platform
    platform=$(detect_platform)
    echo -e "${BLUE}Detected platform: ${platform}${NC}"

    # Check permissions
    check_permissions

    # Get download URL
    local download_url
    download_url=$(get_latest_release "$platform")

    # Create installation directory
    create_install_dir

    # Download and install
    install_binary "$platform" "$download_url"

    # Verify installation
    verify_installation

    echo
    echo -e "${GREEN}🎉 Installation complete!${NC}"
    echo
    echo -e "${BLUE}Get started with:${NC}"
    echo -e "  ${YELLOW}devdrop login${NC}    # Authenticate with DockerHub"
    echo -e "  ${YELLOW}devdrop init${NC}     # Create your development environment"
    echo -e "  ${YELLOW}devdrop commit${NC}   # Save your customizations"
    echo -e "  ${YELLOW}devdrop run${NC}      # Use your environment in any project"
    echo -e "  ${YELLOW}devdrop pull${NC}     # Pull latest version of your environment"
    echo
    echo -e "Documentation: https://github.com/${REPO}"
}

# Run main function
main "$@"