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

## Development Guidelines

### Code Standards

- **Conventional Commits**: When suggesting branch names and commit titles, always use Conventional Commit Guidelines
- **License Headers**: Use `./license.sh false` to add GPL-2.0-or-later headers to new source files
- **Commit Attribution**: Never add yourself as co-author to commits
- **Frontend Development**: Before writing any frontend-code, make sure to read through the docs/style-guide.md first, so you familiarize yourself with our style. This is a crucial step.
- **Import Paths**: Always use the `@` alias for imports in frontend code (e.g., `@/components/...` instead of relative paths like `../components/...`)

## Additional Notes

- **Please read CLAUDE.local.md before writing commits or PRs.**