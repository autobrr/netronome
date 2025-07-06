# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Build and Development

```bash
# Build entire application (frontend + backend)
make build

# Development mode with live reload (requires tmux)
make dev

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
./license.sh      # Add license headers to new files

# Backend live reload during development (requires air)
make watch

# Test commands
go test ./...     # Run Go tests
cd web && pnpm lint  # Frontend linting
```

## Architecture Overview

### Core Application Structure

- **Entry Point**: `cmd/netronome/main.go` using Cobra CLI with commands: `serve`, `generate-config`, `create-user`, `change-password`
- **Configuration**: TOML-based config with environment variable overrides (`NETRONOME__*` prefix)
- **Database**: Supports SQLite (default) and PostgreSQL with embedded migrations
- **Frontend**: React 18 + TypeScript with embedded serving via Go's `embed.FS`

### Service Layer Organization

```
internal/
├── config/          # TOML + env var configuration system
├── database/        # Interface-based DB layer with Squirrel query builder
├── server/          # Gin HTTP server with middleware stack
├── speedtest/       # Core testing logic (4 implementations)
├── scheduler/       # Background cron-like job scheduler
├── auth/           # Session + OIDC authentication
├── notifications/  # Webhook/Discord alert system
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
- Combined with ping tests (`internal/speedtest/ping.go`) for comprehensive metrics
- Supports UDP mode for jitter testing
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

### Database Schema and Migrations

- **Migration system**: Embedded SQL files with version tracking (`internal/database/migrations/`)
- **Core tables**:
  - `speed_tests` - Test results with comprehensive metrics
  - `schedules` - Automated test configurations
  - `users` - Authentication data
  - `saved_iperf_servers` - Custom iperf3 server configs
- **Query patterns**: Interface-based with Squirrel query builder for cross-database compatibility

### Frontend Architecture

- **Framework**: React 18 + TypeScript with functional components
- **Routing**: TanStack Router for client-side navigation
- **State Management**: TanStack Query for server state + React hooks for UI state
- **Styling**: Tailwind CSS with dark theme optimization
- **Charts**: Recharts for speed test visualizations
- **Real-time Updates**: Polling-based progress tracking during tests
- **Traceroute UI**: Responsive table/card views with real-time hop discovery (`web/src/components/speedtest/TracerouteTab.tsx`)

### Configuration System

- **Hierarchical TOML** with sections: `[database]`, `[server]`, `[speedtest]`, `[speedtest.iperf]`, `[speedtest.iperf.ping]`, `[geoip]`, etc.
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
- All external operations (ping, iperf3, librespeed-cli) include timeout handling
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
- **License Headers**: Use `./license.sh` to add GPL-2.0-or-later headers to new source files
- **Commit Attribution**: Never add yourself as co-author to commits

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