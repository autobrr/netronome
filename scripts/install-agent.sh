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
GITHUB_REPO="autobrr/netronome"
USER_NAME="netronome"
AUTO_UPDATE_FLAG=""  # empty means prompt, "true" means enable, "false" means disable

# Detect OS
OS_TYPE=$(uname -s)
case "$OS_TYPE" in
    Linux*)
        OS="linux"
        INSTALL_DIR="/opt/netronome"
        CONFIG_DIR="/etc/netronome"
        SERVICE_NAME="netronome-agent"
        ;;
    Darwin*)
        OS="darwin"
        INSTALL_DIR="/usr/local/opt/netronome"
        CONFIG_DIR="/usr/local/etc/netronome"
        SERVICE_NAME="com.netronome.agent"
        ;;
    *)
        echo "Unsupported OS: $OS_TYPE"
        exit 1
        ;;
esac

# Function to print colored output
print_color() {
    color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to generate random API key
generate_api_key() {
    if command -v openssl &> /dev/null; then
        openssl rand -hex 16
    else
        tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w 32 | head -n 1
    fi
}

# Function to get latest release URL
get_latest_release_url() {
    local arch=$(uname -m)
    
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
    local download_url=$(curl -s $api_url | grep "browser_download_url.*${OS}_${arch}.tar.gz" | cut -d '"' -f 4)
    
    if [ -z "$download_url" ]; then
        print_color $RED "Failed to get latest release URL"
        exit 1
    fi
    
    echo "$download_url"
}

# Function to detect network interfaces
get_network_interfaces() {
    if [ "$OS" = "darwin" ]; then
        # macOS: Use ifconfig to list interfaces
        ifconfig -l | tr ' ' '\n' | grep -v -E '^(lo[0-9]*|bridge[0-9]*|p2p[0-9]*|awdl[0-9]*|llw[0-9]*|utun[0-9]*)$'
    else
        # Linux: Use ip command
        ip -o link show | awk -F': ' '{print $2}' | grep -v lo
    fi
}

# Check if running as root (different for macOS)
check_root() {
    if [ "$OS" = "darwin" ]; then
        if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then 
            print_color $YELLOW "This script requires sudo privileges for installation."
            print_color $YELLOW "You will be prompted for your password when needed."
            # Test sudo access early but don't exit if it fails
            if ! sudo -v 2>/dev/null; then
                print_color $RED "Failed to obtain sudo privileges"
                exit 1
            fi
        fi
    else
        if [ "$EUID" -ne 0 ]; then 
            print_color $RED "This script must be run as root on Linux"
            print_color $YELLOW "Please run: sudo $0"
            exit 1
        fi
    fi
}

# Create user function for cross-platform
create_user() {
    local username=$1
    
    if [ "$OS" = "darwin" ]; then
        # Check if user exists on macOS
        if ! dscl . -read /Users/$username &>/dev/null; then
            print_color $YELLOW "\nCreating user: $username"
            # Get next available UID
            local uid=$(dscl . -list /Users UniqueID | awk '{print $2}' | sort -n | tail -1)
            uid=$((uid + 1))
            
            # Create user on macOS
            sudo dscl . -create /Users/$username
            sudo dscl . -create /Users/$username UniqueID $uid
            sudo dscl . -create /Users/$username PrimaryGroupID 20
            sudo dscl . -create /Users/$username UserShell /usr/bin/false
            sudo dscl . -create /Users/$username RealName "Netronome Agent"
            sudo dscl . -create /Users/$username NFSHomeDirectory /var/empty
        fi
    else
        # Linux user creation
        if ! id "$username" &>/dev/null; then
            print_color $YELLOW "\nCreating user: $username"
            useradd -r -s /bin/false -d /nonexistent -c "Netronome Agent" $username
        fi
    fi
}

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

# Check root/sudo privileges for uninstall and update operations
if [ "$UNINSTALL" = true ] || [ "$UPDATE" = true ]; then
    check_root
fi

# Uninstall function
if [ "$UNINSTALL" = true ]; then
    print_color $YELLOW "Uninstalling Netronome Agent..."
    
    if [ "$OS" = "darwin" ]; then
        # macOS uninstall
        if [ -f "/Library/LaunchDaemons/$SERVICE_NAME.plist" ]; then
            sudo launchctl unload -w /Library/LaunchDaemons/$SERVICE_NAME.plist 2>/dev/null || true
            sudo rm -f /Library/LaunchDaemons/$SERVICE_NAME.plist
        fi
        
        # Remove auto-update launchd job if exists
        if [ -f "/Library/LaunchDaemons/$SERVICE_NAME.update.plist" ]; then
            sudo launchctl unload -w /Library/LaunchDaemons/$SERVICE_NAME.update.plist 2>/dev/null || true
            sudo rm -f /Library/LaunchDaemons/$SERVICE_NAME.update.plist
        fi
    else
        # Linux uninstall
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
    fi
    
    # Remove files
    sudo rm -rf $INSTALL_DIR
    sudo rm -rf $CONFIG_DIR
    
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
    if sudo $INSTALL_DIR/netronome update; then
        print_color $GREEN "Update completed successfully!"
        
        # Restart service
        if [ "$OS" = "darwin" ]; then
            if sudo launchctl list | grep -q $SERVICE_NAME; then
                print_color $YELLOW "Restarting agent service..."
                sudo launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist
                sudo launchctl load -w /Library/LaunchDaemons/$SERVICE_NAME.plist
                print_color $GREEN "Service restarted"
            fi
        else
            if systemctl is-active --quiet $SERVICE_NAME; then
                print_color $YELLOW "Restarting agent service..."
                systemctl restart $SERVICE_NAME
                print_color $GREEN "Service restarted"
            fi
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

# Check if we have a controlling terminal for interactive input
if [ -t 1 ] && [ -t 2 ] && [ -e /dev/tty ]; then
    # We can be interactive by reading from /dev/tty
    INTERACTIVE_MODE=true
    INPUT_SOURCE="/dev/tty"
else
    # Fallback to non-interactive mode
    print_color $YELLOW "Running in non-interactive mode with auto-selected defaults."
    print_color $YELLOW "For interactive installation:"
    print_color $BLUE "  curl -sL https://raw.githubusercontent.com/autobrr/netronome/main/scripts/install-agent.sh -o install.sh && bash install.sh"
    echo ""
    INTERACTIVE_MODE=false
    INPUT_SOURCE="/dev/stdin"
fi

# Check root/sudo privileges
check_root

# Check for required dependencies
print_color $YELLOW "Checking dependencies..."
if ! command -v vnstat &> /dev/null; then
    print_color $RED "vnstat is not installed. Please install vnstat first."
    if [ "$OS" = "darwin" ]; then
        print_color $YELLOW "On macOS: brew install vnstat"
    else
        print_color $YELLOW "On Ubuntu/Debian: apt-get install vnstat"
        print_color $YELLOW "On CentOS/RHEL: yum install vnstat"
    fi
    exit 1
fi

# Get network interface
print_color $YELLOW "\nAvailable network interfaces:"
interfaces=$(get_network_interfaces)
echo "$interfaces" | nl -w2 -s'. '

echo ""
if [ "$INTERACTIVE_MODE" = true ]; then
    # Interactive mode - read from terminal
    read -p "Enter the interface to monitor (leave empty for all): " INTERFACE < "$INPUT_SOURCE"
else
    # Non-interactive mode - default to all interfaces
    INTERFACE=""
    echo "Interface: ${INTERFACE:-all} (auto-selected)"
fi

# Get API key
echo ""
print_color $YELLOW "API Key Configuration:"
print_color $YELLOW "1. Generate a random API key"
print_color $YELLOW "2. Enter your own API key"
print_color $YELLOW "3. No authentication (not recommended)"
echo ""
if [ "$INTERACTIVE_MODE" = true ]; then
    # Interactive mode - read from terminal
    read -p "Select an option [1-3]: " API_KEY_OPTION < "$INPUT_SOURCE"
else
    # Non-interactive mode - default to generating a random API key
    API_KEY_OPTION="1"
    echo "Selected option: $API_KEY_OPTION (auto-selected: Generate random API key)"
fi

case $API_KEY_OPTION in
    1)
        API_KEY=$(generate_api_key)
        print_color $GREEN "Generated API Key: $API_KEY"
        print_color $YELLOW "⚠️  Save this key! You'll need it when adding the agent in Netronome."
        ;;
    2)
        if [ "$INTERACTIVE_MODE" = true ]; then
            read -p "Enter your API key: " API_KEY < "$INPUT_SOURCE"
        else
            # In non-interactive mode, we can't get custom API key, fall back to generated
            API_KEY=$(generate_api_key)
            print_color $YELLOW "Cannot input custom API key in non-interactive mode, generated: $API_KEY"
        fi
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
if [ "$INTERACTIVE_MODE" = true ]; then
    # Interactive mode - read from terminal
    read -p "Enter the host/IP to listen on (default: $DEFAULT_HOST): " HOST < "$INPUT_SOURCE"
    HOST=${HOST:-$DEFAULT_HOST}
    
    read -p "Enter the port number (default: $DEFAULT_PORT): " PORT < "$INPUT_SOURCE"
    PORT=${PORT:-$DEFAULT_PORT}
