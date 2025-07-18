<h1 align="center">Netronome</h1>

<p align="center">
  <img src=".github/assets/netronome_dashboard.png" alt="Netronome">
</p>

Netronome (Network Metronome) is a modern network performance testing and monitoring tool with a clean, intuitive web interface. It offers both scheduled and on-demand speed tests with detailed visualizations and historical tracking.

## 📑 Table of Contents

- [Features](#-features)
- [External Dependencies](#-external-dependencies)
- [Getting Started](#-getting-started)
  - [Linux Generic](#linux-generic)
  - [Docker Installation](#docker-installation)
- [Configuration](#️-configuration)
  - [Configuration File](#configuration-file-configtoml)
    - [LibreSpeed Configuration](#librespeed-configuration)
  - [System Monitoring](#system-monitoring)
  - [Tailscale Integration](#tailscale-integration)
  - [Environment Variables](#environment-variables)
  - [Database](#database)
  - [Authentication](#authentication)
  - [Scheduling Intervals](#scheduling-intervals)
  - [Packet Loss Monitoring](#packet-loss-monitoring)
  - [GeoIP Configuration](#geoip-configuration)
  - [Notifications](#notifications)
  - [CLI Commands](#cli-commands)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

- **Speed Testing**
  - Support for Speedtest.net, iperf3 servers, and LibreSpeed
  - Real-time test progress visualization
  - Latency and jitter measurements

- **Network Diagnostics**
  - **Unified Interface**: Seamless switching between single traceroute tests and continuous monitoring
  - **Traceroute**: Real-time hop discovery with cross-platform support (Linux/macOS/Windows)
  - **Packet Loss Monitoring**: Continuous ICMP ping monitoring with flexible scheduling options
  - **MTR Integration**: Advanced hop-by-hop analysis with packet loss statistics per hop
  - **Smart Fallback**: Automatic fallback from MTR to standard ping when privileges unavailable
  - **GeoIP Integration**: Country flags and ASN information for network path visualization
  - **Cross-tab Navigation**: Easy flow between traceroute results and monitor creation

- **Comprehensive System Monitoring**
  - **Distributed Agent Architecture**: Deploy lightweight agents on remote servers for complete system visibility
  - **Real-time SSE Streaming**: Live system metrics via Server-Sent Events
  - **Multi-server Support**: Monitor multiple servers from one centralized dashboard
  - **Bandwidth Monitoring**: Real-time and historical network usage with vnstat integration
  - **Hardware Monitoring**: CPU usage, memory stats, disk usage, and temperature sensors
  - **System Information**: Hostname, kernel version, uptime, network interfaces, and more
  - **Historical Tracking**: Store and visualize all metrics over time
  - **Auto-reconnection**: Agents automatically reconnect with exponential backoff

- **Monitoring & Visualization**
  - **Speed Test History**: Interactive charts with customizable time ranges (1d, 3d, 1w, 1m, all)
  - **Packet Loss Trends**: Historical packet loss and RTT monitoring with performance charts
  - **Real-time Status**: Live monitoring status with progress tracking and health indicators
  - **Monitor Management**: Start/stop/edit monitors with schedule badge visualization

- **Scheduling & Automation**
  - **Speed Tests**: Automated testing with flexible cron-like scheduling
  - **Packet Loss Monitors**: Interval-based (10 seconds to 24 hours) or exact-time scheduling
  - **Auto-start Option**: "Start monitoring immediately" for new monitors
  - **Multiple Schedule Types**: Choose between regular intervals or daily exact times

- **Modern Interface**
  - **Responsive Design**: Optimized for both desktop and mobile devices
  - **Real-time Progress**: Live updates for iperf3 tests with animated progress indicators
  - **Dark Mode**: Fully optimized dark theme with consistent styling
  - **Interactive Visualizations**: Dynamic charts with smooth animations and transitions
  - **Unified Navigation**: Seamless mode switching between different test types
  - **Form Validation**: Real-time input validation with helpful error messages

- **Flexible Authentication**
  - Built-in user authentication
  - OpenID Connect support

## 📦 External Dependencies

Netronome requires the following external tools for full functionality:

### Required for Speed Tests

- **iperf3** - Required for iperf3 speed testing (automatically included in Docker). Note: Jitter measurement is not supported for iperf3 tests
- **librespeed-cli** - Required for LibreSpeed testing (automatically included in Docker)
- **traceroute** - Required for network diagnostics (automatically included in Docker, usually pre-installed on most systems)

### Required for Development

- **air** - Go live reload tool for `make watch` command
- **tmux** - Terminal multiplexer for `make dev` command
- **pnpm** - Package manager for frontend dependencies

## 🚀 Getting Started

### Linux Generic

Download the latest release, or download the source code and build it yourself using make build.

```bash
wget $(curl -s https://api.github.com/repos/autobrr/netronome/releases/latest | grep download | grep linux_x86_64 | cut -d\" -f4)
```

**Note**: Release binaries still include most temperature monitoring (CPU, NVMe, battery) via gopsutil. Only SATA/HDD temperatures and disk model names require building from source with SMART support. See [Building from Source](#building-from-source) for details.

#### Unpack

Run with root or sudo. If you do not have root, or are on a shared system, place the binaries somewhere in your home directory like ~/.bin.

```bash
tar -C /usr/local/bin -xzf netronome*.tar.gz
```

This will extract both netronome and netronomectl to /usr/local/bin. Note: If the command fails, prefix it with sudo and re-run again.
Systemd (Recommended)

On Linux-based systems, it is recommended to run netronome as a sort of service with auto-restarting capabilities, in order to account for potential downtime. The most common way is to do it via systemd.

You will need to create a service file in /etc/systemd/system/ called netronome.service.

```bash
touch /etc/systemd/system/netronome@.service
```

Then place the following content inside the file (e.g. via nano/vim/ed):

```
[Unit]
Description=netronome service for %i
After=syslog.target network-online.target

[Service]
Type=simple
User=%i
Group=%i
ExecStart=/usr/bin/netronome --config=/home/%i/.config/netronome/config.toml

[Install]
WantedBy=multi-user.target
```

Start the service. Enable will make it startup on reboot.

```bash
systemctl enable -q --now --user netronome@$USER
```

By default, the configuration is set to listen on 127.0.0.1. While netronome works fine as is exposed to the internet, it is recommended to use a reverse proxy like nginx, caddy or traefik.

If you are not running a reverse proxy change host in the config.toml to 0.0.0.0.

### Docker Installation

For containerized deployment see [docker-compose.yml](docker-compose.yml) and [docker-compose.postgres.yml](docker-compose.postgres.yml).

**Important:** The Docker container requires the `NET_RAW` capability for MTR and privileged ping operations to work properly. This is already configured in the provided docker-compose files.

If running Docker manually, add the capability:

```bash
docker run --cap-add=NET_RAW -p 7575:7575 -v ./netronome:/data ghcr.io/autobrr/netronome:latest
```

**Note about MTR without NET_RAW**: When MTR runs without the NET_RAW capability (unprivileged mode), it falls back to UDP mode instead of ICMP. UDP mode may show higher packet loss than ICMP because:

- Some routers prioritize ICMP traffic over UDP
- Firewalls may rate-limit or drop UDP packets more aggressively
- UDP packets may be treated as lower priority during network congestion

For the most accurate packet loss measurements, ensure the container has NET_RAW capability or run netronome with sudo/root privileges.

## ⚙️ Configuration

### Configuration File (config.toml)

Netronome can be configured using a TOML file. Generate a default configuration:

```bash
netronome generate-config
```

This will create a `config.toml` file with default settings:

```toml
# Netronome Configuration

[database]
type = "sqlite"
path = "netronome.db"

[server]
host = "127.0.0.1"
port = 7575
base_url = ""

[logging]
level = "info"

[oidc]
issuer = ""
client_id = ""
client_secret = ""
redirect_url = ""

[speedtest]
timeout = 30

[speedtest.iperf]
test_duration = 10
parallel_conns = 4
timeout = 30

[speedtest.iperf.ping]
count = 5
interval = 1000
timeout = 10

[geoip]
country_database_path = ""
asn_database_path = ""

[notifications]
enabled = false
webhook_url = ""
ping_threshold = 30
upload_threshold = 200
download_threshold = 200

[tailscale]
enabled = false
hostname = ""
auth_key = ""
ephemeral = false
state_dir = ""
control_url = ""

[tailscale.agent]
port = 8200

[tailscale.monitor]
auto_discover = true
discovery_interval = "5m"
discovery_prefix = "netronome-agent-"
```

#### LibreSpeed Configuration

Netronome supports LibreSpeed servers for speed testing. A `librespeed-servers.json` file should be placed in the same directory as your configuration file. This file contains the LibreSpeed server definitions.

Example `librespeed-servers.json`:

```json
[
  {
    "id": 1,
    "name": "Clouvider - London, UK",
    "server": "http://lon.speedtest.clouvider.net/backend",
    "dlURL": "garbage.php",
    "ulURL": "empty.php",
    "pingURL": "empty.php",
    "getIpURL": "getIP.php"
  },
  {
    "id": 2,
    "name": "Your Custom Server",
    "server": "http://your-server.example.com/backend",
    "dlURL": "garbage.php",
    "ulURL": "empty.php",
    "pingURL": "empty.php",
    "getIpURL": "getIP.php"
  }
]
```

**Note:** When using Docker, the LibreSpeed CLI tool (`librespeed-cli`) is automatically included in the container.

### System Monitoring

Netronome includes a comprehensive distributed monitoring system using lightweight agents that can be deployed on remote servers to track bandwidth, hardware resources, and system information.

#### Agent Setup

The same `netronome` binary can run as a lightweight agent that can be deployed:

- **Remote servers**: Monitor system resources, bandwidth, and hardware across different servers/locations
- **Same server**: Useful when Netronome runs in Docker but you want to monitor the host system

##### Quick Installation (Recommended)

Use our interactive one-liner installation script for automatic setup:

```bash
curl -sL https://netrono.me/install-agent | bash
```

The script provides a **fully interactive installation** experience and will:

- Check for vnstat dependency
- **Interactively prompt** for network interface selection
- **Interactively choose** API key configuration (generate, custom, or none)
- **Interactively configure** listening address and port
- Create a systemd/launchd service
- Start the agent automatically
- **Interactively enable** automatic daily updates

##### Installation Options

```bash
# Interactive installation (prompts for all options) - RECOMMENDED
curl -sL https://netrono.me/install-agent | bash

# Enable auto-updates without prompting (non-interactive)
curl -sL https://netrono.me/install-agent | bash -s -- --auto-update true

# Disable auto-updates without prompting (non-interactive)
curl -sL https://netrono.me/install-agent | bash -s -- --auto-update false

# Update existing installation
curl -sL https://netrono.me/install-agent | bash -s -- --update

# Uninstall
curl -sL https://netrono.me/install-agent | bash -s -- --uninstall
```

##### Automatic Updates

When enabled, the agent will automatically check for updates daily at a random time (with up to 4 hours delay to prevent server overload). The update process:

1. Checks GitHub releases for newer versions
2. Downloads and verifies the new binary
3. Automatically restarts the service

You can check the update status with:

```bash
# Check update timer status
systemctl status netronome-agent-update.timer

# View last update attempt
journalctl -u netronome-agent-update

# Manually trigger update
/opt/netronome/netronome update
```

##### Manual Installation

1. **Deploy the Agent**

   ```bash
   # Run agent on default settings (0.0.0.0:8200)
   netronome agent

   # Run agent with API key authentication (recommended)
   netronome agent --api-key your-secret-key

   # Run agent on custom host and port
   netronome agent --host 192.168.1.100 --port 8300 --api-key your-secret-key

   # Run agent with specific interface
   netronome agent --interface eth0 --api-key your-secret-key

   # Include additional disk mounts (e.g., /mnt/storage)
   netronome agent --disk-include /mnt/storage --disk-include /mnt/backup

   # Exclude certain disk mounts from monitoring
   netronome agent --disk-exclude /boot --disk-exclude /tmp

   # Use host's existing tailscaled (if available)
   netronome agent --tailscale --tailscale-prefer-host

   # Create separate tsnet instance (default when --tailscale is used)
   netronome agent --tailscale --tailscale-auth-key tskey-auth-xxx

   # Run with config file
   netronome agent --config /path/to/config.toml
   ```

2. **Agent Configuration**

   Add to your `config.toml`:

   ```toml
   [agent]
   host = "0.0.0.0"  # IP address to bind to (0.0.0.0 for all interfaces)
   port = 8200
   interface = ""    # Empty for all interfaces, or specify like "eth0"
   api_key = ""      # API key for authentication (recommended)
   disk_includes = []  # Additional disk mounts to monitor, e.g., ["/mnt/storage", "/mnt/backup"]
   disk_excludes = []  # Disk mounts to exclude from monitoring, e.g., ["/boot", "/tmp"]

   [monitor]
   enabled = true
   ```

3. **Add Agents in Netronome UI**
   - Navigate to the "Monitoring" tab
   - Click "Add Agent"
   - Enter agent details:
     - Name: Friendly name for the server
     - URL: `http://server-ip:8200` (agent URL)
     - API Key: Enter the key configured on the agent (if authentication is enabled)
     - Enable monitoring: Start monitoring immediately
   - Once connected, view:
     - Real-time bandwidth usage
     - CPU, memory, and disk utilization
     - System information and uptime
     - Network interface details
     - Temperature sensors (if available)

#### Agent Systemd Service

Create `/etc/systemd/system/netronome-agent.service`:

```ini
[Unit]
Description=Netronome Monitor Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=netronome
Group=netronome
ExecStart=/usr/local/bin/netronome agent --config /etc/netronome/agent-config.toml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
systemctl enable netronome-agent
systemctl start netronome-agent
```

#### Security Considerations

- Agents expose bandwidth data via HTTP SSE endpoint
- No authentication on agent endpoint (rely on network security)
- Consider using reverse proxy with authentication if exposing to internet
- Agents are read-only and don't accept commands

#### Monitoring Capabilities

The monitoring agents provide comprehensive system visibility:

- **Bandwidth Monitoring**: Real-time network usage via vnstat integration
- **Hardware Statistics**: 
  - CPU usage percentage and load averages
  - Memory and swap usage
  - Disk usage across all mounted filesystems
  - Temperature sensors (where available)
- **System Information**:
  - Hostname and kernel version
  - System uptime
  - Network interface details and IP addresses
  - Peak bandwidth statistics

#### Disk Filtering

The agent supports flexible disk filtering to customize which filesystems are monitored:

- **Include Specific Mounts**: Use `disk_includes` to monitor additional mount points that might be excluded by default (e.g., `/mnt/storage`, `/mnt/nas`)
- **Exclude Mounts**: Use `disk_excludes` to hide specific mount points from monitoring (e.g., `/boot`, `/tmp`)
- **Glob Patterns**: Supports standard glob patterns for flexible matching (e.g., `/System/*`, `/mnt/disk*`)
- **Priority**: Explicitly included paths take precedence over exclusion rules
- **Default Behavior**: Special filesystems (`/snap/*`, `/run/*`, tmpfs, devfs, squashfs) are excluded unless explicitly included

Examples:
```bash
# Monitor /mnt/storage even if it's a special filesystem type
netronome agent --disk-include /mnt/storage

# Exclude all System volumes on macOS
netronome agent --disk-exclude "/System/*"

# Include all /mnt/disk* mounts
netronome agent --disk-include "/mnt/disk*"

# Hide /boot and all /tmp* directories from monitoring
netronome agent --disk-exclude /boot --disk-exclude "/tmp*"

# Using environment variables
export NETRONOME__AGENT_DISK_INCLUDES="/mnt/storage,/mnt/disk*"
export NETRONOME__AGENT_DISK_EXCLUDES="/boot,/tmp*,/System/*"
```

#### Data Accuracy and Unit Display

Netronome fetches bandwidth data directly from vnstat's native JSON output, ensuring exact data parity with other vnstat-based tools like swizzin panel.

**Unit Display**: Netronome uses proper binary units following IEC standards:

- **1 KiB = 1024 bytes** (binary kilobyte)
- **Displayed as**: KiB, MiB, GiB, TiB, PiB
- **Consistent with vnstat**: vnstat internally uses binary units

This ensures accurate and unambiguous representation of bandwidth data.

### Tailscale Integration

Netronome includes native Tailscale support, allowing agents to securely connect through Tailscale's mesh VPN without exposing ports or configuring NAT/firewall rules. This integration supports both isolated tsnet instances and using the host's existing tailscaled connection.

#### Key Features

- **Zero-configuration Networking**: Agents automatically connect through Tailscale's encrypted mesh network
- **Automatic Discovery**: Server automatically discovers ALL Netronome agents on port 8200 - no naming requirements!
- **Port-based Discovery**: Simply run agents on the default port (8200) and they're automatically found
- **Flexible Deployment**: Use isolated tsnet instances OR leverage existing host tailscaled
- **Visual Indicators**: UI shows shield icons and badges for Tailscale-connected agents
- **Mixed Mode Support**: Run both Tailscale and traditional HTTP agents simultaneously
- **Defense in Depth**: Combine Tailscale network isolation with API key authentication

#### Quick Start

##### For Agents

1. **Get a Tailscale Auth Key**
   - Visit https://login.tailscale.com/admin/settings/keys
   - Create a new auth key (reusable recommended for multiple agents)

2. **Start Agent with Tailscale**
   ```bash
   # Basic Tailscale agent (creates tsnet instance)
   ./bin/netronome agent --tailscale --tailscale-auth-key tskey-auth-YOUR-KEY

   # Use host's existing tailscaled (no new machine in Tailscale)
   ./bin/netronome agent --tailscale --tailscale-prefer-host

   # With custom hostname (any name works - no prefix required!)
   ./bin/netronome agent --tailscale --tailscale-hostname "webserver-prod" \
     --tailscale-auth-key tskey-auth-YOUR-KEY

   # Ephemeral agent (removes from Tailscale on shutdown)
   ./bin/netronome agent --tailscale --tailscale-ephemeral \
     --tailscale-auth-key tskey-auth-YOUR-KEY

   # With API key for additional security
   ./bin/netronome agent --tailscale --api-key "secret123" \
     --tailscale-auth-key tskey-auth-YOUR-KEY
   ```

3. **Using the Installation Script**
   ```bash
   # The install script supports Tailscale configuration
   curl -sL https://netrono.me/install-agent | bash
   # Follow prompts to enable Tailscale during setup
   ```

##### For Server

**Important Note**: The server's Tailscale configuration has two modes:

1. **Option A: Use Host's Tailscaled for Discovery** (recommended if server has tailscaled)
   
   If your server already has tailscaled running:
   ```toml
   # Don't enable [tailscale] section - use host's tailscaled!
   # [tailscale]
   # enabled = false  # Keep this false or omit entirely
   
   [tailscale.monitor]
   auto_discover = true      # Enable discovery using host's connection
   discovery_interval = "5m"
   discovery_port = 8200     # Discovers ALL agents on this port!
   ```

   The server will use the host's tailscaled to discover agents.

2. **Option B: Create Separate tsnet Instance** (if server doesn't have tailscaled)
   
   Add to your server's `config.toml`:
   ```toml
   [tailscale]
   enabled = true          # Creates a tsnet instance
   hostname = ""           # Optional custom hostname
   auth_key = ""           # Your Tailscale auth key
   prefer_host = false     # Set true to try host's tailscaled first

   [tailscale.monitor]
   auto_discover = true
   discovery_interval = "5m"
   discovery_port = 8200  # Discovers ALL agents on this port!
   ```

3. **Option C: Manual Agent Addition** (no auto-discovery)
   
   ```toml
   [tailscale.monitor]
   auto_discover = false  # Disable auto-discovery
   ```
   
   Then manually add agents through the UI.

4. **Start the Server**
   ```bash
   ./bin/netronome serve
   ```

#### Configuration Options

##### Agent Configuration

```toml
[tailscale]
enabled = false          # Enable Tailscale for this agent
hostname = ""           # Custom hostname (auto-generated if empty)
auth_key = ""          # Tailscale auth key (required for tsnet)
ephemeral = false      # Remove from Tailscale on shutdown
state_dir = ""         # Custom state directory (default: ~/.config/netronome/tailscale-agent)
control_url = ""       # Custom control server (for Headscale)
prefer_host = false    # Use host's tailscaled if available

[tailscale.agent]
port = 8200           # Port for agent to listen on Tailscale network
```

##### Server Configuration

```toml
[tailscale]
enabled = false         # Enable Tailscale support
hostname = ""          # Custom hostname for server
auth_key = ""         # Tailscale auth key (for tsnet)
ephemeral = false     # Remove from Tailscale on shutdown
state_dir = ""        # Custom state directory
control_url = ""      # Custom control server (for Headscale)
prefer_host = false   # Use host's tailscaled if available

[tailscale.monitor]
auto_discover = true          # Auto-discover Tailscale agents
discovery_interval = "5m"     # How often to check for new agents
discovery_port = 8200         # Port to check for Netronome agents
discovery_prefix = ""         # Optional: filter by hostname prefix
```

#### Environment Variables

All Tailscale settings can be configured via environment variables:

```bash
# Agent-specific
export NETRONOME__TAILSCALE_ENABLED=true
export NETRONOME__TAILSCALE_AUTH_KEY="tskey-auth-YOUR-KEY"
export NETRONOME__TAILSCALE_HOSTNAME="netronome-agent-web01"
export NETRONOME__TAILSCALE_EPHEMERAL=true
export NETRONOME__TAILSCALE_PREFER_HOST=true

# Server discovery
export NETRONOME__TAILSCALE_MONITOR_AUTO_DISCOVER=true
export NETRONOME__TAILSCALE_MONITOR_DISCOVERY_INTERVAL="2m"
```

#### How It Works

1. **Agent Side**:
   - **With tsnet** (default): Creates isolated instance in `~/.config/netronome/tailscale-agent`
   - **With host tailscaled** (`--tailscale-prefer-host`): Uses existing tailscaled daemon
   - Joins your Tailscale network with any hostname
   - Listens on the Tailscale network interface (100.x.x.x)
   - Regular HTTP/SSE endpoints work normally through Tailscale
   - Responds to `/netronome/info` for automatic identification

2. **Server Side**:
   - **With host tailscaled**: Discovery works using existing connection (no tsnet needed)
   - **With tsnet**: Creates separate instance for discovery if `[tailscale]` enabled
   - Periodically scans all peers and probes port 8200
   - Identifies Netronome agents via `/netronome/info` endpoint
   - Automatically creates monitor entries for discovered agents
   - Uses Tailscale hostnames for reliable connectivity

3. **Discovery Process**:
   ```
   Server → Scans all Tailscale peers
         → Probes each peer on port 8200
         → Checks /netronome/info endpoint
         → Auto-adds verified Netronome agents
   ```

4. **Network Flow**:
   ```
   Agent (tailscale) → Tailscale Network (encrypted) → Server
   ```

#### Important: Existing Tailscale Installations

**Q: What if my server/agent already has tailscaled running?**

**A: You have several options:**

- **For Agents**:
  1. **Use host's tailscaled** (recommended): `netronome agent --tailscale --tailscale-prefer-host`
     - No new machine in Tailscale admin
     - Uses existing connection
     - Automatically discovered on port 8200
  
  2. **Create separate tsnet**: `netronome agent --tailscale`
     - Creates new machine in Tailscale admin
     - Completely isolated from system tailscaled
  
  3. **Just use standard mode**: `netronome agent --host 0.0.0.0`
     - Access via host's Tailscale hostname
     - No --tailscale flag needed
     - Still auto-discovered if on port 8200

- **For Server**:
  1. **Use host's tailscaled for discovery**: Don't enable `[tailscale]`, only `[tailscale.monitor]`
  2. **Create tsnet for discovery**: Enable `[tailscale]` with auth key
  3. **Manual mode**: Disable auto-discovery, add agents manually

**Q: How do I avoid creating extra machines in Tailscale?**

**A: Use the `--tailscale-prefer-host` flag:**
```bash
# Agent will use host's tailscaled if available
netronome agent --tailscale --tailscale-prefer-host

# Server can discover without creating tsnet instance
# Just enable [tailscale.monitor] without [tailscale]
```

#### Visual Indicators

The UI provides clear indicators for Tailscale connectivity:

- 🛡️ **Shield Icon**: Agent is connected via Tailscale
- **Auto-discovered Badge**: Agent was found via Tailscale discovery
- **Connection Info**: Shows Tailscale hostname in agent details

#### Troubleshooting

**Agent not discovered?**
- Check agent is running on port 8200 (default discovery port)
- Verify both server and agent are on same Tailnet
- Check logs: `grep -i tailscale /path/to/netronome.log`
- Ensure discovery is enabled on server
- Test manually: `curl http://<agent-hostname>:8200/netronome/info`

**Connection issues?**
- Verify Tailscale connectivity: `tailscale ping <agent-hostname>`
- Check agent is listening: `curl http://<agent-hostname>:8200/health`
- Ensure no firewall blocking Tailscale (100.64.0.0/10)

**Want to use existing tailscaled instead?**
- For agents: Run without `--tailscale` flag, access via host's Tailscale hostname
- For server: Skip `[tailscale]` config, manually add agents
- Both approaches work perfectly!

#### Security Considerations

- **Network Isolation**: Tailscale provides encrypted point-to-point connections
- **No Port Exposure**: Agents don't need public ports or NAT configuration
- **Access Control**: Use Tailscale ACLs to restrict agent access
- **API Keys**: Combine with API authentication for defense in depth
- **Ephemeral Nodes**: Use `--tailscale-ephemeral` for temporary agents

#### Advanced Usage

##### Headscale Support

For self-hosted Tailscale control servers:

```bash
netronome agent --tailscale \
  --tailscale-control-url "https://headscale.example.com" \
  --tailscale-auth-key "YOUR-HEADSCALE-KEY"
```

##### Multiple Tailnets

Run agents on different Tailnets by using separate state directories:

```bash
# Production Tailnet
netronome agent --tailscale \
  --tailscale-state-dir ~/.config/netronome/tailscale-prod \
  --tailscale-auth-key "PROD-KEY"

# Development Tailnet  
netronome agent --tailscale \
  --tailscale-state-dir ~/.config/netronome/tailscale-dev \
  --tailscale-auth-key "DEV-KEY"
```

### Environment Variables

| Variable                                            | Description                                                       | Default                                      | Required               |
| --------------------------------------------------- | ----------------------------------------------------------------- | -------------------------------------------- | ---------------------- |
| `NETRONOME__HOST`                                   | Server host                                                       | `127.0.0.1`                                  | No                     |
| `NETRONOME__PORT`                                   | Server port                                                       | `7575`                                       | No                     |
| `NETRONOME__GIN_MODE`                               | Gin framework mode (`debug`/`release`)                            | `release`                                    | No                     |
| `NETRONOME__DB_TYPE`                                | Database type (`sqlite`/`postgres`)                               | `sqlite`                                     | No                     |
| `NETRONOME__DB_PATH`                                | SQLite database file path                                         | `./netronome.db`                             | Only for SQLite        |
| `NETRONOME__DB_HOST`                                | PostgreSQL host                                                   | `localhost`                                  | Only for PostgreSQL    |
| `NETRONOME__DB_PORT`                                | PostgreSQL port                                                   | `5432`                                       | Only for PostgreSQL    |
| `NETRONOME__DB_USER`                                | PostgreSQL user                                                   | `postgres`                                   | Only for PostgreSQL    |
| `NETRONOME__DB_PASSWORD`                            | PostgreSQL password                                               | -                                            | Only for PostgreSQL    |
| `NETRONOME__DB_NAME`                                | PostgreSQL database name                                          | `netronome`                                  | Only for PostgreSQL    |
| `NETRONOME__DB_SSLMODE`                             | PostgreSQL SSL mode                                               | `disable`                                    | Only for PostgreSQL    |
| `NETRONOME__IPERF_TEST_DURATION`                    | Duration of iPerf tests in seconds                                | `10`                                         | No                     |
| `NETRONOME__IPERF_PARALLEL_CONNS`                   | Number of parallel iPerf connections                              | `4`                                          | No                     |
| `NETRONOME__IPERF_TIMEOUT`                          | Timeout for iperf3 tests in seconds                               | `60`                                         | No                     |
| `NETRONOME__IPERF_PING_COUNT`                       | Number of ping packets to send for iperf3 tests                   | `5`                                          | No                     |
| `NETRONOME__IPERF_PING_INTERVAL`                    | Interval between ping packets in milliseconds for iperf3 tests    | `1000`                                       | No                     |
| `NETRONOME__IPERF_PING_TIMEOUT`                     | Timeout for ping tests in seconds for iperf3 tests                | `10`                                         | No                     |
| `NETRONOME__SPEEDTEST_TIMEOUT`                      | Speedtest timeout in seconds                                      | `30`                                         | No                     |
| `NETRONOME__LOG_LEVEL`                              | Log level (`trace`/`debug`/`info`/`warn`/`error`/`fatal`/`panic`) | `info`                                       | No                     |
| `NETRONOME__OIDC_ISSUER`                            | OpenID Connect issuer URL                                         | -                                            | Only for OIDC          |
| `NETRONOME__OIDC_CLIENT_ID`                         | OpenID Connect client ID                                          | -                                            | Only for OIDC          |
| `NETRONOME__OIDC_CLIENT_SECRET`                     | OpenID Connect client secret                                      | -                                            | Only for OIDC          |
| `NETRONOME__OIDC_REDIRECT_URL`                      | OpenID Connect redirect URL                                       | http://localhost:7575/api/auth/oidc/callback | Only for OIDC          |
| `NETRONOME__DEFAULT_PAGE`                           | Default page number for pagination                                | `1`                                          | No                     |
| `NETRONOME__DEFAULT_PAGE_SIZE`                      | Default page size for pagination                                  | `20`                                         | No                     |
| `NETRONOME__MAX_PAGE_SIZE`                          | Maximum page size for pagination                                  | `100`                                        | No                     |
| `NETRONOME__DEFAULT_TIME_RANGE`                     | Default time range for data queries                               | `1w`                                         | No                     |
| `NETRONOME__DEFAULT_LIMIT`                          | Default limit for data queries                                    | `20`                                         | No                     |
| `NETRONOME__SESSION_SECRET`                         | Session secret for authentication                                 | -                                            | No                     |
| `NETRONOME__NOTIFICATIONS_ENABLED`                  | Enable or disable notifications                                   | `false`                                      | No                     |
| `NETRONOME__NOTIFICATIONS_WEBHOOK_URL`              | Webhook URL for notifications                                     | -                                            | Only for Notifications |
| `NETRONOME__NOTIFICATIONS_PING_THRESHOLD`           | Ping threshold in ms for notifications                            | `30`                                         | No                     |
| `NETRONOME__NOTIFICATIONS_UPLOAD_THRESHOLD`         | Upload threshold in Mbps for notifications                        | `200`                                        | No                     |
| `NETRONOME__NOTIFICATIONS_DOWNLOAD_THRESHOLD`       | Download threshold in Mbps for notifications                      | `200`                                        | No                     |
| `NETRONOME__NOTIFICATIONS_DISCORD_MENTION_ID`       | Discord user/role ID to mention on alerts                         | -                                            | No                     |
| `NETRONOME__AUTH_WHITELIST`                         | Whitelist for authentication                                      | -                                            | No                     |
| `NETRONOME__GEOIP_COUNTRY_DATABASE_PATH`            | Path to GeoLite2-Country.mmdb file                                | -                                            | No                     |
| `NETRONOME__GEOIP_ASN_DATABASE_PATH`                | Path to GeoLite2-ASN.mmdb file                                    | -                                            | No                     |
| `NETRONOME__PACKETLOSS_ENABLED`                     | Enable packet loss monitoring feature                             | `true`                                       | No                     |
| `NETRONOME__PACKETLOSS_MAX_CONCURRENT_MONITORS`     | Maximum number of monitors that can run simultaneously            | `10`                                         | No                     |
| `NETRONOME__PACKETLOSS_PRIVILEGED_MODE`             | Use privileged ICMP mode for better accuracy                      | `false`                                      | No                     |
| `NETRONOME__PACKETLOSS_RESTORE_MONITORS_ON_STARTUP` | **WARNING**: Immediately run ALL enabled monitors on startup      | `false`                                      | No                     |
| `NETRONOME__AGENT_HOST`                             | Agent server bind address                                         | `0.0.0.0`                                    | No                     |
| `NETRONOME__AGENT_PORT`                             | Agent server port                                                 | `8200`                                       | No                     |
| `NETRONOME__AGENT_INTERFACE`                        | Network interface for agent to monitor (empty for all)            | ``                                           | No                     |
| `NETRONOME__AGENT_DISK_INCLUDES`                    | Comma-separated list of disk mounts to include                    | ``                                           | No                     |
| `NETRONOME__AGENT_DISK_EXCLUDES`                    | Comma-separated list of disk mounts to exclude                    | ``                                           | No                     |
| `NETRONOME__MONITOR_ENABLED`                        | Enable monitor client service in main server                      | `true`                                       | No                     |
| `NETRONOME__MONITOR_RECONNECT_INTERVAL`             | Reconnection interval for monitor agent connections               | `30s`                                        | No                     |
| `NETRONOME__TAILSCALE_ENABLED`                      | Enable Tailscale integration                                      | `false`                                      | No                     |
| `NETRONOME__TAILSCALE_HOSTNAME`                     | Custom Tailscale hostname (auto-generated if empty)               | ``                                           | No                     |
| `NETRONOME__TAILSCALE_AUTH_KEY`                     | Tailscale auth key for automatic registration                     | ``                                           | When Tailscale enabled |
| `NETRONOME__TAILSCALE_EPHEMERAL`                    | Remove from Tailscale network on shutdown                         | `false`                                      | No                     |
| `NETRONOME__TAILSCALE_STATE_DIR`                    | Custom Tailscale state directory                                  | `~/.config/netronome/tailscale-agent`        | No                     |
| `NETRONOME__TAILSCALE_CONTROL_URL`                  | Custom control server URL (for Headscale)                         | ``                                           | No                     |
| `NETRONOME__TAILSCALE_AGENT_PORT`                   | Port for agent to listen on Tailscale network                     | `8200`                                       | No                     |
| `NETRONOME__TAILSCALE_MONITOR_AUTO_DISCOVER`        | Enable automatic Tailscale agent discovery                        | `true`                                       | No                     |
| `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_INTERVAL`   | How often to scan for new Tailscale agents                        | `5m`                                         | No                     |
| `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_PORT`       | Port to probe for Netronome agents                                | `8200`                                       | No                     |
| `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_PREFIX`     | Optional hostname prefix filter                                   | ``                                           | No                     |
| `NETRONOME__TAILSCALE_PREFER_HOST`                  | Use host's tailscaled if available                               | `false`                                      | No                     |

### Database

Netronome supports two database backends:

1. **SQLite** (Default)
   - No additional setup required

2. **PostgreSQL**
   - Configure via:
     ```bash
     NETRONOME__DB_TYPE=postgres
     NETRONOME__DB_HOST=localhost
     NETRONOME__DB_PORT=5432
     NETRONOME__DB_USER=postgres
     NETRONOME__DB_PASSWORD=your-password
     NETRONOME__DB_NAME=netronome
     NETRONOME__DB_SSLMODE=disable
     ```

### Authentication

Netronome supports two authentication methods:

1. **Built-in Authentication**
   - Username/password authentication
   - Default option if no OIDC is configured

2. **OpenID Connect (OIDC)**
   - Integration with identity providers (Google, Okta, Auth0, Keycloak, Pocket-ID, Authelia, Authentik etc.)
   - PKCE support
   - Configure via environment variables:
     ```bash
     OIDC_ISSUER=https://pocketid.domain.net
     OIDC_CLIENT_ID=your-client-id
     OIDC_CLIENT_SECRET=your-client-secret
     OIDC_REDIRECT_URL=https://netronome.domain.net/api/auth/oidc/callback
     ```

3. **IP Whitelisting**
   - Bypass authentication for specific network ranges or IP addresses.
   - Configure in `config.toml` using CIDR notation:
     ```toml
     [auth]
     whitelist = ["127.0.0.1/32"]
     ```

### Scheduling Intervals

Netronome supports two types of scheduling intervals for both speed tests and packet loss monitors:

#### 1. **Duration-based Intervals**

- Format: Standard Go duration strings (e.g., `"30s"`, `"5m"`, `"1h"`, `"24h"`)
- Example: `"1m"` runs the test every 1 minute
- Next run calculation: Current time + interval duration + random jitter

#### 2. **Exact Time Intervals**

- Format: `"exact:HH:MM"` or `"exact:HH:MM,HH:MM"` for multiple times
- Example: `"exact:14:30"` runs at 2:30 PM daily
- Example: `"exact:00:00,12:00"` runs at midnight and noon daily
- Next run calculation: Next occurrence of specified time(s) + random jitter

#### Jitter (Randomization)

To prevent the "thundering herd" problem where all tests run simultaneously:

- **Duration-based**: Adds 1-300 seconds of random delay
- **Exact time**: Adds 1-60 seconds of random delay

This means if you have multiple monitors with the same interval, they won't all execute at exactly the same moment, preventing network congestion and ensuring more accurate results.

#### Startup and Missed Runs

When Netronome starts up, it recalculates the next run time for all scheduled tests:

- **Missed runs are NOT executed** - If Netronome was offline and missed scheduled runs, those tests won't be run retroactively
- **Next occurrence is calculated** - The scheduler finds the next valid time based on the interval
- **No catch-up behavior** - This prevents network flooding and ensures test results reflect current conditions

**Examples:**

- Exact time "exact:14:00" starting at 10:00 → Next run: 14:00 today
- Exact time "exact:14:00" starting at 15:00 → Next run: 14:00 tomorrow
- Duration "1h" starting anytime → Next run: current time + 1 hour + jitter

This design ensures predictable scheduling and prevents outdated tests from running after downtime.

### Packet Loss Monitoring

Netronome features a comprehensive packet loss monitoring system integrated into the unified traceroute interface:

#### Key Features

- **Unified Interface**: Switch between single traceroute tests and continuous monitoring in one tab
- **Flexible Scheduling**: Choose between interval-based (10s to 24h) or exact daily times
- **Auto-start Monitors**: Option to start monitoring immediately upon creation
- **Real-time Progress**: Live test progress with animated indicators during active testing
- **Multiple Test Types**: ICMP ping with automatic MTR fallback for detailed hop analysis
- **Historical Tracking**: Performance trend charts showing packet loss and RTT over time
- **Monitor Lifecycle**: Full start/stop/edit/delete capabilities with visual status indicators
- **Schedule Visualization**: Color-coded badges showing monitoring schedules and states
- **Cross-platform Support**: Works on Linux, macOS, and Windows with appropriate privileges

#### Monitor States

- **Active Monitoring**: Running on schedule with real-time status updates
- **Stopped**: Manually stopped, schedule visible but monitoring disabled
- **Testing**: Currently running a test with live progress indication

#### Important Notes on Monitor Startup Behavior

By default, packet loss monitors **DO NOT** start automatically when Netronome starts. This prevents:

- Network congestion from multiple simultaneous tests
- False packet loss readings due to concurrent ICMP traffic
- Unexpected network usage on startup

Monitors will run on their configured intervals after being manually started by users. If you want monitors to start immediately on program startup (not recommended), you can set:

```toml
[speedtest.packetloss]
restore_monitors_on_startup = true  # WARNING: This will immediately run ALL enabled monitors
```

**⚠️ Warning**: Setting `restore_monitors_on_startup = true` will cause ALL enabled monitors to run their tests immediately when Netronome starts, which can cause significant network traffic and potentially inaccurate results due to congestion.

### GeoIP Configuration

Netronome can display country flags and ASN information in traceroute results using MaxMind's GeoLite2 databases.

#### Setup Instructions

1. **Get a MaxMind License Key**
   - Sign up for a free account at [MaxMind](https://www.maxmind.com/en/geolite2/signup)
   - Generate a license key in your account dashboard

2. **Download the Databases**

   ```bash
   # Download GeoLite2-Country database (for country flags)
   curl -L "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=YOUR_LICENSE_KEY&suffix=tar.gz" -o GeoLite2-Country.tar.gz
   tar -xzf GeoLite2-Country.tar.gz
   cp GeoLite2-Country_*/GeoLite2-Country.mmdb /path/to/your/databases/

   # Download GeoLite2-ASN database (for ASN information)
   curl -L "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=YOUR_LICENSE_KEY&suffix=tar.gz" -o GeoLite2-ASN.tar.gz
   tar -xzf GeoLite2-ASN.tar.gz
   cp GeoLite2-ASN_*/GeoLite2-ASN.mmdb /path/to/your/databases/
   ```

3. **Configure Netronome**

   ```toml
   [geoip]
   country_database_path = "/path/to/your/databases/GeoLite2-Country.mmdb"
   asn_database_path = "/path/to/your/databases/GeoLite2-ASN.mmdb"
   ```

   Or using environment variables:

   ```bash
   NETRONOME__GEOIP_COUNTRY_DATABASE_PATH=/path/to/your/databases/GeoLite2-Country.mmdb
   NETRONOME__GEOIP_ASN_DATABASE_PATH=/path/to/your/databases/GeoLite2-ASN.mmdb
   ```

**Note:** Both databases are optional. You can configure only one if you only want country flags or ASN information. The databases should be updated monthly for best accuracy.

#### Understanding MTR Packet Loss Calculations

MTR displays two types of packet loss measurements that serve different purposes:

1. **Overall Packet Loss** - Measures end-to-end connectivity to the final destination
2. **Hop-by-Hop Loss** - Shows packet loss at each intermediate router along the path

**Why Overall Loss Can Be 0% Despite Intermediate Hop Timeouts:**

It's common and normal to see 100% packet loss at intermediate hops (showing as timeouts) while overall packet loss remains 0%. This occurs because:

- **Security Policies**: Many routers block or rate-limit ICMP responses for security reasons
- **Router Configuration**: Routers prioritize data forwarding over responding to diagnostic packets
- **Load Balancing**: Actual traffic may take different paths than probe packets
- **Network Design**: Intermediate infrastructure may be "invisible" to traceroute but still functional

**Practical Interpretation:**

- **0% overall loss** = Your connection to the destination is working perfectly
- **100% loss at intermediate hops** = Those routers don't respond to probes, but they're still forwarding your traffic

This is why MTR shows both metrics - overall connectivity health (most important for users) and detailed path analysis (useful for network troubleshooting). The overall packet loss percentage is the primary indicator of your actual network performance to the destination.

### Notifications

Netronome can send notifications to a webhook URL after each speed test. This is useful for integrating with services like Discord, Slack, or any other service that accepts webhooks.

To enable notifications, you need to set the following in your `config.toml` or as environment variables:

```toml
[notifications]
enabled = true
webhook_url = "your-webhook-url"
ping_threshold = 30
upload_threshold = 200
download_threshold = 200
discord_mention_id = ""
```

- `enabled` - Enable or disable notifications
- `webhook_url` - The webhook URL to send notifications to
- `ping_threshold` - The ping threshold in ms. If the ping is higher than this value, a notification will be sent.
- `upload_threshold` - The upload threshold in Mbps. If the upload speed is lower than this value, a notification will be sent.
- `download_threshold` - The download threshold in Mbps. If the download speed is lower than this value, a notification will be sent.
- `discord_mention_id` - Optional. A Discord user ID or role ID to mention when an alert is triggered. For example, `123456789012345678` for a user or `&123456789012345678` for a role.

### CLI Commands

Netronome provides several command-line commands:

- `generate-config` - Generate a default configuration file to `~/.config/netronome/config.toml`
- `serve` - Starts the Netronome server
- `agent` - Starts a monitor SSE agent for system and bandwidth monitoring
- `create-user` - Create a new user
- `change-password` - Change password for an existing user

Examples:

```bash
netronome generate-config

# Start monitor agent with custom settings
netronome agent --host 192.168.1.100 --port 8300 --interface eth0

# Create a new user (interactive)
netronome --config /home/username/.config/netronome/config.toml create-user username

# Create a new user (non-interactive)
echo "password123" | netronome --config /home/username/.config/netronome/config.toml create-user username

# Change user password (interactive)
netronome --config /home/username/.config/netronome/config.toml change-password username

# Change user password (non-interactive)
echo "newpassword123" | netronome --config /home/username/.config/netronome/config.toml change-password username
```

## ❓ FAQ

### Temperature Monitoring

**Q: What temperature sensors are supported?**

Netronome supports comprehensive temperature monitoring across multiple platforms:

- **Linux**: CPU cores/package, NVMe drives, SATA drives (via SMART), ACPI thermal zones
- **macOS**: CPU dies, thermal devices, calibration sensors, NVMe drives, NAND storage, battery
- **Windows**: CPU cores/package, NVMe drives, ACPI thermal zones

**Q: Why don't I see disk temperatures in the monitoring interface?**

A: Disk temperature monitoring has several requirements:

1. **Platform Support**: 
   - **Linux**: Full SMART support for SATA and NVMe drives
   - **macOS**: NVMe drives only (SATA drives not supported)
   - **Windows**: No SMART support (uses stub implementation)
   - **Other platforms**: No SMART support

2. **Build Requirements**: 
   - Release binaries are built without SMART support but still include:
     - CPU temperature monitoring (all platforms)
     - NVMe temperature monitoring (via gopsutil)
     - Battery temperature monitoring
   - SMART functionality (only for SATA/HDD temps and disk model names) requires:
     - The `github.com/anatol/smart.go` library
     - Building from source with CGO enabled
     - Only available on Linux and macOS
   - The `nosmart` build tag disables SMART support but keeps gopsutil temperatures

3. **Privileges Required**: The agent must run with elevated privileges (root/sudo) to access SMART data from disk devices. Without proper permissions, HDD temperatures cannot be read.

4. **SMART Support**: The drive must support SMART (Self-Monitoring, Analysis and Reporting Technology) and have it enabled. Most modern drives support this, but some older or specialized drives may not.

5. **Temperature Sensor**: Not all drives report temperature via SMART. Some drives, particularly older models, may not have temperature sensors.

**Q: Why do I see multiple temperature entries for NVMe drives?**

A: This is a limitation of the gopsutil library used for temperature monitoring. NVMe drives often have multiple temperature sensors (composite and individual sensors), but the library doesn't always provide enough context to distinguish which physical drive each sensor belongs to. You may see duplicate "nvme_composite" entries without clear identification of which drive they represent.

**Q: What do the different temperature sensor categories mean?**

Temperature sensors are organized into user-friendly categories:

- **CPU**: All processor-related sensors (cores, dies, thermal devices, packages)
- **Storage**: NVMe drives, NAND storage, SATA drives (where supported)
- **Power & Battery**: Battery temperature sensors (mainly on laptops)
- **System**: Calibration sensors, ACPI thermal zones, and other system sensors

**Q: How can I enable disk temperature monitoring?**

Run the agent with elevated privileges:

```bash
# Using sudo
sudo netronome agent --config /path/to/config.toml

# Or in the systemd service file, run as root
User=root
```

**Note**: Running with elevated privileges has security implications. Only do this in trusted environments.

## 🔨 Building from Source

### Building with Full SMART Support

The release binaries include most temperature monitoring (CPU, NVMe, battery) but lack SMART support for SATA/HDD temperatures and disk model names. To build with full SMART support on Linux or macOS:

```bash
# Clone the repository
git clone https://github.com/autobrr/netronome
cd netronome

# Build with SMART support (default when building locally)
make build

# The binary will be in ./bin/netronome with full SMART support
sudo ./bin/netronome agent  # Run with sudo for SATA/HDD temperature access
```

### Building without SMART Support

To build without SMART support (like the release binaries):

```bash
# Build with nosmart tag (still includes CPU/NVMe/battery temps via gopsutil)
CGO_ENABLED=0 go build -tags nosmart -o bin/netronome ./cmd/netronome
```

**What works without SMART:**
- ✅ CPU temperature monitoring (all cores, packages)
- ✅ NVMe temperature monitoring 
- ✅ Battery temperature monitoring
- ✅ Other system sensors via gopsutil

**What requires SMART (Linux/macOS only):**
- ❌ SATA/HDD temperature monitoring
- ❌ Disk model and serial number display

### Docker Build

The Docker images include SMART support by default:

```bash
make docker-build
make docker-run
```

**Note**: When running Docker containers, ensure you use `--cap-add=NET_RAW` and run with appropriate privileges for SMART disk access.

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.
