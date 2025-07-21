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

3. **Agent Architecture**:
   - Lightweight HTTP server exposing SSE endpoints
   - Collects system metrics via gopsutil and vnstat
   - Can run standalone or integrated with Tailscale
   - Auto-discovery support for Tailscale networks

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

## Development Guidelines

### Code Standards

- **Conventional Commits**: When suggesting branch names and commit titles, always use Conventional Commit Guidelines
- **License Headers**: Use `./license.sh false` to add GPL-2.0-or-later headers to new source files
- **Commit Attribution**: Never add yourself as co-author to commits
- **Frontend Development**: Before writing any frontend-code, make sure to read through the ai_docs/style-guide.md first, so you familiarize yourself with our style. This is a crucial step.
- **Import Paths**: Always use the `@` alias for imports in frontend code (e.g., `@/components/...` instead of relative paths like `../components/...`)

## Additional Notes

- **Please read CLAUDE.local.md before writing commits or PRs.**