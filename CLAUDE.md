# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Build and Development

```bash
# Build entire application (frontend + backend)
make build

# Development mode with live reload (requires tmux)
make dev

# Backend live reload only (requires air)
make watch

# Clean build artifacts
make clean

# Run built application
make run

# Docker commands
make docker-build
make docker-run
```

### Configuration Management

```bash
# Generate default config (creates ~/.config/netronome/config.toml)
./bin/netronome generate-config

# Run with specific config
./bin/netronome serve --config config.toml

# Create user (interactive)
./bin/netronome create-user username

# Change password (interactive)
./bin/netronome change-password username
```

### Frontend Development

```bash
cd web
pnpm install     # Install dependencies
pnpm dev         # Start dev server (port 5173)
pnpm build       # Build for production
pnpm lint        # Run ESLint
```

### Code Formatting and Quality

```bash
# License header management for all code files
./license.sh false # Add headers without interactive prompts

# Backend live reload during development (requires air)
make watch

# Test commands
go test ./...           # Run all Go tests
go test ./internal/...  # Run internal package tests
go test -v ./internal/speedtest -run TestPing  # Run specific test with verbose output
cd web && pnpm lint     # Frontend linting
```

## Architecture Overview

### Core Application Structure

- **Entry Point**: `cmd/netronome/main.go` using Cobra CLI with commands: `serve`, `generate-config`, `create-user`, `change-password`, `agent`
- **Configuration**: TOML-based config with environment variable overrides (`NETRONOME__*` prefix)
- **Database**: Supports SQLite (default) and PostgreSQL with embedded migrations
- **Frontend**: React 18 + TypeScript with embedded serving via Go's `embed.FS`

### Service Layer Organization

```
internal/
├── config/          # TOML + env var configuration system
├── database/        # Interface-based DB layer with Squirrel query builder
├── server/          # Gin HTTP server with middleware stack
├── speedtest/       # Core testing logic (6 implementations)
├── scheduler/       # Background cron-like job scheduler
├── auth/           # Session + OIDC authentication
├── notifications/  # Webhook/Discord alert system
├── agent/          # vnstat SSE agent server implementation
├── vnstat/         # vnstat SSE client and database service
└── types/          # Shared data structures
```

### Speed Test Implementations

**1. Speedtest.net** (`internal/speedtest/speedtest.go`)

- Uses `github.com/showwin/speedtest-go` library
- Auto-discovers servers with distance-based sorting
- 30-minute server cache
- Real-time progress callbacks

**2. iperf3** (`internal/speedtest/iperf.go`)

- Requires `iperf3` binary installation
- JSON output parsing for real-time progress
- Combined with ping tests (`internal/speedtest/ping.go`) for latency and packet loss metrics
- Jitter measurement is not supported for iperf3 (use Speedtest.net for jitter measurements)
- Cross-platform ping implementation (Linux/macOS/Windows)

**3. LibreSpeed** (`internal/speedtest/librespeed.go`)

- Requires `librespeed-cli` binary
- JSON server configuration (`librespeed-servers.json`)
- Single command execution with result parsing

**4. Traceroute** (`internal/speedtest/traceroute.go`)

- Cross-platform implementation (Linux/macOS/Windows)
- Uses system `traceroute`/`tracert` commands
- Real-time streaming progress updates during execution
- GeoIP integration for country flags and ASN information
- Smart early termination on consecutive timeouts or destination reached
- Supports URL hostname extraction and port stripping

**5. Packet Loss Monitoring** (`internal/speedtest/packetloss.go`)

- Uses `github.com/prometheus-community/pro-bing` library for ICMP ping tests
- Continuous monitoring with configurable intervals (10s to hours/days)
- Configurable packet count (1-100 packets per test)
- Real-time progress tracking using OnSend callbacks
- Robust timeout handling with tri-state logic (testing/monitoring/disabled)
- Supports both privileged (raw sockets) and unprivileged (UDP) modes
- Database-backed monitor configurations and historical results
- MTR integration (requires root/privileged mode) with automatic fallback to standard ICMP ping

**6. MTR (My TraceRoute)** (`internal/speedtest/mtr.go`)

- Advanced network diagnostic tool combining ping and traceroute
- Per-hop packet loss and latency statistics
- Automatic fallback to standard ping when MTR binary is unavailable or lacks root privileges
- MTR requires privileged mode (root/NET_RAW capability) for ICMP; falls back to UDP or ping otherwise
- Real-time progress updates during execution