else
    # Non-interactive mode - use defaults
    HOST=$DEFAULT_HOST
    PORT=$DEFAULT_PORT
    echo "Host: $HOST (auto-selected)"
    echo "Port: $PORT (auto-selected)"
fi

# Get disk filtering configuration
echo ""
if [ "$INTERACTIVE_MODE" = true ]; then
    print_color $YELLOW "Disk Monitoring Configuration (optional):"
    print_color $YELLOW "You can specify additional disk mounts to monitor or exclude certain mounts."
    echo ""
    
    # Disk includes
    read -p "Enter disk mounts to include (comma-separated, e.g., /mnt/storage,/mnt/backup): " DISK_INCLUDES_INPUT < "$INPUT_SOURCE"
    if [ -n "$DISK_INCLUDES_INPUT" ]; then
        # Convert comma-separated list to TOML array format
        DISK_INCLUDES=$(echo "$DISK_INCLUDES_INPUT" | sed 's/,/", "/g' | sed 's/^/["/' | sed 's/$/"]/')
    else
        DISK_INCLUDES="[]"
    fi
    
    # Disk excludes
    read -p "Enter disk mounts to exclude (comma-separated, e.g., /boot,/tmp): " DISK_EXCLUDES_INPUT < "$INPUT_SOURCE"
    if [ -n "$DISK_EXCLUDES_INPUT" ]; then
        # Convert comma-separated list to TOML array format
        DISK_EXCLUDES=$(echo "$DISK_EXCLUDES_INPUT" | sed 's/,/", "/g' | sed 's/^/["/' | sed 's/$/"]/')
    else
        DISK_EXCLUDES="[]"
    fi
