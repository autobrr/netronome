#!/bin/bash

# Netronome vnstat Agent Docker Installation Script

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
echo "  Netronome vnstat Agent Docker Setup"
echo "========================================"
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

# Create Docker run command
DOCKER_CMD="docker run -d \\
    --name $CONTAINER_NAME \\
    --restart unless-stopped \\
    $NETWORK_MODE \\
    $PUBLISH_PORT \\
    -v /proc:/host/proc:ro \\
    -v /sys:/host/sys:ro \\"

if [ ! -z "$API_KEY" ]; then
    DOCKER_CMD="$DOCKER_CMD
    -e NETRONOME__AGENT_API_KEY=\"$API_KEY\" \\"
fi

DOCKER_CMD="$DOCKER_CMD
    ghcr.io/autobrr/netronome:latest agent"

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
        echo "Container commands:"
        echo "  - Logs: docker logs -f $CONTAINER_NAME"
        echo "  - Stop: docker stop $CONTAINER_NAME"
        echo "  - Start: docker start $CONTAINER_NAME"
        echo "  - Remove: docker rm -f $CONTAINER_NAME"
    else
        print_error "Container failed to start. Check logs with: docker logs $CONTAINER_NAME"
        exit 1
    fi
else
    echo ""
    print_info "You can run the command later to start the agent."
fi