**7. vnstat Bandwidth Monitoring** (`internal/agent/agent.go`, `internal/vnstat/client.go`)

- **Distributed Architecture**: Lightweight agents broadcast vnstat data via Server-Sent Events (SSE)
- **Agent Implementation**: Uses Gin framework to serve SSE endpoint at `/events?stream=live-data` and historical data endpoint at `/export/historical`
- **Client Implementation**: SSE client with automatic reconnection and exponential backoff
- **Real-time Streaming**: Processes `vnstat --live --json` output and broadcasts to connected clients
- **Multi-agent Support**: Central server can connect to multiple remote agents
- **Native Data Usage**: Fetches and displays vnstat's native JSON calculations directly
- **URL Normalization**: Automatically formats agent URLs to ensure correct SSE endpoint
- **Error Recovery**: Graceful handling of agent disconnections with automatic retry

### Database Schema and Migrations

- **Migration system**: Embedded SQL files with version tracking (`internal/database/migrations/`)
- **Core tables**:
  - `speed_tests` - Test results with comprehensive metrics
  - `schedules` - Automated test configurations
  - `users` - Authentication data
  - `saved_iperf_servers` - Custom iperf3 server configs
  - `packet_loss_monitors` - Packet loss monitor configurations
  - `packet_loss_results` - Historical packet loss monitoring results
  - `vnstat_agents` - vnstat agent configurations and connection settings
- **Query patterns**: Interface-based with Squirrel query builder for cross-database compatibility

### Frontend Architecture

- **Framework**: React 18 + TypeScript with functional components
- **Routing**: TanStack Router for client-side navigation
- **State Management**: TanStack Query for server state + React hooks for UI state
- **Styling**: Tailwind CSS with dark theme optimization
- **Charts**: Recharts for speed test visualizations
- **Real-time Updates**: Polling-based progress tracking during tests
- **Unified Traceroute UI**: Combined single-trace and monitoring interface with mode switching (`web/src/components/speedtest/TracerouteTab.tsx`)
- **vnstat Bandwidth UI**: Real-time and historical bandwidth monitoring (`web/src/components/vnstat/VnstatTab.tsx`)

### Chart Data Patterns

- **API Data Ordering**: Backend APIs typically return results in descending order (newest first)
- **Chart Display Order**: Charts should display data chronologically (oldest to newest)
- **Data Transformation Pattern**:
  ```typescript
  const chartData = historyList
    .slice(0, 30) // Take first 30 items (most recent)
    .reverse() // Reverse to chronological order
    .map((item) => ({
      /* transform data */
    }));
  ```
- **Common Mistake**: Using `.slice(-30)` on descending data gets the oldest items instead of newest

### Configuration System

- **Hierarchical TOML** with sections: `[database]`, `[server]`, `[speedtest]`, `[speedtest.iperf]`, `[speedtest.iperf.ping]`, `[speedtest.packetloss]`, `[geoip]`, `[agent]`, `[vnstat]`, etc.
- **Environment overrides**: Any config value can be overridden with `NETRONOME__SECTION_KEY` format
- **Auto-generation**: Creates sensible defaults if no config exists
- **Container detection**: Automatically binds to `0.0.0.0` in containerized environments
- **GeoIP Configuration**: Optional MaxMind GeoLite2 database paths for traceroute country/ASN lookup

### Authentication and Security

- **Dual auth modes**: Local user accounts OR OpenID Connect (OIDC)
- **IP whitelisting**: CIDR-based authentication bypass
- **Session management**: Secure session tokens with configurable secrets
- **CORS handling**: Configurable for API access
- **OIDC Support**: Integration with identity providers (Google, Okta, Auth0, Keycloak, Pocket-ID, Authelia, Authentik etc.) with PKCE support

## Development Patterns

### Error Handling

- Use structured errors with context throughout the codebase
- All external operations (ping, iperf3, librespeed-cli, mtr) include timeout handling
- Database operations use the interface pattern for testability
- HTTP handlers return consistent JSON error responses

### Real-time Progress Updates

- Speed tests broadcast progress via `SpeedUpdate` structs
- Frontend polls `/api/speedtest/status` endpoint during active tests
- Progress includes: test type, server name, speed, completion percentage, latency