else
    # Non-interactive mode - use defaults
    DISK_INCLUDES="[]"
    DISK_EXCLUDES="[]"
    echo "Disk includes: none (auto-selected)"
    echo "Disk excludes: none (auto-selected)"
fi

# Get Tailscale configuration
echo ""
TAILSCALE_ENABLED="false"
TAILSCALE_METHOD=""
TAILSCALE_AUTH_KEY=""
TAILSCALE_HOSTNAME=""

if [ "$INTERACTIVE_MODE" = true ]; then
    print_color $YELLOW "Tailscale Configuration (optional):"
    print_color $YELLOW "Tailscale provides secure, encrypted connectivity without exposing ports to the internet."
    echo ""
    
    read -p "Do you want to enable Tailscale for secure connectivity? (y/n): " ENABLE_TAILSCALE < "$INPUT_SOURCE"
    if [ "$ENABLE_TAILSCALE" = "y" ] || [ "$ENABLE_TAILSCALE" = "Y" ]; then
        TAILSCALE_ENABLED="true"
        
        echo ""
        print_color $YELLOW "Tailscale Method:"
        print_color $YELLOW "1. Use host's existing Tailscale (no new machine in Tailscale admin)"
        print_color $YELLOW "2. Create dedicated Tailscale node (requires auth key)"
        echo ""
        
        read -p "Select method [1-2]: " TAILSCALE_METHOD_OPTION < "$INPUT_SOURCE"
        case $TAILSCALE_METHOD_OPTION in
            1)
                TAILSCALE_METHOD="host"
                print_color $GREEN "Using host's existing Tailscale connection"
                ;;
            2)
                TAILSCALE_METHOD="tsnet"
                echo ""
                read -p "Enter your Tailscale auth key (required): " TAILSCALE_AUTH_KEY < "$INPUT_SOURCE"
                if [ -z "$TAILSCALE_AUTH_KEY" ]; then
                    print_color $RED "Auth key is required for tsnet mode"
                    TAILSCALE_ENABLED="false"
                    TAILSCALE_METHOD=""
                fi
                ;;
            *)
                print_color $RED "Invalid option, Tailscale will not be configured"
                TAILSCALE_ENABLED="false"
                ;;
        esac
        
        if [ "$TAILSCALE_ENABLED" = "true" ]; then
            echo ""
            read -p "Enter custom Tailscale hostname (leave empty for default): " TAILSCALE_HOSTNAME < "$INPUT_SOURCE"
        fi
    fi
