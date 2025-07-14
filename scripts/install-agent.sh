#!/bin/bash
# Copyright (c) 2024-2025, s0up and the autobrr contributors.
# SPDX-License-Identifier: GPL-2.0-or-later

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
DEFAULT_PORT=8200
DEFAULT_HOST="0.0.0.0"
INSTALL_DIR="/opt/netronome"
CONFIG_DIR="/etc/netronome"
SERVICE_NAME="netronome-agent"
GITHUB_REPO="autobrr/netronome"
USER_NAME="netronome"
AUTO_UPDATE_FLAG=""  # empty means prompt, "true" means enable, "false" means disable

# Function to print colored output
print_color() {
    color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to generate random API key
generate_api_key() {
    tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w 32 | head -n 1
}

# Function to get latest release URL
get_latest_release_url() {
    local arch=$(uname -m)
    local os="linux"
    
    # Map architecture names
    case "$arch" in
        x86_64)
            arch="x86_64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l)
            arch="armv7"
            ;;
        *)
            print_color $RED "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    # Get latest release URL from GitHub API
    local api_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local download_url=$(curl -s $api_url | grep "browser_download_url.*${os}_${arch}.tar.gz" | cut -d '"' -f 4)
    
    if [ -z "$download_url" ]; then
        print_color $RED "Failed to get latest release URL"
        exit 1
    fi
    
    echo "$download_url"
}

# Function to detect network interfaces
get_network_interfaces() {
    ip -o link show | awk -F': ' '{print $2}' | grep -v lo
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    print_color $RED "This script must be run as root"
    exit 1
fi

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--uninstall)
            UNINSTALL=true
            shift
            ;;
        --update)
            UPDATE=true
            shift
            ;;
        --auto-update)
            shift
            if [ "$1" = "true" ] || [ "$1" = "false" ]; then
                AUTO_UPDATE_FLAG=$1
                shift
            else
                AUTO_UPDATE_FLAG="true"
            fi
            ;;
        -h|--help)
            echo "Netronome Agent Installation Script"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  -u, --uninstall          Uninstall the agent"
            echo "  --update                 Update the agent to the latest version"
            echo "  --auto-update [true|false] Enable/disable automatic daily updates (default: prompt)"
            echo "  -h, --help               Show this help message"
            echo ""
            echo "The script will interactively prompt for:"
            echo "  - Network interface to monitor"
            echo "  - API key (or generate one)"
            echo "  - Host/IP to listen on"
            echo "  - Port number"
            exit 0
            ;;
        *)
            print_color $RED "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Uninstall function
if [ "$UNINSTALL" = true ]; then
    print_color $YELLOW "Uninstalling Netronome Agent..."
    
    # Stop and disable service
    systemctl stop $SERVICE_NAME 2>/dev/null || true
    systemctl disable $SERVICE_NAME 2>/dev/null || true
    
    # Stop and disable auto-update timer if it exists
    systemctl stop $SERVICE_NAME-update.timer 2>/dev/null || true
    systemctl disable $SERVICE_NAME-update.timer 2>/dev/null || true
    
    # Remove service files
    rm -f /etc/systemd/system/$SERVICE_NAME.service
    rm -f /etc/systemd/system/$SERVICE_NAME-update.service
    rm -f /etc/systemd/system/$SERVICE_NAME-update.timer
    systemctl daemon-reload
    
    # Remove files
    rm -rf $INSTALL_DIR
    rm -rf $CONFIG_DIR
    
    # Remove user (optional, commented out for safety)
    # userdel -r $USER_NAME 2>/dev/null || true
    
    print_color $GREEN "Netronome Agent uninstalled successfully!"
    exit 0
fi

# Update function
if [ "$UPDATE" = true ]; then
    print_color $YELLOW "Updating Netronome Agent..."
    
    # Check if agent is installed
    if [ ! -f "$INSTALL_DIR/netronome" ]; then
        print_color $RED "Netronome agent is not installed at $INSTALL_DIR/netronome"
        exit 1
    fi
    
    # Use the built-in update command
    if $INSTALL_DIR/netronome update; then
        print_color $GREEN "Update completed successfully!"
        
        # Restart service if it's running
        if systemctl is-active --quiet $SERVICE_NAME; then
            print_color $YELLOW "Restarting agent service..."
            systemctl restart $SERVICE_NAME
            print_color $GREEN "Service restarted"
        fi
    else
        print_color $RED "Update failed!"
        exit 1
    fi
    
    exit 0
fi

# Installation
print_color $BLUE "==================================="
print_color $BLUE "Netronome Agent Installation Script"
print_color $BLUE "==================================="
echo ""

# Check for required dependencies
print_color $YELLOW "Checking dependencies..."
if ! command -v vnstat &> /dev/null; then
    print_color $RED "vnstat is not installed. Please install vnstat first."
    print_color $YELLOW "On Ubuntu/Debian: apt-get install vnstat"
    print_color $YELLOW "On CentOS/RHEL: yum install vnstat"
    exit 1
fi

# Get network interface
print_color $YELLOW "\nAvailable network interfaces:"
interfaces=$(get_network_interfaces)
echo "$interfaces" | nl -w2 -s'. '

echo ""
read -p "Enter the interface to monitor (leave empty for all): " INTERFACE

# Get API key
echo ""
print_color $YELLOW "API Key Configuration:"
print_color $YELLOW "1. Generate a random API key"
print_color $YELLOW "2. Enter your own API key"
print_color $YELLOW "3. No authentication (not recommended)"
echo ""
read -p "Select an option [1-3]: " API_KEY_OPTION

