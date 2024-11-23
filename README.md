# Netronome

Netronome is a modern network speed testing and monitoring tool built with Go and React. It provides a clean web interface for running network performance tests and visualizing results.

## Features

- Network speed testing using speedtest-go
- Real-time monitoring dashboard
- Modern React frontend
- SQLite database for storing test results
- Docker support

## Tech Stack

- Backend: Go 1.23
- Frontend: React 18 with TypeScript
- Database: SQLite
- UI: Tailwind CSS
- Build Tools: Vite, pnpm
- Container: Docker

## Prerequisites

- Go 1.23 or later
- Node.js 22 or later
- pnpm
- Docker (optional)
- Make

## Getting Started

1. Clone the repository:

```bash
git clone https://github.com/autobrr/netronome.git
cd netronome
```

2. Install dependencies:

```bash
# Install frontend dependencies
cd web && pnpm install
cd ..

# Install Go dependencies
go mod download
```

3. Run the development environment:

```bash
make dev
```

This will start both the frontend and backend development servers with live reload.

## Development Commands

- `make all` - Build everything with tests
- `make build` - Build frontend and backend
- `make run` - Run the application
- `make dev` - Start development environment with live reload
- `make watch` - Live reload backend only
- `make test` - Run test suite
- `make clean` - Clean build artifacts

## Docker Deployment

Build and run with Docker:

```bash
docker compose up -d
```

The application will be available at `http://localhost:8080`

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.
