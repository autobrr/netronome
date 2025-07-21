#!/bin/bash

# Netronome Agent Docker Installation Script
#
# This script sets up a Netronome monitoring agent in Docker.
# Currently uses the main netronome image with the 'agent' command.
#
# Future Enhancement: When dedicated agent images become available
# (e.g., ghcr.io/autobrr/netronome-agent), this script will be updated
# to use those lighter-weight, agent-specific images for better
# resource efficiency and security.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
DEFAULT_HOST="0.0.0.0"
DEFAULT_PORT="8200"
DEFAULT_INTERFACE=""

# Image configuration
# TODO: When agent-specific images are available, switch to:
# AGENT_IMAGE="ghcr.io/autobrr/netronome-agent:latest"
# AGENT_CMD=""
AGENT_IMAGE="ghcr.io/autobrr/netronome:latest"
AGENT_CMD="agent"

# Function to detect and configure the appropriate agent image
configure_agent_image() {
    # Check if dedicated agent image is available
    if docker manifest inspect ghcr.io/autobrr/netronome-agent:latest >/dev/null 2>&1; then
        print_info "Dedicated agent image detected, using optimized image"
        AGENT_IMAGE="ghcr.io/autobrr/netronome-agent:latest"
        AGENT_CMD=""
    else
        print_info "Using main image with agent command"
        AGENT_IMAGE="ghcr.io/autobrr/netronome:latest"
        AGENT_CMD="agent"
    fi
}

# Function to get security settings optimized for agent
get_security_settings() {
    # Agent-specific images can have more restrictive security settings
    if [[ "$AGENT_IMAGE" == *"netronome-agent"* ]]; then
        echo "--cap-drop=ALL --cap-add=NET_ADMIN --read-only --tmpfs /tmp --security-opt=no-new-privileges"
    else
        # Main image needs broader permissions for full functionality
        echo "--cap-drop=ALL --cap-add=NET_ADMIN --cap-add=SYS_ADMIN --read-only --tmpfs /tmp"
    fi
}

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

echo ""
echo "========================================"
echo "  Netronome Agent Docker Setup"
echo "========================================"
echo ""

# Configure the appropriate agent image
configure_agent_image

print_info "Using image: $AGENT_IMAGE"
if [ ! -z "$AGENT_CMD" ]; then
    print_info "Agent command: $AGENT_CMD"
fi
echo ""

# Interactive prompts
read -p "Container name (default: netronome-agent): " CONTAINER_NAME
CONTAINER_NAME=${CONTAINER_NAME:-netronome-agent}

read -p "Port to expose (default: $DEFAULT_PORT): " PORT
PORT=${PORT:-$DEFAULT_PORT}

# API Key prompt
echo ""
print_info "API Key Setup"
echo "An API key provides authentication for the agent."
echo "Leave empty to run without authentication (not recommended for production)."
read -p "API Key (press enter to generate random): " API_KEY

# Generate random API key if not provided
if [ -z "$API_KEY" ]; then
    API_KEY=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
    print_success "Generated API Key: $API_KEY"
    echo ""
    print_warning "Please save this API key securely. You'll need it to connect from Netronome."
    echo ""
    read -p "Press enter to continue..."
fi

# Network interface
echo ""
print_info "Network Interface"
echo "For Docker, you typically want to monitor the host network."
read -p "Use host network mode? (Y/n): " USE_HOST_NETWORK

if [[ ! "$USE_HOST_NETWORK" =~ ^[Nn]$ ]]; then
    NETWORK_MODE="--network host"
    PUBLISH_PORT=""
else
    NETWORK_MODE=""
    PUBLISH_PORT="-p $PORT:8200"
fi

# Get security settings optimized for the image type
SECURITY_SETTINGS=$(get_security_settings)

# Create Docker run command
DOCKER_CMD="docker run -d \\
    --name $CONTAINER_NAME \\
    --restart unless-stopped \\
    $NETWORK_MODE \\
    $PUBLISH_PORT \\
    -v /proc:/host/proc:ro \\
    -v /sys:/host/sys:ro \\
    $SECURITY_SETTINGS \\"

if [ ! -z "$API_KEY" ]; then
    DOCKER_CMD="$DOCKER_CMD
    -e NETRONOME__AGENT_API_KEY=\"$API_KEY\" \\"
fi

# Build final command with image and optional command
if [ ! -z "$AGENT_CMD" ]; then
    DOCKER_CMD="$DOCKER_CMD
    $AGENT_IMAGE $AGENT_CMD"
else
    DOCKER_CMD="$DOCKER_CMD
    $AGENT_IMAGE"
fi

echo ""
print_info "Docker command to run:"
echo ""
echo "$DOCKER_CMD"
echo ""

read -p "Run this command now? (y/N): " RUN_NOW

if [[ "$RUN_NOW" =~ ^[Yy]$ ]]; then
    print_info "Starting container..."
    eval "$DOCKER_CMD"
    
    sleep 3
    
    if docker ps | grep -q "$CONTAINER_NAME"; then
        print_success "Container started successfully!"
        echo ""
        echo "========================================"
        echo "  Setup Complete!"
        echo "========================================"
        echo ""
        if [[ "$USE_HOST_NETWORK" =~ ^[Nn]$ ]]; then
            echo "Agent URL: http://localhost:$PORT"
        else
            echo "Agent URL: http://<host-ip>:8200"
        fi
        if [ ! -z "$API_KEY" ]; then
            echo "API Key: $API_KEY"
        fi
        echo ""
        echo "Image Info:"
        echo "  - Image: $AGENT_IMAGE"
        if [[ "$AGENT_IMAGE" == *"netronome-agent"* ]]; then
            echo "  - Type: Dedicated agent image (optimized)"
        else
            echo "  - Type: Main image with agent command"
        fi
        echo ""
        echo "Container commands:"
        echo "  - Logs: docker logs -f $CONTAINER_NAME"
        echo "  - Stop: docker stop $CONTAINER_NAME"
        echo "  - Start: docker start $CONTAINER_NAME"
        echo "  - Remove: docker rm -f $CONTAINER_NAME"
        echo "  - Update: docker pull $AGENT_IMAGE && docker restart $CONTAINER_NAME"
    else
        print_error "Container failed to start. Check logs with: docker logs $CONTAINER_NAME"
        exit 1
    fi
else
    echo ""
    print_info "You can run the command later to start the agent."
fi

echo ""
echo "========================================"
echo "  Migration Notes"
echo "========================================"
echo ""
echo "Current Setup: Using main netronome image with 'agent' command"
echo ""
echo "Future Enhancement: When dedicated agent images become available:"
echo "  - Lighter weight: Smaller image size with only agent components"
echo "  - Better security: More restrictive container permissions"
echo "  - Optimized performance: Agent-specific optimizations"
echo "  - Automatic detection: This script will auto-detect and use them"
echo ""
echo "The migration will be seamless - just re-run this script when"
echo "agent-specific images are released."