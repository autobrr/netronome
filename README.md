<h1 align="center">Netronome</h1>
<p align="center">
  <strong>Monitor. Analyze. Alert.</strong><br>
  A complete network performance monitoring solution with distributed agents, real-time metrics, and beautiful visualizations.
</p>
<div align="center">
<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-blue?logo=go" alt="Go version">
  <img src="https://img.shields.io/badge/build-passing-brightgreen" alt="Build Status">
  <img src="https://img.shields.io/github/v/release/autobrr/netronome" alt="Latest Release">
  </a>
    <a href="https://github.com/autobrr/netronome">
    <img src="https://img.shields.io/badge/%F0%9F%92%A1%20netronome-docs-00ACD7.svg?style=flat-square">
  </a>
</p>

[![Discord](https://img.shields.io/discord/881212911849209957.svg?label=&logo=discord&logoColor=ffffff&color=7389D8&labelColor=6A7EC2)](https://discord.gg/WehFCZxq5B)

</div>

<p align="center">
  <img src=".github/assets/netronome_dashboard.png" alt="Netronome Dashboard">
</p>

Netronome is a complete network performance monitoring solution that helps you understand and track your network's health. Whether you're monitoring your home internet connection, managing multi-site infrastructure, or tracking server performance, Netronome provides the insights you need through an intuitive web interface.

Built with Go and designed for simplicity, Netronome packages both frontend and backend into a single binary for effortless deployment. With a minimal footprint of just ~35MB RAM usage, it's perfect for resource-constrained environments. No complex setups - just download, configure, and run.

**Key capabilities:** Speed testing across multiple providers, continuous packet loss monitoring, distributed server monitoring via lightweight agents, and automated alerting - all with beautiful visualizations and historical tracking.

## Quick Start

Get Netronome running in under 5 minutes:

### Option 1: Download from Releases Page

Download prebuilt binaries from the [Releases page](https://github.com/autobrr/netronome/releases/latest).

### Option 2: One-liner Installation

```bash
# Download latest release
wget $(curl -s https://api.github.com/repos/autobrr/netronome/releases/latest | grep download | grep linux_x86_64 | cut -d\" -f4)
tar -C /usr/local/bin -xzf netronome*.tar.gz

# Generate default config
netronome generate-config

# Start the server
netronome serve
```

Open `http://localhost:7575` in your browser and create your account through the registration page. For Docker users, see the [Docker Installation](#docker-installation) section.

## Table of Contents

- [Features](#features)
- [External Dependencies](#external-dependencies)
- [Installation](#installation)
  - [Linux Generic](#linux-generic)
  - [Docker Installation](#docker-installation)
- [Basic Configuration](#basic-configuration)
  - [First Run Setup](#first-run-setup)
  - [Authentication](#authentication)
  - [Database](#database)
- [Advanced Configuration](#advanced-configuration)
  - [System Monitoring](#system-monitoring)
  - [Packet Loss Monitoring](#packet-loss-monitoring)
  - [Tailscale Integration](#tailscale-integration)
  - [GeoIP Configuration](#geoip-configuration)
  - [Notifications](#notifications)
  - [Scheduling](#scheduling)
- [Reference](#reference)
  - [Environment Variables](#environment-variables)
  - [CLI Commands](#cli-commands)
- [FAQ & Troubleshooting](#faq--troubleshooting)
- [Building from Source](#building-from-source)
- [Contributing](#contributing)
- [License](#license)

## Features

### Core Functionality

- **Speed Testing**: Multiple providers (Speedtest.net, iperf3, LibreSpeed) with real-time progress and historical tracking
- **Network Diagnostics**: Traceroute and continuous packet loss monitoring with MTR integration
- **System Monitoring**: Deploy agents for distributed server monitoring with real-time metrics
- **Flexible Scheduling**: Automated tests with customizable intervals and smart jitter prevention

### Network Diagnostics

<p align="center">
  <img src=".github/assets/packetloss-monitors(mtr).png" alt="Network Diagnostics Interface">
</p>

Advanced network path analysis with:

- Cross-platform traceroute support
- Continuous ICMP monitoring
- Per-hop packet loss statistics
- GeoIP visualization with country flags
- Historical performance tracking

### System Monitoring

<p align="center">
  <img src=".github/assets/agents-dashboard.png" alt="System Monitoring Dashboard">
</p>

<p align="center">
  <img src=".github/assets/agents-bandwidth.png" alt="Bandwidth Monitoring">
</p>

<p align="center">
  <img src=".github/assets/agents-systeminfo.png" alt="Agent System Information">
</p>

#### Binary Types

Netronome provides two binary options for different deployment scenarios:

- **`netronome`** - Full server binary (~66MB) with complete web interface, database, and agent functionality
- **`netronome-agent`** - Lightweight agent-only binary (~32MB) optimized for distributed monitoring

**When to use each:**
- **Full Binary (`netronome`)**: Main server deployment, development, or when you need all commands
- **Agent Binary (`netronome-agent`)**: Remote monitoring agents, resource-constrained deployments, or containerized agents

Both binaries are included in release archives. The installation script automatically detects and prefers the lightweight agent binary when available.

#### Agent Setup

Netronome agents can be deployed using either binary depending on your needs:

Monitor multiple servers from one dashboard:

- CPU, memory, disk, and temperature metrics
- Real-time bandwidth monitoring (vnstat)
- Auto-discovery for Tailscale networks
- Configurable alerting thresholds
- Live data streaming via SSE
- Agents are also single binaries - same simple deployment

### Additional Features

- **Modern UI**: Responsive design with dark mode support
- **Authentication**: Built-in auth, OIDC support, IP whitelisting
- **Notifications**: 15+ services via Shoutrrr (Discord, Telegram, Email, etc.)
- **Database Support**: SQLite (default) or PostgreSQL
- **Tailscale Integration**: Secure mesh networking without port exposure

### Technical Overview

- **Single Binary**: Frontend and backend compiled into one executable (~66MB)
- **Language**: Written in Go for performance and easy deployment
- **Frontend**: React with TypeScript, embedded in the binary
- **Database**: SQLite by default, PostgreSQL optional
- **No Runtime Dependencies**: Just the binary and optional external tools

## Prerequisites

### System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Architecture**: x86_64, ARM64
- **Memory**: ~35MB (typical usage)
- **Disk Space**: 65MB for application + database growth

### External Dependencies

The following tools enable specific features (automatically included in Docker):

- **iperf3** - For iperf3 speed testing
- **librespeed-cli** - For LibreSpeed testing
- **traceroute** - For basic network path discovery (usually pre-installed)
- **mtr** - For advanced packet loss analysis per hop (optional, falls back to traceroute)
- **vnstat** - For bandwidth monitoring on agents (optional but recommended)

Install on Linux:

```bash
# Debian/Ubuntu
sudo apt-get install iperf3 traceroute mtr vnstat

# RHEL/Fedora
sudo dnf install iperf3 traceroute mtr vnstat
```

Notes:

- Speedtest.net is built-in
- All external dependencies are optional - Netronome gracefully handles missing tools

## Installation

### Linux Generic

1. **Download and Install**

   ```bash
   # Download latest release
   wget $(curl -s https://api.github.com/repos/autobrr/netronome/releases/latest | grep download | grep linux_x86_64 | cut -d\" -f4)
   tar -C /usr/local/bin -xzf netronome*.tar.gz
   ```

2. **Create Systemd Service** (Recommended)

   ```bash
   sudo tee /etc/systemd/system/netronome@.service > /dev/null <<EOF
   [Unit]
   Description=netronome service for %i
   After=syslog.target network-online.target

   [Service]
   Type=simple
   User=%i
   Group=%i
   ExecStart=/usr/local/bin/netronome serve --config=/home/%i/.config/netronome/config.toml

   [Install]
   WantedBy=multi-user.target
   EOF
   ```

3. **Enable and Start**
   ```bash
   systemctl enable --now netronome@$USER
   ```

### Docker Installation

Quick Docker deployment with automatic dependency installation:

```bash
# Clone the repository (for docker-compose files)
git clone https://github.com/autobrr/netronome.git
cd netronome

# Basic setup with SQLite
docker-compose up -d

# Or with PostgreSQL for better performance
docker-compose -f docker-compose.postgres.yml up -d
```

The Docker image includes all dependencies (iperf3, librespeed-cli, traceroute, mtr, vnstat) pre-installed, so you don't need to install them separately. For Tailscale integration with Docker, see the [Docker Tailscale Sidecar Guide](docs/docker-tailscale-sidecar.md).

## Basic Configuration

### First Run Setup

1. **Generate Configuration**

   ```bash
   netronome generate-config
   ```

   This creates `~/.config/netronome/config.toml` with default settings.

2. **Start Server**

   ```bash
   netronome serve
   ```

3. **Access Interface**
   Navigate to `http://localhost:7575` and register your account through the web interface.

   To access from other devices on your network, change the host in config.toml from `127.0.0.1` to `0.0.0.0`.

### Authentication

Netronome supports multiple authentication methods:

#### Built-in Authentication

Users can register directly through the web interface on first visit. For automation or admin purposes, you can also manage users via CLI:

```bash
netronome create-user <username>     # Create user via CLI
netronome change-password <username>  # Change password via CLI
```

#### OpenID Connect (OIDC)

Configure via environment variables:

```bash
export NETRONOME__OIDC_ISSUER=https://your-provider.com
export NETRONOME__OIDC_CLIENT_ID=your-client-id
export NETRONOME__OIDC_CLIENT_SECRET=your-client-secret
export NETRONOME__OIDC_REDIRECT_URL=https://netronome.example.com/api/auth/oidc/callback
```

#### IP Whitelisting

Add to `config.toml`:

```toml
[auth]
whitelist = ["127.0.0.1/32", "192.168.1.0/24"]
```

### Database

#### SQLite (Default)

No additional setup required. Database file is created automatically.

#### PostgreSQL

Configure via environment variables:

```bash
export NETRONOME__DB_TYPE=postgres
export NETRONOME__DB_HOST=localhost
export NETRONOME__DB_PORT=5432
export NETRONOME__DB_USER=postgres
export NETRONOME__DB_PASSWORD=your-password
export NETRONOME__DB_NAME=netronome
```

### Reverse Proxy with Base URL

To serve Netronome under a subpath (e.g., `/netronome`) behind nginx:

#### 1. Configure Netronome

Set the base URL in your `config.toml`:

```toml
[server]
host = "127.0.0.1"  # Listen only on localhost since nginx will proxy
port = 7575
base_url = "/netronome"  # The subpath you want to use
```

#### 2. Configure nginx

Add this location block to your nginx configuration:

```nginx
# Redirect /netronome to /netronome/
location = /netronome {
    return 301 /netronome/;
}

location /netronome/ {
    proxy_pass http://127.0.0.1:7575;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 86400;
}
```

That's it! The minimal configuration above handles WebSocket/SSE for real-time features.

## Common Use Cases

### Home Network Monitoring

Monitor your internet connection quality:

1. Schedule hourly speed tests to your ISP
2. Set up packet loss monitoring to `8.8.8.8` or `1.1.1.1`
3. Configure notifications when speeds drop below expected thresholds

### Multi-Site Infrastructure

Monitor connectivity between office locations:

1. Deploy agents at each site
2. Configure iperf3 tests between locations
3. Use Tailscale for secure agent communication
4. Set up alerts for degraded inter-site connectivity

### Server Health Monitoring

Track server performance metrics:

1. Install agents on production servers
2. Monitor CPU, memory, disk usage, and temperatures
3. Configure threshold alerts for resource exhaustion
4. Track bandwidth usage patterns

## Advanced Configuration

### System Monitoring

Deploy monitoring agents on remote servers for full system visibility.

#### Quick Agent Installation

```bash
curl -sL https://netrono.me/install-agent | bash
```

The script provides interactive setup for:

- Network interface selection
- API key configuration
- Listening address and port
- Systemd service creation
- Automatic updates

#### Manual Agent Setup

```bash
# Basic agent
netronome agent

# With authentication
netronome agent --api-key your-secret-key

# Custom configuration
netronome agent --host 192.168.1.100 --port 8300 --interface eth0
```

#### Agent Configuration

Add to `config.toml`:

```toml
[agent]
host = "0.0.0.0"
port = 8200
interface = ""  # Empty for all interfaces
api_key = "your-secret-key"
disk_includes = ["/mnt/storage"]  # Additional mounts to monitor
disk_excludes = ["/boot", "/tmp"] # Mounts to exclude

[monitor]
enabled = true
```

### Packet Loss Monitoring

Continuous network monitoring with MTR integration and performance tracking.

#### Key Features

- Flexible scheduling (10 seconds to 24 hours or exact daily times)
- Real-time progress indicators
- Historical performance charts
- Cross-platform support with privilege fallback

#### Important Notes

- MTR requires elevated privileges for full functionality
- Overall packet loss can be 0% even with intermediate hop timeouts (normal behavior)

### Tailscale Integration

Native Tailscale support for secure mesh networking without port exposure.

#### Agent Setup

```bash
# Basic Tailscale agent
netronome agent --tailscale --tailscale-auth-key tskey-auth-YOUR-KEY

# Use existing tailscaled
netronome agent --tailscale --tailscale-method host

# Custom hostname
netronome agent --tailscale --tailscale-hostname "webserver-prod"
```

#### Server Configuration

```toml
[tailscale]
enabled = true
method = "auto"  # auto, host, or tsnet
auth_key = ""    # Required for tsnet mode
hostname = ""    # Optional custom hostname

# Discovery settings
auto_discover = true
discovery_interval = "5m"
discovery_port = 8200
```

### GeoIP Configuration

Enable country flags and ASN information in traceroute results (optional):

1. Sign up for a free license at [MaxMind](https://www.maxmind.com/en/geolite2/signup)
2. Download the GeoLite2 databases (Country and ASN)
3. Add paths to your config:
   ```toml
   [geoip]
   country_database_path = "/path/to/GeoLite2-Country.mmdb"
   asn_database_path = "/path/to/GeoLite2-ASN.mmdb"
   ```

Netronome works perfectly without GeoIP - this just adds visual country indicators.

### Notifications

<p align="center">
  <img src=".github/assets/notifications.png" alt="Notifications Configuration">
</p>

Configure notifications through the web interface at **Settings > Notifications**.

#### Supported Services

- Discord, Telegram, Slack, Teams
- Email (SMTP), Pushover, Pushbullet
- Gotify, Matrix, Ntfy, Webhook
- [And 15+ more via Shoutrrr](https://containrrr.dev/shoutrrr/)

#### Notification Events

- Speed test completion, failures, threshold breaches
- Packet loss state changes (degraded/recovered)
- Agent metrics: CPU, memory, disk, bandwidth, temperature thresholds

### Scheduling

Two scheduling types supported:

#### Duration-based Intervals

```
"30s", "5m", "1h", "24h"
```

Adds 1-300 seconds of random jitter to prevent simultaneous execution.

#### Exact Time Intervals

```
"exact:14:30"           # Daily at 2:30 PM
"exact:00:00,12:00"     # Midnight and noon daily
```

Adds 1-60 seconds of random jitter.

## Reference

### Environment Variables

All configuration options can be set via environment variables using the `NETRONOME__` prefix. Here are the most commonly used:

```bash
# Server settings
NETRONOME__HOST=0.0.0.0              # Listen address
NETRONOME__PORT=7575                 # Web UI port
NETRONOME__BASE_URL=/                # Base URL for reverse proxy

# Database (SQLite by default)
NETRONOME__DB_TYPE=sqlite            # sqlite or postgres
NETRONOME__DB_PATH=netronome.db      # SQLite database path

# PostgreSQL (when DB_TYPE=postgres)
NETRONOME__DB_HOST=localhost
NETRONOME__DB_PORT=5432
NETRONOME__DB_USER=postgres
NETRONOME__DB_PASSWORD=secret
NETRONOME__DB_NAME=netronome
NETRONOME__DB_SSLMODE=disable

# Authentication
NETRONOME__AUTH_WHITELIST=127.0.0.1/32,192.168.1.0/24  # IP whitelist (comma-separated)
NETRONOME__SESSION_SECRET=           # Session secret (auto-generated if empty)

# OIDC (optional)
NETRONOME__OIDC_ISSUER=https://accounts.google.com
NETRONOME__OIDC_CLIENT_ID=your-client-id
NETRONOME__OIDC_CLIENT_SECRET=your-secret
NETRONOME__OIDC_REDIRECT_URL=https://example.com/api/auth/oidc/callback
```

<details>
<summary><b>Complete Environment Variables Reference</b> (click to expand)</summary>

### Server Configuration

```bash
NETRONOME__HOST=127.0.0.1                    # Server listen address
NETRONOME__PORT=7575                         # Server port
NETRONOME__BASE_URL=/                        # Base URL path (for reverse proxy)
NETRONOME__GIN_MODE=                         # Gin framework mode (debug/release/test)
```

### Database Configuration

```bash
NETRONOME__DB_TYPE=sqlite                    # Database type: sqlite or postgres
NETRONOME__DB_PATH=netronome.db              # SQLite database file path
NETRONOME__DB_HOST=localhost                 # PostgreSQL host
NETRONOME__DB_PORT=5432                      # PostgreSQL port
NETRONOME__DB_USER=postgres                  # PostgreSQL user
NETRONOME__DB_PASSWORD=                      # PostgreSQL password
NETRONOME__DB_NAME=netronome                 # PostgreSQL database name
NETRONOME__DB_SSLMODE=disable                # PostgreSQL SSL mode
```

### Logging

```bash
NETRONOME__LOG_LEVEL=info                    # Log level: trace, debug, info, warn, error, fatal, panic
```

### Authentication

```bash
NETRONOME__AUTH_WHITELIST=                   # Comma-separated CIDR networks to bypass auth
NETRONOME__SESSION_SECRET=                   # Session encryption secret (auto-generated if empty)
```

### OIDC Configuration

```bash
NETRONOME__OIDC_ISSUER=                      # OIDC provider URL
NETRONOME__OIDC_CLIENT_ID=                   # OIDC client ID
NETRONOME__OIDC_CLIENT_SECRET=               # OIDC client secret
NETRONOME__OIDC_REDIRECT_URL=                # OIDC callback URL
```

### Speed Test Configuration

```bash
NETRONOME__SPEEDTEST_TIMEOUT=30              # Overall speedtest timeout (seconds)

# iperf3 settings
NETRONOME__IPERF_TEST_DURATION=10            # Test duration (seconds)
NETRONOME__IPERF_PARALLEL_CONNS=4            # Parallel connections
NETRONOME__IPERF_TIMEOUT=60                  # iperf3 timeout (seconds)
NETRONOME__IPERF_PING_COUNT=5                # Ping count for latency test
NETRONOME__IPERF_PING_INTERVAL=1000          # Ping interval (milliseconds)
NETRONOME__IPERF_PING_TIMEOUT=10             # Ping timeout (seconds)

# LibreSpeed settings
NETRONOME__LIBRESPEED_TIMEOUT=60             # LibreSpeed timeout (seconds)
```

### Pagination

```bash
NETRONOME__DEFAULT_PAGE=1                    # Default page number
NETRONOME__DEFAULT_PAGE_SIZE=20              # Default items per page
NETRONOME__MAX_PAGE_SIZE=100                 # Maximum items per page
NETRONOME__DEFAULT_TIME_RANGE=1w             # Default time range for queries
NETRONOME__DEFAULT_LIMIT=20                  # Default query limit
```

### GeoIP Configuration

```bash
NETRONOME__GEOIP_COUNTRY_DATABASE_PATH=      # Path to GeoLite2-Country.mmdb
NETRONOME__GEOIP_ASN_DATABASE_PATH=          # Path to GeoLite2-ASN.mmdb
```

### Packet Loss Monitoring

```bash
NETRONOME__PACKETLOSS_ENABLED=true                      # Enable packet loss monitoring
NETRONOME__PACKETLOSS_DEFAULT_INTERVAL=3600             # Default test interval (seconds)
NETRONOME__PACKETLOSS_DEFAULT_PACKET_COUNT=10           # Packets per test
NETRONOME__PACKETLOSS_MAX_CONCURRENT_MONITORS=10        # Max concurrent monitors
NETRONOME__PACKETLOSS_PRIVILEGED_MODE=true              # Use privileged ICMP mode
NETRONOME__PACKETLOSS_RESTORE_MONITORS_ON_STARTUP=false # Restore monitors on startup
```

### Agent Configuration

```bash
NETRONOME__AGENT_HOST=0.0.0.0                # Agent listen address
NETRONOME__AGENT_PORT=8200                   # Agent port
NETRONOME__AGENT_INTERFACE=                  # Network interface to monitor (empty for all)
NETRONOME__AGENT_API_KEY=                    # Agent API key for authentication
NETRONOME__AGENT_DISK_INCLUDES=              # Comma-separated paths to include
NETRONOME__AGENT_DISK_EXCLUDES=              # Comma-separated paths to exclude
```

### Monitor Configuration

```bash
NETRONOME__MONITOR_ENABLED=true              # Enable system monitoring
NETRONOME__MONITOR_RECONNECT_INTERVAL=30s    # Agent reconnection interval
```

### Tailscale Configuration

```bash
# Core settings
NETRONOME__TAILSCALE_ENABLED=false           # Enable Tailscale integration
NETRONOME__TAILSCALE_METHOD=auto             # Method: auto, host, or tsnet
NETRONOME__TAILSCALE_AUTH_KEY=               # Auth key (required for tsnet)

# TSNet settings
NETRONOME__TAILSCALE_HOSTNAME=               # Custom hostname (optional)
NETRONOME__TAILSCALE_EPHEMERAL=false         # Remove on shutdown
NETRONOME__TAILSCALE_STATE_DIR=~/.config/netronome/tsnet  # State directory
NETRONOME__TAILSCALE_CONTROL_URL=            # For Headscale (optional)

# Agent settings
NETRONOME__TAILSCALE_AGENT_PORT=8200         # Port for agent to listen on

# Discovery settings
NETRONOME__TAILSCALE_AUTO_DISCOVER=true      # Auto-discover Tailscale agents
NETRONOME__TAILSCALE_DISCOVERY_INTERVAL=5m   # Discovery check interval
NETRONOME__TAILSCALE_DISCOVERY_PORT=8200     # Port to probe for agents
NETRONOME__TAILSCALE_DISCOVERY_PREFIX=       # Hostname prefix filter

# Deprecated (for backward compatibility)
NETRONOME__TAILSCALE_PREFER_HOST=false       # Prefer host mode over tsnet
NETRONOME__TAILSCALE_AGENT_ENABLED=false     # Enable agent mode
NETRONOME__TAILSCALE_AGENT_ACCEPT_ROUTES=true # Accept Tailscale routes
```

</details>

### CLI Commands

```bash
# Server management
netronome serve                    # Start the server
netronome generate-config          # Generate default config

# User management
netronome create-user <username>   # Create new user
netronome change-password <username> # Change user password

# Agent mode
netronome agent                    # Start monitoring agent
netronome agent --api-key secret   # Agent with authentication
```

## FAQ & Troubleshooting

### Getting Started

**Q: What's the default username/password?**

For manual installation, you register through the web interface on first visit - no default credentials.

**Q: Can I access Netronome from other devices on my network?**

Yes! Change the host in your config from `127.0.0.1` to `0.0.0.0` or set `NETRONOME__HOST=0.0.0.0`.

**Q: Do I need all the external dependencies?**

No, Netronome works without them. You only need:

- `iperf3` for iperf3 speed tests
- `librespeed-cli` for LibreSpeed tests
- Speedtest.net works out of the box

### Common Issues

**Q: "Port 7575 already in use" error?**

Another service is using the port. Either stop that service or change Netronome's port in the config.

**Q: Speed tests are slower than expected?**

1. Ensure you're testing to a nearby server
2. Check if other devices are using bandwidth
3. Try different test providers (some ISPs throttle specific tests)
4. For iperf3, ensure the server supports your connection speed

### Temperature Monitoring

**Q: Why don't I see disk temperatures?**

Disk temperature monitoring requires:

1. Platform support (Linux/macOS for SATA, all platforms for NVMe)
2. Elevated privileges (root/sudo for SMART access)
3. SMART-enabled drives with temperature sensors

### Network Diagnostics

**Q: Why do I see 0% packet loss but 100% loss at intermediate hops?**

This is normal behavior. Intermediate routers often block/rate-limit ICMP responses for security while still forwarding traffic. Overall packet loss measures end-to-end connectivity, which is the primary indicator of network performance.

### Agent Discovery

**Q: Tailscale agent not discovered?**

- Ensure agent runs on discovery port (default 8200)
- Verify both server and agent are on same Tailnet
- Check connectivity: `tailscale ping <agent-hostname>`
- Test endpoint: `curl http://<agent-hostname>:8200/netronome/info`

## Building from Source

Building Netronome creates a single binary with the frontend embedded:

### With Full SMART Support (Linux/macOS)

```bash
# Clone the repository
git clone https://github.com/autobrr/netronome
cd netronome

# Build full server binary with web interface (requires frontend build)
make build

# Build lightweight agent-only binary (no frontend required)
make build-agent  

# Build both binaries
make build-all
```

**Build target details:**
- **`make build`** - Full server binary (~66MB) with embedded web UI, database, all commands
- **`make build-agent`** - Agent-only binary (~32MB) with just monitoring functionality
- **`make build-all`** - Both binaries for comprehensive deployment options

### Building with Full SMART Support

The release binaries include most temperature monitoring (CPU, NVMe, battery) but lack SMART support for SATA/HDD temperatures and disk model names. To build with full SMART support on Linux or macOS:

```bash
# Build full server with SMART support (requires CGO, default when building locally)
CGO_ENABLED=1 make build

# Build agent with SMART support (requires CGO)
CGO_ENABLED=1 make build-agent

# The binaries will be in ./bin/ with full SMART support
sudo ./bin/netronome agent  # Run with sudo for SATA/HDD temperature access
sudo ./bin/netronome-agent  # Or use the dedicated agent binary
```

**Note**: SMART support requires `CGO_ENABLED=1` to interface with system disk libraries.

### Building without SMART Support

To build without SMART support (like the release binaries):

```bash
# First build frontend
cd web && pnpm install && pnpm build && cd ..
# Then build Go binary
CGO_ENABLED=0 go build -tags nosmart -o bin/netronome ./cmd/netronome

# Agent binary without SMART  
CGO_ENABLED=0 go build -tags nosmart -o bin/netronome-agent ./cmd/netronome-agent
```

### Docker Build

```bash
make docker-build
make docker-run
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.
