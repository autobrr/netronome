---
title: Build from source
description: Build the Netronome binary and Docker image from source, and contribute changes.
sidebar:
  order: 2
---

The build puts the web interface and the backend into one binary, `bin/netronome`.

## Prerequisites

- Go 1.27 or later.
- Node.js and pnpm. The web interface uses pnpm only.
- `make` and `git`.

## Build with SMART support

SMART support adds SATA and HDD temperatures and disk model names to the agent data. It works on Linux and macOS.

```bash
git clone https://github.com/autobrr/netronome
cd netronome
make build
```

`make build` installs the web dependencies, builds the web interface, and compiles `bin/netronome`.

On macOS, SMART support uses cgo, so you need a C compiler. On Linux, you do not.

:::note
The binaries on the Releases page do not include SMART support. The Docker image and a `make build` binary include it.
:::

## Build without SMART support

The `nosmart` build tag removes SMART support. The release binaries use this tag. Builds for platforms other than Linux and macOS do not include SMART support.

```bash
# First build frontend
cd web && pnpm install && pnpm build && cd ..
# Then build Go binary
CGO_ENABLED=0 go build -tags nosmart -o bin/netronome ./cmd/netronome
```

Build the web interface first. The Go binary embeds the files in `web/dist`.

## Premium themes in source builds

A source build has the default theme only. The premium themes come from a private repository, and `make build` fetches them only when you set `THEMES_REPO_TOKEN`. A source build also has no license server ID, so the server logs `No Polar organization ID configured - premium themes will be disabled` at start. You can ignore this message.

## Build the Docker image

```bash
make docker-build
make docker-run
```

`make docker-build` builds `distrib/docker/Dockerfile` into an image with the name `netronome`. `make docker-run` builds the image and runs it with port `7575` published.

:::caution
`make docker-run` starts the container without a configuration file and without a volume. The server then listens on `127.0.0.1` inside the container, and you cannot open it from the host. To test the image, run it with the listen address set:

```bash
docker run -p 7575:7575 -e NETRONOME__HOST=0.0.0.0 netronome
```

This container keeps its data only until you remove it. For a permanent setup, see the Docker steps in [Installation](/getting-started/installation/).
:::

## Development

- `make dev`: Starts the Vite dev server and the Go watcher (`air`) in tmux.
- `make watch`: Starts only the Go watcher. It offers to install `air` if it is missing.
- `make run`: Builds and runs `serve --config config/config.toml`.
- `pnpm -C web dev`: Starts the web dev server.
- `pnpm -C web lint`: Lints the web interface with ESLint.
- `pnpm -C web test`: Runs the web unit tests.
- `go test ./...`: Runs the backend unit tests.

## Contributing

Read [CONTRIBUTING.md](https://github.com/autobrr/netronome/blob/develop/.github/CONTRIBUTING.md) before you open an issue or a pull request. The short version:

- Pick a clear issue and keep the change to that issue.
- Do not include unrelated refactors or formatting changes.
- Use pnpm. Do not add `package-lock.json`.
- Use Conventional Commits, for example `fix(speedtest): resolve server redirects`.
- In the pull request, say what changed, why, and how you tested it. Add screenshots for changes to the web interface.

You can use AI tools, but you must review and edit the change yourself before you open the pull request.

To send a change:

1. Fork the repository.
2. Create a branch from `develop`: `git checkout -b feature/amazing-feature`.
3. Commit your changes: `git commit -m 'feat: add amazing feature'`.
4. Push the branch: `git push origin feature/amazing-feature`.
5. Open a pull request against `develop`.