case $API_KEY_OPTION in
    1)
        API_KEY=$(generate_api_key)
        print_color $GREEN "Generated API Key: $API_KEY"
        print_color $YELLOW "⚠️  Save this key! You'll need it when adding the agent in Netronome."
        ;;
    2)
        read -p "Enter your API key: " API_KEY
        ;;
    3)
        API_KEY=""
        print_color $YELLOW "⚠️  Warning: Agent will run without authentication!"
        ;;
    *)
        print_color $RED "Invalid option"
        exit 1
        ;;
esac

# Get host and port
echo ""
read -p "Enter the host/IP to listen on (default: $DEFAULT_HOST): " HOST
HOST=${HOST:-$DEFAULT_HOST}

read -p "Enter the port number (default: $DEFAULT_PORT): " PORT
PORT=${PORT:-$DEFAULT_PORT}

# Create user if it doesn't exist
if ! id "$USER_NAME" &>/dev/null; then
    print_color $YELLOW "\nCreating user: $USER_NAME"
    useradd -r -s /bin/false -d /nonexistent -c "Netronome Agent" $USER_NAME
fi

# Create directories
print_color $YELLOW "\nCreating directories..."
mkdir -p $INSTALL_DIR
mkdir -p $CONFIG_DIR
chown $USER_NAME:$USER_NAME $CONFIG_DIR

# Download and install binary
print_color $YELLOW "\nDownloading Netronome agent..."
DOWNLOAD_URL=$(get_latest_release_url)
print_color $BLUE "Downloading from: $DOWNLOAD_URL"

cd /tmp
curl -L -o netronome-agent.tar.gz "$DOWNLOAD_URL"
tar -xzf netronome-agent.tar.gz
cp netronome $INSTALL_DIR/
chmod +x $INSTALL_DIR/netronome
chown $USER_NAME:$USER_NAME $INSTALL_DIR/netronome
rm -f netronome-agent.tar.gz netronome

# Create configuration file
print_color $YELLOW "\nCreating configuration..."
cat > $CONFIG_DIR/agent.toml << EOF
# Netronome Agent Configuration

[agent]
host = "$HOST"
port = $PORT
interface = "$INTERFACE"
api_key = "$API_KEY"

[logging]
level = "info"
EOF

chown $USER_NAME:$USER_NAME $CONFIG_DIR/agent.toml
chmod 600 $CONFIG_DIR/agent.toml

# Create systemd service
print_color $YELLOW "\nCreating systemd service..."
cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=Netronome vnstat Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$USER_NAME
Group=$USER_NAME
ExecStart=$INSTALL_DIR/netronome agent --config $CONFIG_DIR/agent.toml
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$CONFIG_DIR

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and start service
print_color $YELLOW "\nStarting service..."
systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl start $SERVICE_NAME

# Check service status
if systemctl is-active --quiet $SERVICE_NAME; then
    print_color $GREEN "\n✅ Netronome Agent installed and started successfully!"
    
    # Display connection information
    print_color $BLUE "\n==================================="
    print_color $BLUE "Agent Information:"
    print_color $BLUE "==================================="
    print_color $GREEN "URL: http://$HOST:$PORT"
    if [ -n "$API_KEY" ]; then
        print_color $GREEN "API Key: $API_KEY"
    fi
    print_color $YELLOW "\nAdd this agent in Netronome with the above URL and API key."
    
    # Show service commands
    print_color $BLUE "\n==================================="
    print_color $BLUE "Useful Commands:"
    print_color $BLUE "==================================="
    echo "View logs:        journalctl -u $SERVICE_NAME -f"
    echo "Check status:     systemctl status $SERVICE_NAME"
    echo "Restart service:  systemctl restart $SERVICE_NAME"
    echo "Stop service:     systemctl stop $SERVICE_NAME"
    echo "Update agent:     $INSTALL_DIR/netronome update"
    echo "Check version:    $INSTALL_DIR/netronome version"
    echo ""
    
    # Prompt for auto-update setup (unless flag was provided)
    if [ -z "$AUTO_UPDATE_FLAG" ]; then
        echo ""
        read -p "Would you like to enable automatic daily updates? (y/n): " AUTO_UPDATE
    elif [ "$AUTO_UPDATE_FLAG" = "true" ]; then
        AUTO_UPDATE="y"
    else
        AUTO_UPDATE="n"
    fi
    
    if [ "$AUTO_UPDATE" = "y" ] || [ "$AUTO_UPDATE" = "Y" ]; then
        print_color $YELLOW "\nSetting up automatic updates..."
        
        # Create systemd timer for daily updates
        cat > /etc/systemd/system/$SERVICE_NAME-update.timer << EOF
[Unit]
Description=Daily update check for netronome-agent

[Timer]
OnCalendar=daily
RandomizedDelaySec=4h
Persistent=true

[Install]
WantedBy=timers.target
EOF
        
        # Create update service
        cat > /etc/systemd/system/$SERVICE_NAME-update.service << EOF
[Unit]
Description=Update netronome-agent

[Service]
Type=oneshot
ExecStart=/bin/bash -c 'curl -sL https://raw.githubusercontent.com/autobrr/netronome/main/scripts/install-agent.sh | bash -s -- --update'
User=root
StandardOutput=journal
StandardError=journal
EOF
        
        systemctl daemon-reload
        systemctl enable --now $SERVICE_NAME-update.timer
        
        print_color $GREEN "✅ Automatic daily updates enabled!"
        print_color $YELLOW "Updates will run daily with a random delay of up to 4 hours."
        print_color $YELLOW "Check update timer: systemctl status $SERVICE_NAME-update.timer"
        print_color $YELLOW "Check last update: journalctl -u $SERVICE_NAME-update"
    fi
else
    print_color $RED "\n❌ Failed to start Netronome Agent"
    print_color $YELLOW "Check logs with: journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi