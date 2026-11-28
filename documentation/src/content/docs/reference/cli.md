---
title: CLI
description: All commands and flags of the netronome binary.
sidebar:
  order: 3
---

The `netronome` binary contains the server, the monitoring agent, and the tools for users and updates. Run `netronome <command> --help` to see the help for a command.

```
netronome [command] [flags]
```

## Global flags

All commands accept these flags.

- `--config <path>` (default empty): The path to the configuration file. If empty, Netronome looks in the default locations. See [Configuration file](/reference/configuration-file/).
- `--help`, `-h`: Shows the help for the command.

## serve

Starts the Netronome server and the web interface.

```sh
netronome serve --config ~/.config/netronome/config.toml
```

The server listens on `server.host` and `server.port`, which are `127.0.0.1:7575` by default. If you give `--config` and Netronome cannot read the file, the command stops with an error. Without `--config`, a bad file gives a warning, and the server starts with the default values and the [environment variables](/reference/environment-variables/).

To stop the server, send `SIGINT` or `SIGTERM`. The server then waits up to 5 seconds for open requests to finish. For the first start, see [First run](/getting-started/first-run/).

## generate-config

Writes a configuration file with the default values and a random `session_secret`.

```sh
netronome generate-config
netronome generate-config --config /etc/netronome/config.toml
```

Without `--config`, the command writes `~/.config/netronome/config.toml`. If it cannot make that directory, it tries the configuration directory of the operating system, and then `config.toml` in the working directory. If the file already exists, the command stops with an error and does not change the file. In a container, the file sets `server.host` to `0.0.0.0`.

## create-user

Creates a user for the built-in login.

```sh
netronome create-user <username>
```

In a terminal, the command asks for the password and does not show it. If standard input is not a terminal, the command reads the password from standard input and removes spaces and newlines at the start and the end:

```sh
echo "my-password" | netronome create-user admin
```

The command stops with an error if the password is empty. If no configuration file exists, the command writes one with the default values first. For login options, see [Authentication](/configuration/authentication/).

## change-password

Sets a new password for a user.

```sh
netronome change-password <username>
```

The command reads the password in the same way as `create-user`. It also writes a configuration file if none exists.

## agent

Starts the monitoring agent. The agent sends bandwidth data from vnstat and system data to a Netronome server. For the setup, see [Agents](/monitoring/agents/).

```sh
netronome agent --port 8300 --api-key mysecretkey
```

A flag overrides the matching key in the `[agent]` or `[tailscale]` section of the configuration file.

- `--host`, `-H` (default `0.0.0.0`, key `agent.host`): The IP address that the agent listens on.
- `--port`, `-p` (default `8200`, key `agent.port`): The port that the agent listens on.
- `--interface`, `-i` (default empty, key `agent.interface`): The network interface to monitor. See [The interface setting](/monitoring/agents/#the-interface-setting).
- `--api-key`, `-k` (default empty, key `agent.api_key`): The API key that the server must send.
- `--log-level`, `-l` (default empty, key `logging.level`): The log level: `trace`, `debug`, `info`, `warn`, or `error`.
- `--disk-include` (default empty, key `agent.disk_includes`): Mount points that the agent always reports, for example `/mnt/storage`. Use commas between items, or give the flag more than once.
- `--disk-exclude` (default empty, key `agent.disk_excludes`): Mount points that the agent does not report, for example `/boot`.
- `--disable-system-metrics` (default `false`, key `agent.disable_system_metrics`): Stops the collection of CPU, memory, disk, and temperature data.
- `--tailscale` (default `false`, key `tailscale.enabled`): Starts the agent with Tailscale.
- `--tailscale-hostname` (default `netronome-agent-<hostname>`, key `tailscale.hostname`): The name of the tsnet node.
- `--tailscale-auth-key` (default empty, key `tailscale.auth_key`): The Tailscale auth key, for a host with no login in a browser.
- `--tailscale-state-dir` (default `~/.config/netronome/tsnet`, key `tailscale.state_dir`): The directory for the tsnet state.
- `--tailscale-method` (default `auto`, key `tailscale.method`): The Tailscale method: `auto`, `host`, or `tsnet`.

The agent uses Tailscale only if you give `--tailscale`, or if the file sets the deprecated `tailscale.agent.enabled = true`. The key `tailscale.enabled` alone does not start the agent with Tailscale. The other `--tailscale-*` flags have an effect only when Tailscale is on. For more, see [Tailscale](/monitoring/tailscale/).

With Tailscale, the agent listens on `tailscale.agent_port`, which is `8200` by default. This key overrides `--port`. To use the `--port` value, set `tailscale.agent_port = 0`.

Examples:

```sh
# Monitor one interface
netronome agent --interface eth0

# Join the tailnet with an auth key and a custom name
netronome agent --tailscale --tailscale-auth-key tskey-auth-xxx --tailscale-hostname my-server

# Read the agent settings from a file
netronome agent --config /etc/netronome/agent.toml
```

## update

Replaces the binary with the latest release from GitHub.

```sh
netronome update
```

The command compares the current version with the latest release on `github.com/autobrr/netronome`. If a newer release exists, the command downloads it, compares it with `checksums.txt` from the release, and replaces the binary that runs. After the update, restart the service. The command does not restart it for you.

## version

Shows the version, the Git commit, the build time, the Go version, and the operating system and architecture.

```sh
netronome version
```

A build without release information does not show the commit and the build time.

## help

Shows the list of commands, or the help for one command.

```sh
netronome help agent
```