else
    # Non-interactive mode - Tailscale disabled by default
    echo "Tailscale: disabled (auto-selected)"
fi

# Create user if it doesn't exist
create_user $USER_NAME

# Create directories
print_color $YELLOW "\nCreating directories..."
sudo mkdir -p $INSTALL_DIR
sudo mkdir -p $CONFIG_DIR

if [ "$OS" = "darwin" ]; then
    sudo chown $USER_NAME:staff $CONFIG_DIR
else
    sudo chown $USER_NAME:$USER_NAME $CONFIG_DIR
fi

# Download and install binary
print_color $YELLOW "\nInstalling Netronome agent..."

# Check if we're in test mode with a local binary
if [ -f "$TEST_DIR/netronome" ]; then
    print_color $BLUE "Using local test binary"
    sudo cp "$TEST_DIR/netronome" $INSTALL_DIR/
else
    DOWNLOAD_URL=$(get_latest_release_url)
    print_color $BLUE "Downloading from: $DOWNLOAD_URL"
    
    # Create temp directory for extraction
    TEMP_EXTRACT_DIR="/tmp/netronome-extract-$$"
    mkdir -p "$TEMP_EXTRACT_DIR"
    cd "$TEMP_EXTRACT_DIR"
    
    curl -L -o netronome-agent.tar.gz "$DOWNLOAD_URL"
    tar -xzf netronome-agent.tar.gz
    sudo cp netronome $INSTALL_DIR/
    
    # Clean up temp directory
    cd /tmp
    rm -rf "$TEMP_EXTRACT_DIR"
fi

sudo chmod +x $INSTALL_DIR/netronome

if [ "$OS" = "darwin" ]; then
    sudo chown $USER_NAME:staff $INSTALL_DIR/netronome
else
    sudo chown $USER_NAME:$USER_NAME $INSTALL_DIR/netronome
fi

# No cleanup needed as we use temp directory

# Create configuration file
print_color $YELLOW "\nCreating configuration..."
sudo tee $CONFIG_DIR/agent.toml > /dev/null << EOF
# Netronome Agent Configuration

[agent]
host = "$HOST"
port = $PORT
interface = "$INTERFACE"
api_key = "$API_KEY"
disk_includes = $DISK_INCLUDES
disk_excludes = $DISK_EXCLUDES

[logging]
level = "info"
EOF

# Add Tailscale configuration if enabled
if [ "$TAILSCALE_ENABLED" = "true" ]; then
    sudo tee -a $CONFIG_DIR/agent.toml > /dev/null << EOF

[tailscale]
enabled = true
method = "$TAILSCALE_METHOD"
EOF
    
    if [ "$TAILSCALE_METHOD" = "tsnet" ]; then
        sudo tee -a $CONFIG_DIR/agent.toml > /dev/null << EOF
auth_key = "$TAILSCALE_AUTH_KEY"
EOF
    fi
    
    if [ -n "$TAILSCALE_HOSTNAME" ]; then
        sudo tee -a $CONFIG_DIR/agent.toml > /dev/null << EOF
hostname = "$TAILSCALE_HOSTNAME"
EOF
    fi
fi

if [ "$OS" = "darwin" ]; then
    sudo chown $USER_NAME:staff $CONFIG_DIR/agent.toml
else
    sudo chown $USER_NAME:$USER_NAME $CONFIG_DIR/agent.toml
fi
sudo chmod 600 $CONFIG_DIR/agent.toml

# Create service
print_color $YELLOW "\nCreating service..."

