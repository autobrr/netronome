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

# Run agent mode
./bin/netronome agent --config config.toml
```

### Frontend Development

```bash
cd web
pnpm install     # Install dependencies
pnpm dev         # Start dev server (port 5173)
pnpm build       # Build for production
pnpm lint        # Run ESLint
pnpm tsc --noEmit # Type checking
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

### Testing

```bash
# Run all Go tests
go test ./...

# Run tests for a specific package
go test ./internal/server/...

# Run tests with coverage
go test -cover ./...

# Run a specific test with verbose output
go test -v -run TestIsWhitelisted ./internal/server

# Frontend linting
cd web && pnpm lint

# Frontend type checking
cd web && pnpm tsc --noEmit
```

## High-Level Architecture

### Backend Architecture (Go)

The backend follows a clean architecture pattern with dependency injection:

1. **Entry Point** (`cmd/netronome/main.go`): CLI commands using Cobra

   - `serve`: Runs the web server
   - `agent`: Runs the monitoring agent
   - `generate-config`: Creates default configuration
   - User management commands

2. **Core Services** (`internal/`):

   - **server**: HTTP server using Gin framework, handles routing and middleware
   - **database**: Data persistence layer with SQLite/PostgreSQL support via interfaces
   - **speedtest**: Core speed testing logic (iperf3, librespeed, speedtest.net)
   - **monitor**: System monitoring and agent management
   - **scheduler**: Cron-like scheduling for automated tests
   - **auth**: Authentication (built-in and OIDC support)
   - **broadcaster**: WebSocket/SSE for real-time updates
   - **tailscale**: Tailscale integration for secure networking
   - **notifications**: Shoutrrr-based notification system supporting 15+ services

3. **Agent Architecture**:

   - Lightweight HTTP server exposing SSE endpoints
   - Collects system metrics via gopsutil and vnstat
   - Can run standalone or integrated with Tailscale
   - Auto-discovery support for Tailscale networks
   - Temperature sensor monitoring with per-component details

4. **Database Patterns**:
   - Interface-based design for multiple backends
   - Migrations in `internal/database/migrations/`
   - Squirrel query builder for complex queries
   - Separate implementations for SQLite and PostgreSQL

### Frontend Architecture (React + TypeScript)

The frontend uses modern React patterns with TypeScript:

1. **Core Stack**:

   - React 19 with TypeScript
   - TanStack Query for data fetching and caching
   - TanStack Router for routing
   - Tailwind CSS v4 for styling
   - Motion (framer-motion) for animations
   - Vite for bundling

2. **Component Organization**:

   - `components/auth/`: Authentication components
   - `components/common/`: Shared UI components
   - `components/speedtest/`: Speed test features
   - `components/monitor/`: System monitoring UI
   - `components/settings/`: Configuration and settings UI
   - `components/settings/notifications/`: Notification management components
   - `components/ui/`: Base UI components

3. **State Management**:

   - Local state with useState for component state
   - TanStack Query for server state
   - localStorage for user preferences
   - Context API for authentication

4. **API Integration** (`api/`):
   - Type-safe API clients
   - Error handling and retry logic
   - WebSocket/SSE connections for real-time data

### Key Architectural Decisions

1. **Embedded Frontend**: Frontend is built and embedded into the Go binary for single-file deployment

2. **Real-time Updates**: Uses Server-Sent Events (SSE) for live monitoring data and WebSocket for speed test progress

3. **Plugin Architecture**: Speed test providers implement common interfaces, making it easy to add new providers

4. **Agent-Based Monitoring**: Distributed architecture where agents can be deployed separately from the main server

5. **Tailscale Integration**: Optional but deeply integrated, supporting both tsnet and host modes for flexible deployment

6. **Notification System**: Uses Shoutrrr library for multi-service notifications with rate limiting and state-based alerts

## Database Migrations

The project uses a custom migration system with separate SQL files for SQLite and PostgreSQL:

```
internal/database/migrations/
├── migrations.go        # Migration runner
├── postgres/           # PostgreSQL migrations
│   └── *.sql
└── sqlite/            # SQLite migrations
    └── *.sql
```

Key points:

- Migrations are numbered sequentially (001, 002, etc.)
- Each migration has a corresponding file for both SQLite and PostgreSQL
- The system tracks applied migrations in a `schema_migrations` table
- New features should add migrations following the existing pattern

## Notification System