### Traceroute Implementation Patterns

- **Cross-platform command execution**: Adapts arguments for `traceroute` (Unix/Linux/macOS) vs `tracert` (Windows)
- **Streaming output parsing**: Real-time line-by-line parsing with regex patterns for each OS
- **Smart termination logic**: Early termination after 3 consecutive timeouts or reaching destination
- **GeoIP enrichment**: Optional country flags and ASN lookup for each hop using MaxMind databases
- **Progress broadcasting**: Real-time hop discovery updates via `TracerouteUpdate` structs
- **Hostname extraction**: Handles URLs, hostnames with ports, and IP addresses uniformly

### Packet Loss Monitoring Patterns

- **Tri-state monitor logic**: Distinguishes between actively testing, scheduled monitoring, and disabled states
- **Context-based goroutine management**: Uses `context.WithTimeout` to prevent hanging on unresponsive hosts
- **Multi-path completion handling**: Supports normal completion, timeout, and fallback scenarios
- **Input validation**: Automatically trims whitespace from hostnames to prevent DNS resolution failures
- **Real-time progress tracking**: Uses OnSend callbacks since OnRecv may not fire for all hosts
- **Startup behavior**: Monitors DO NOT auto-start on program startup by default (prevents network congestion)
- **Manual control**: Users must manually start monitors, which then run on their configured intervals

### Background Services

- Scheduler service runs independently with cron-like intervals
- Notification service processes webhook deliveries
- All services use context for graceful shutdown

### Database Interactions

- All database operations go through the `database.Service` interface
- Use Squirrel query builder for cross-database SQL generation
- Migrations are embedded and run automatically on startup
- Connection health monitoring included

### Configuration Management

- Load order: Default values → TOML file → Environment variables
- All config sections support environment variable overrides
- Use `config.Load()` to get fully resolved configuration
- Configuration validation happens at startup

## Key Dependencies

### Backend (Go 1.23.4)

- **Gin**: HTTP framework and middleware
- **Cobra**: CLI command structure
- **Squirrel**: SQL query builder
- **zerolog**: Structured logging
- **showwin/speedtest-go**: Speedtest.net integration
- **prometheus-community/pro-bing**: Packet loss monitoring via ICMP ping
- **TOML**: Configuration parsing
- **OIDC**: Authentication provider integration

### Frontend (React 18 + TypeScript)

- **TanStack Router**: Client-side routing
- **TanStack Query**: Server state management
- **Tailwind CSS**: Utility-first styling
- **Recharts**: Data visualization
- **Motion**: Smooth animations
- **Heroicons**: Icon system

### External Dependencies

- **iperf3**: Binary required for iperf3 speed tests
- **librespeed-cli**: Binary required for LibreSpeed tests
- **ping**: System ping command (cross-platform support)
- **traceroute/tracert**: System traceroute command (Unix/Linux/macOS) or tracert (Windows)
- **mtr**: Binary for advanced network diagnostics (optional, falls back to ping)
- **vnstat**: Network statistics monitor required for bandwidth monitoring agents

## Special Considerations

### Speed Test Coordination

- Only one speed test can run at a time (enforced by backend)
- Tests can be scheduled or triggered manually
- Progress updates are broadcast to all connected clients
- Failed tests are logged but don't crash the application

### Database Flexibility

- Designed to work with both SQLite (single-user) and PostgreSQL (multi-user)
- Migration system handles schema evolution
- All queries use parameter binding for security

### Deployment Options

- **Single binary**: Frontend assets embedded in Go binary
- **Docker**: Multi-stage builds for optimized images
- **Systemd**: Service file templates provided
- **Development**: Hot reload with `make dev` (requires tmux)

### Configuration Hierarchy

Always follows: Built-in defaults → TOML file → Environment variables (highest priority)

Example: `NETRONOME__IPERF_TEST_DURATION=30` overrides `[speedtest.iperf] test_duration = 10` in TOML.

## Development Guidelines

### Code Standards

- **Conventional Commits**: When suggesting branch names and commit titles, always use Conventional Commit Guidelines
- **License Headers**: Use `./license.sh false` to add GPL-2.0-or-later headers to new source files
- **Commit Attribution**: Never add yourself as co-author to commits
- **Frontend Development**: Before writing any frontend-code, make sure to read through the docs/style-guide.md first, so you familiarize yourself with our style. This is a crucial step.