if [ "$OS" = "darwin" ]; then
    # Create launchd plist for macOS
    sudo tee /Library/LaunchDaemons/$SERVICE_NAME.plist > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$SERVICE_NAME</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/netronome</string>
        <string>agent</string>
        <string>--config</string>
        <string>$CONFIG_DIR/agent.toml</string>
    </array>
    <key>UserName</key>
    <string>$USER_NAME</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>Crashed</key>
        <true/>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/netronome-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/netronome-agent.error.log</string>
</dict>
</plist>
EOF
    
    # Load the service
    print_color $YELLOW "\nStarting service..."
    sudo launchctl load -w /Library/LaunchDaemons/$SERVICE_NAME.plist
    
    # Check if service is running
    sleep 2
    if sudo launchctl list | grep -q $SERVICE_NAME; then
        SERVICE_RUNNING=true
    else
        SERVICE_RUNNING=false
    fi
else
    # Create systemd service for Linux
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
        SERVICE_RUNNING=true
    else
        SERVICE_RUNNING=false
    fi
fi

# Display results
if [ "$SERVICE_RUNNING" = true ]; then
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
    
    if [ "$OS" = "darwin" ]; then
        echo "View logs:        tail -f /tmp/netronome-agent.log"
        echo "Check status:     sudo launchctl list | grep $SERVICE_NAME"
        echo "Stop service:     sudo launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist"
        echo "Start service:    sudo launchctl load -w /Library/LaunchDaemons/$SERVICE_NAME.plist"
    else
        echo "View logs:        journalctl -u $SERVICE_NAME -f"
        echo "Check status:     systemctl status $SERVICE_NAME"
        echo "Restart service:  systemctl restart $SERVICE_NAME"
        echo "Stop service:     systemctl stop $SERVICE_NAME"
    fi
    
    echo "Update agent:     sudo $INSTALL_DIR/netronome update"
    echo "Check version:    $INSTALL_DIR/netronome version"
    echo "Uninstall agent:  curl -sL https://raw.githubusercontent.com/autobrr/netronome/main/scripts/install-agent.sh | bash -s -- --uninstall"
    echo ""
    
    # Prompt for auto-update setup (unless flag was provided)
    if [ -z "$AUTO_UPDATE_FLAG" ]; then
        echo ""
        if [ "$INTERACTIVE_MODE" = true ]; then
            # Interactive mode - read from terminal
            read -p "Would you like to enable automatic daily updates? (y/n): " AUTO_UPDATE < "$INPUT_SOURCE"
        else
            # Non-interactive mode - default to yes for auto-updates
            AUTO_UPDATE="y"
            echo "Auto-update: $AUTO_UPDATE (auto-selected: yes)"
        fi
    elif [ "$AUTO_UPDATE_FLAG" = "true" ]; then
        AUTO_UPDATE="y"
    else
        AUTO_UPDATE="n"
    fi
    
    if [ "$AUTO_UPDATE" = "y" ] || [ "$AUTO_UPDATE" = "Y" ]; then
        print_color $YELLOW "\nSetting up automatic updates..."
        
        if [ "$OS" = "darwin" ]; then
            # Create launchd plist for auto-update
            sudo tee /Library/LaunchDaemons/$SERVICE_NAME.update.plist > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$SERVICE_NAME.update</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>-c</string>
        <string>curl -sL https://raw.githubusercontent.com/autobrr/netronome/main/scripts/install-agent.sh | bash -s -- --update</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>3</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>/var/log/netronome-agent-update.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/netronome-agent-update.error.log</string>
</dict>
</plist>
EOF
            
            sudo launchctl load -w /Library/LaunchDaemons/$SERVICE_NAME.update.plist
            print_color $GREEN "✅ Automatic daily updates enabled!"
            print_color $YELLOW "Updates will run daily at 3:00 AM."
            print_color $YELLOW "Check update logs: tail -f /var/log/netronome-agent-update.log"
        else
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
    fi
else
    print_color $RED "\n❌ Failed to start Netronome Agent"
    if [ "$OS" = "darwin" ]; then
        print_color $YELLOW "Check logs with: tail -50 /tmp/netronome-agent.error.log"
    else
        print_color $YELLOW "Check logs with: journalctl -u $SERVICE_NAME -n 50"
    fi
    exit 1
fi