The notification system is built on [Shoutrrr](https://github.com/containrrr/shoutrrr) and supports 15+ services:

### Supported Services

- Discord, Telegram, Slack, Teams
- Email (SMTP), Pushover, Pushbullet
- Gotify, Matrix, Ntfy, OpsGenie
- Rocketchat, Zulip, Join, Mattermost

### Notification Events

1. **Speed Test Events**

   - Test completed
   - Test failed
   - Speed below threshold
   - Latency above threshold

2. **Packet Loss Monitoring**

   - Packet loss degraded (state-based)
   - Packet loss recovered

3. **Agent Monitoring**
   - Agent online/offline
   - CPU usage threshold
   - Memory usage threshold
   - Disk usage threshold
   - Bandwidth usage threshold
   - Temperature threshold (with sensor details)

### Rate Limiting

- Agent metric notifications: 1 hour cooldown
- Packet loss: State-based (only on state change)
- Speed tests: Per test completion

## Development Guidelines

### Code Standards

- **Conventional Commits**: When suggesting branch names and commit titles, always use Conventional Commit Guidelines
- **License Headers**: Use `./license.sh false` to add GPL-2.0-or-later headers to new source files
- **Commit Attribution**: Never add yourself as co-author to commits
- **Frontend Development**: ALWAYS read `ai_docs/style-guide.md` before writing any frontend code - this contains essential patterns for React, TypeScript, Tailwind CSS v4, Motion animations, and component architecture
- **Import Paths**: Always use the `@` alias for imports in frontend code (e.g., `@/components/...` instead of relative paths like `../components/...`)

### Commit Guidelines

- **Commit Attribution**: When writing commits for the user, never add Co-Authored-By: Claude <noreply@anthropic.com> and/or 🤖 Generated with [Claude Code](https://claude.ai/code) to the commit details

### Testing Approach

1. **Backend Testing**

   - Unit tests for business logic
   - Integration tests for database operations
   - Mock interfaces for external dependencies
   - Use testify for assertions

2. **Frontend Testing**
   - Component testing with React Testing Library
   - Type safety with TypeScript
   - Linting with ESLint

### Common Patterns

1. **Error Handling**

   - Return errors from functions, don't panic
   - Use structured logging with zerolog
   - Wrap errors with context using `fmt.Errorf`

2. **API Responses**

   - Consistent JSON structure
   - Proper HTTP status codes
   - Error messages in `error` field

3. **Database Operations**
   - Use transactions for multi-step operations
   - Always use parameterized queries
   - Handle null values appropriately

## Agent Architecture Details

The agent package (`internal/agent/`) has been refactored into focused, single-responsibility modules:

1. **Core Agent** (`agent.go`): ~95 lines - Main constructor and Start() method only
2. **Tailscale Integration** (`tailscale.go`): Handles both tsnet and host mode startup logic
3. **Broadcasting** (`broadcast.go`): SSE/real-time data streaming to clients
4. **Bandwidth Monitoring** (`bandwidth.go`): vnstat integration with peak bandwidth tracking
5. **System Info** (`system.go`): OS details, network interfaces, and vnstat data collection
6. **Hardware Stats** (`hardware.go`): CPU, memory, disk usage, and temperature via gopsutil
7. **Disk Utilities** (`disk_utils.go`): Path matching and device discovery with glob support
8. **SMART Monitoring** (`smart.go`/`smart_stub.go`): Platform-specific disk health (Linux/macOS only)

**Deployment Modes:**
- Standalone HTTP server (default)
- Tailscale tsnet (creates new Tailscale node)
- Tailscale host mode (uses existing tailscaled)

## Testing Specific Components

```bash
# Test speed test implementations
go test -v ./internal/speedtest/...

# Test database migrations
go test -v ./internal/database/migrations/...

# Test monitor service
go test -v ./internal/monitor/...

# Test with race detector
go test -race ./...

# Benchmark tests
go test -bench=. ./internal/...
```

## Important Project-Specific Details

- **Frontend Style Guide**: The `ai_docs/style-guide.md` is the authoritative reference for all frontend development, containing:

  - React component patterns with TypeScript interfaces
  - Tailwind CSS v4 utility classes and dark mode patterns
  - Motion (framer-motion) animation configurations and timing
  - Responsive design breakpoints and mobile-first patterns
  - Comprehensive color system with semantic usage guidelines
  - Typography standards and accessibility requirements

- **Agent Discovery**: The Tailscale discovery service automatically finds and adds agents on the network:

  - Only discovers agents with Tailscale enabled
  - Uses DNSName (not HostName) for proper identification
  - Supports auto-discovery via the `discovery_interval` config

- **Migration Patterns**: When adding new features requiring database changes:
  1. Create migration files in both `sqlite/` and `postgres/` directories
  2. Use sequential numbering (e.g., 025_feature_name.sql) and append \_postgres to the filename for PostgreSQL migrations
  3. Test both database types before committing

## Additional Notes

- **Always read CLAUDE.local.md if it exists, regardless of .gitignore**
- The project uses semantic versioning
- All new features should include appropriate tests
- Documentation updates are expected with feature changes
- When working on frontend code, always check `ai_docs/style-guide.md` first

## Guidelines

- **Emojis**: Never use emojis