### Technical Notes

- **Frontend Build**: Frontend is embedded into the Go binary using `embed.FS` during build
- **Container Detection**: The application detects container environments and automatically adjusts network bindings
- **Documentation**: When entering plan mode for work involving TanStack, Tailwind, Motion or Recharts, MUST check the context7 MCP for the most up-to-date documentation

## External Tool Requirements

The following external tools are required for full functionality:

### Required for Development

- **air**: Go live reload tool for `make watch` command
- **tmux**: Terminal multiplexer for `make dev` command
- **pnpm**: Package manager for frontend dependencies

### Required for Speed Tests

- **iperf3**: Binary required for iperf3 speed testing functionality
- **librespeed-cli**: Binary required for LibreSpeed testing (automatically included in Docker)
- **traceroute**: System traceroute command required for traceroute functionality (usually pre-installed on most systems)

### Optional GeoIP Enhancement

- **MaxMind GeoLite2 databases**: For country flags and ASN information in traceroute results
  - Download from MaxMind with free license key
  - Configure paths in `[geoip]` section of config.toml

## Packet Loss Monitoring Implementation Notes

### Backend Architecture (`internal/speedtest/packetloss.go`)

- **Monitor Lifecycle**: Monitors are database-backed configurations that can be enabled/disabled independently of active testing
- **Tri-state Status Logic**:
  - Actively testing: `IsRunning=true, Progress>0` (show progress bar)
  - Scheduled monitoring: `IsRunning=false, IsComplete=false` (show "Monitoring every Xs")
  - Disabled: `IsComplete=true` (show disabled state)
- **Timeout Handling**: Uses multiple completion paths (OnFinish callback, context timeout, goroutine completion, fallback cleanup)
- **Progress Tracking**: Uses OnSend callbacks instead of OnRecv since some hosts block ICMP responses
- **Input Sanitization**: Automatically trims whitespace from hostnames to prevent DNS lookup failures
- **MTR Integration**: Automatically attempts MTR first (if available and has root), falls back to standard ping otherwise

### Frontend Integration (`web/src/components/speedtest/TracerouteTab.tsx`)

- **Unified Interface**: Single tab with mode switching between "Single Trace" and "Monitors"
- **Smart Polling**: Only polls enabled monitors every 2 seconds, stops when no monitors are active
- **Seamless Navigation**: Easy flow between single traces and creating monitors from results
- **Real-time Updates**: Shows progress during active tests, historical results during monitoring periods
- **Responsive Design**: Table view on desktop, card view on mobile
- **State Management**: Uses TanStack Query for server state, local state for UI interactions
- **MTR Results Display**: Shows hop-by-hop statistics when MTR data is available

### Testing Endpoints for Packet Loss

- **Responsive hosts**: `8.8.8.8`, `1.1.1.1` (should show 0-5% loss)
- **Unresponsive hosts**: `203.0.113.1`, `192.0.2.1`, `198.51.100.1` (documentation IPs, should show 100% loss)
- **Geographic distance**: Far locations may show real packet loss due to routing

## Frontend Style Guide Reference

The project includes a comprehensive style guide at `docs/style-guide.md` that covers:

- Component architecture patterns
- Tailwind CSS usage conventions
- Dark mode implementation
- Animation patterns with Motion
- Responsive design approaches
- Color system and semantic usage
- Typography standards
- State management patterns
- Accessibility requirements
- Performance optimizations

Always refer to this guide when implementing new frontend features or modifying existing components.

## Scheduling System

### How Scheduling Works

The scheduler runs every minute checking for due tests/monitors. It supports two interval formats:

1. **Duration-based**: Standard Go duration strings (e.g., "30s", "5m", "1h")
   - Next run = current time + duration + random jitter (1-300 seconds)

2. **Exact time**: "exact:HH:MM" or "exact:HH:MM,HH:MM" for multiple times
   - Next run = next occurrence of specified time + random jitter (1-60 seconds)

### Startup Behavior

- **Missed runs are NOT executed** - prevents network flooding after downtime
- **Next run times are recalculated** based on current time and interval
- **No catch-up mechanism** - ensures fresh data, not stale results

### Important Files

- `internal/scheduler/scheduler.go` - Core scheduling logic
- `calculateNextRun()` - Handles interval parsing and jitter
- `initializePacketLossMonitors()` - Startup initialization for monitors
- `initializeSchedules()` - Startup initialization for speed tests

## vnstat Agent Architecture

### Agent Mode (`netronome agent`)

The vnstat agent is a lightweight SSE server that broadcasts network bandwidth data:

- **Command**: `netronome agent [--host HOST] [--port PORT] [--interface INTERFACE] [--api-key KEY]`
- **Default Port**: 8200
- **Default Host**: 0.0.0.0
- **Authentication**: Optional API key authentication via X-API-Key header or ?apikey= query param
- **SSE Endpoint**: `http://agent-host:port/events?stream=live-data`
- **Historical Export**: `http://agent-host:port/export/historical`
- **Data Source**: Executes `vnstat --live --json` and streams output
- **CORS Support**: Enabled for cross-origin access from Netronome server
- **Graceful Shutdown**: Properly closes connections on termination

### Installation Script

A one-liner installation script is available for easy agent deployment:

```bash
curl -sL https://raw.githubusercontent.com/autobrr/netronome/main/scripts/install-agent.sh | bash
```

Features:

- Interactive configuration prompts
- Automatic API key generation
- Systemd service creation
- Security hardening with dedicated user
- Automatic daily updates (optional)
- Self-update capability via `netronome update` command
- Version checking via `netronome version` command

Script options:

- `--uninstall` - Remove the agent completely
- `--update` - Update to latest version
- `--auto-update [true|false]` - Enable/disable automatic updates without prompting

### Client Integration (`internal/vnstat/`)

- **Service Management**: Manages multiple SSE client connections to remote agents
- **Connection Lifecycle**:
  1. Agent added via UI/API with base URL (e.g., `http://192.168.1.100:8200`)
  2. URL automatically formatted to SSE endpoint
  3. SSE client connects and starts receiving data
  4. Data parsed and stored in database
  5. Live data available via status endpoint
- **Reconnection Strategy**: Exponential backoff from 5s to 5m on connection failure
- **Authentication**: Sends API key via X-API-Key header if configured

### API Endpoints

- `GET /api/vnstat/agents` - List all configured agents
- `POST /api/vnstat/agents` - Add new agent
- `PUT /api/vnstat/agents/:id` - Update agent configuration
- `DELETE /api/vnstat/agents/:id` - Remove agent
- `GET /api/vnstat/agents/:id/status` - Get live connection status and current bandwidth
- `GET /api/vnstat/agents/:id/bandwidth` - Get historical bandwidth data
- `POST /api/vnstat/agents/:id/start` - Start monitoring an agent
- `POST /api/vnstat/agents/:id/stop` - Stop monitoring an agent

### Frontend Components

- **VnstatTab**: Main container component with agent list and details view
- **VnstatAgentList**: Displays all agents with connection status indicators
- **VnstatAgentForm**: Modal form for adding/editing agents
- **VnstatAgentDetails**: Shows real-time and historical bandwidth charts
- **VnstatBandwidthChart**: Recharts-based visualization of bandwidth data

### Configuration

```toml
[agent]
host = "0.0.0.0"     # IP address to bind to
port = 8200          # Port for agent to listen on
interface = ""       # Network interface to monitor (empty = all)
api_key = ""         # API key for authentication (optional but recommended)

[vnstat]
enabled = true       # Enable vnstat client service in main server
reconnect_interval = "30s"  # Reconnection interval for agent connections
```

### vnstat Native Data Architecture

Netronome directly uses vnstat's native JSON output for bandwidth calculations, ensuring exact parity with other vnstat-based tools like swizzin panel.

#### Direct vnstat Integration

- **Native JSON Parsing**: Fetches data from agent's `/export/historical` endpoint
- **No Database Aggregation**: All calculations performed by vnstat itself
- **Real-time Updates**: SSE stream for live bandwidth monitoring
- **Historical Accuracy**: Uses vnstat's own hour/day/month/year calculations

#### Unit Display

Netronome uses proper binary units for bandwidth display:

- **Binary Units**: 1 KiB = 1024 bytes, displayed as "KiB", "MiB", "GiB", "TiB", "PiB"
- **Consistent with vnstat**: vnstat also uses binary units internally
- **Industry Standard**: Follows IEC binary prefix standards for clarity

This ensures accurate representation of data sizes and bandwidth calculations.

# important-instruction-reminders

Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (\*.md) or README files. Only create documentation files if explicitly requested by the User.
