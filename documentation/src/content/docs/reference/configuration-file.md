---
title: Configuration file
description: All sections and keys of config.toml, with the type, the default value, and what each key does.
sidebar:
  order: 1
---

Netronome reads its configuration from a TOML file named `config.toml`. Each key in this file also has an environment variable. An environment variable overrides the value in the file. For the variable names, see [Environment variables](/reference/environment-variables/).

## File location

If you start Netronome with `--config <path>`, it reads only that file. If you do not give `--config`, Netronome looks for the file in these locations, in this order:

1. `~/.config/netronome/config.toml`
2. `netronome/config.toml` in the configuration directory of your operating system. On Linux, this is also `~/.config`. On macOS, it is `~/Library/Application Support`.
3. `config.toml` in the working directory.

If Netronome finds no file, it starts with the default values and the environment variables. To write a file with the default values, run `netronome generate-config`. For more, see [CLI](/reference/cli/).

Netronome resolves a relative `database.path` from the directory of the configuration file. It also reads `librespeed-servers.json` from that directory.

The default values in the tables below are the values that Netronome uses when a key is missing from the file. The file that `generate-config` writes can contain other values. For example, it sets `server.host` to `0.0.0.0` in a container and writes a random `session_secret`.

## Top-level keys

These keys go at the top of the file, before the first section header.

| Key | Type | Default | Description |
|---|---|---|---|
| `check_for_updates` | boolean | `true` | Checks GitHub for new releases and shows a notice in the web interface. See [FAQ](/help/faq/#how-do-i-stop-the-update-notice). |

## [database]

Netronome uses SQLite by default. For PostgreSQL, see [Database](/configuration/database/).

| Key | Type | Default | Description |
|---|---|---|---|
| `type` | string | `"sqlite"` | The database type: `sqlite` or `postgres`. |
| `path` | string | `"netronome.db"` | The SQLite database file. A relative path starts at the directory of the configuration file. SQLite only. |
| `host` | string | `"localhost"` | The PostgreSQL host. |
| `port` | integer | `5432` | The PostgreSQL port. |
| `user` | string | `"postgres"` | The PostgreSQL user. |
| `password` | string | `""` | The PostgreSQL password. |
| `dbname` | string | `"netronome"` | The PostgreSQL database name. |
| `sslmode` | string | `"disable"` | The PostgreSQL SSL mode, for example `disable` or `require`. |

## [server]

For `base_url` behind a reverse proxy, see [Reverse proxy](/configuration/reverse-proxy/).

| Key | Type | Default | Description |
|---|---|---|---|
| `host` | string | `"127.0.0.1"` | The address that the web server listens on. See [Reverse proxy](/configuration/reverse-proxy/). |
| `port` | integer | `7575` | The port of the web server. |
| `base_url` | string | `"/"` | The URL path that Netronome runs under, for example `/netronome`. |
| `gin_mode` | string | `""` | The Gin web framework mode: `debug`, `release`, or `test`. If empty, Netronome uses `release`. |

## [logging]

| Key | Type | Default | Description |
|---|---|---|---|
| `level` | string | `"info"` | The log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, or `panic`. |

## [auth]

For login, OIDC, and the IP whitelist, see [Authentication](/configuration/authentication/).

| Key | Type | Default | Description |
|---|---|---|---|
| `whitelist` | list of strings | `[]` | Networks in CIDR notation that can use Netronome without a login, for example `["127.0.0.1/32"]`. |
| `trusted_proxies` | list of strings | `[]` | Proxies that Netronome trusts for the client address headers. See [Behind a reverse proxy](/configuration/authentication/#behind-a-reverse-proxy). |

Before you set `whitelist = ["0.0.0.0/0", "::/0"]`, read [Turn off authentication](/configuration/authentication/#turn-off-authentication).

## [oidc]

Netronome turns on OpenID Connect (OIDC) login when `issuer` has a value. For the setup, see [Authentication](/configuration/authentication/).

| Key | Type | Default | Description |
|---|---|---|---|
| `issuer` | string | `""` | The URL of the OIDC provider. |
| `client_id` | string | `""` | The client ID from the OIDC provider. |
| `client_secret` | string | `""` | The client secret from the OIDC provider. |
| `redirect_url` | string | `""` | The callback URL, for example `https://netronome.example.com/api/auth/oidc/callback`. |
| `scopes` | list of strings | `[]` | The scopes to request. If empty, Netronome requests `openid` and `profile`. See [OIDC](/configuration/authentication/#openid-connect-oidc). |

## [session]

| Key | Type | Default | Description |
|---|---|---|---|
| `session_secret` | string | `""` | The key that signs session tokens and encrypts OIDC refresh tokens. See [Sessions](/configuration/authentication/#sessions). |

## [speedtest]

For the test types, see [Speed tests](/configuration/speed-tests/).

| Key | Type | Default | Description |
|---|---|---|---|
| `timeout` | integer | `30` | The time limit in seconds for a speed test. See [Speed tests](/configuration/speed-tests/#speedtestnet). |
| `connections` | integer | `0` | The maximum number of Speedtest.net connections. `0` keeps the library default. See [Speed tests](/configuration/speed-tests/#speedtestnet). |

Before you set a high `connections` value, read the caution in [Speed tests](/configuration/speed-tests/#speedtestnet).

### [speedtest.iperf]

| Key | Type | Default | Description |
|---|---|---|---|
| `test_duration` | integer | `10` | The length of an iperf3 test in seconds. |
| `parallel_conns` | integer | `4` | The number of parallel iperf3 streams. |
| `timeout` | integer | `60` | The time limit in seconds for an iperf3 test. |

### [speedtest.iperf.ping]

These keys control the ping test that measures latency.

| Key | Type | Default | Description |
|---|---|---|---|
| `count` | integer | `5` | The number of ping packets. |
| `interval` | integer | `1000` | The time between packets in milliseconds. |
| `timeout` | integer | `10` | The time limit in seconds for the ping test. |

### [speedtest.librespeed]

| Key | Type | Default | Description |
|---|---|---|---|
| `timeout` | integer | `60` | The time limit in seconds for a LibreSpeed test. If the file sets `0`, Netronome uses `60`. |

## [pagination]

These keys set the defaults for the speed test history. A request to the API can override each one.

| Key | Type | Default | Description |
|---|---|---|---|
| `default_page` | integer | `1` | The first page to show. |
| `default_time_range` | string | `"1w"` | The time range of the history, for example `1w` for one week. |
| `default_limit` | integer | `20` | The number of results on one page. |

## [geoip]

GeoIP adds country flags and ASN names to traceroute results. Both keys are empty by default, and GeoIP is off. For the setup, see [GeoIP](/configuration/geoip/).

| Key | Type | Default | Description |
|---|---|---|---|
| `country_database_path` | string | `""` | The path to `GeoLite2-Country.mmdb`. |
| `asn_database_path` | string | `""` | The path to `GeoLite2-ASN.mmdb`. |

## [packetloss]

For packet loss monitors and MTR, see [Packet loss](/monitoring/packet-loss/).

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | `true` | Turns on packet loss monitoring. |
| `max_concurrent_monitors` | integer | `10` | Netronome reads this value but does not apply it. See [Packet loss](/monitoring/packet-loss/). |
| `privileged_mode` | boolean | `true` | Tries ICMP first. See [Privileges](/monitoring/packet-loss/#privileges). |
| `mtr_enable_dns` | boolean | `false` | Resolves hop addresses to hostnames in MTR results. |

## [monitor]

For remote system monitoring, see [Agents](/monitoring/agents/).

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | `true` | Turns on the monitor service. The server uses it to collect data from agents. |

## [agent]

The `netronome agent` command reads this section. The server does not use it. Each key also has a command-line flag. For the setup, see [Agents](/monitoring/agents/).

| Key | Type | Default | Description |
|---|---|---|---|
| `host` | string | `"0.0.0.0"` | The address that the agent listens on. |
| `port` | integer | `8200` | The port of the agent. |
| `interface` | string | `""` | The network interface that vnstat monitors. See [The interface setting](/monitoring/agents/#the-interface-setting). |
| `api_key` | string | `""` | The API key that the server must send. If empty, the agent does not ask for a key. |
| `disk_includes` | list of strings | `[]` | Mount points that the agent always reports. See [Disk filters](/monitoring/agents/#disk-filters). |
| `disk_excludes` | list of strings | `[]` | Mount points that the agent does not report. See [Disk filters](/monitoring/agents/#disk-filters). |
| `disable_system_metrics` | boolean | `false` | Stops the collection of CPU, memory, disk, and temperature data. |

## [tailscale]

Tailscale connects the server and the agents over your tailnet. For the setup, see [Tailscale](/monitoring/tailscale/).

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | `false` | Turns on the Tailscale integration. |
| `method` | string | `"auto"` | How Netronome connects to Tailscale: `auto`, `host`, or `tsnet`. See [Methods](/monitoring/tailscale/#methods). |
| `auth_key` | string | `""` | The Tailscale auth key. The `tsnet` method must have one. |
| `hostname` | string | `""` | The name of the tsnet node. See [Tailscale](/monitoring/tailscale/). |
| `ephemeral` | boolean | `false` | Removes the tsnet node from the tailnet when Netronome stops. |
| `state_dir` | string | `"~/.config/netronome/tsnet"` | The directory for the tsnet state. |
| `control_url` | string | `""` | The URL of a different control server, for example Headscale. |
| `agent_port` | integer | `8200` | The port that the agent listens on over Tailscale. `0` uses `agent.port`. See [tsnet mode](/monitoring/tailscale/#tsnet-mode). |
| `auto_discover` | boolean | `true` | The server finds Netronome agents on the tailnet. |
| `discovery_interval` | string | `"5m"` | The time between two discovery runs, as a Go duration, for example `5m` or `1h`. |
| `discovery_port` | integer | `8200` | The port that the server probes on each Tailscale peer. |
| `discovery_prefix` | string | `""` | Netronome does not read this key. To filter by prefix, use `tailscale.monitor.discovery_prefix`. |
| `prefer_host` | boolean | `false` | Deprecated. Use `method = "host"`. |

### [tailscale.agent]

This section is deprecated. Use the keys in `[tailscale]`.

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | `false` | Deprecated. If `true`, `netronome agent` starts with Tailscale, like `--tailscale`. |
| `port` | integer | `8200` | Deprecated. Use `tailscale.agent_port`. |

### [tailscale.monitor]

This section is deprecated. Use the keys in `[tailscale]`.

| Key | Type | Default | Description |
|---|---|---|---|
| `auto_discover` | boolean | `true` | Deprecated. Use `tailscale.auto_discover`. |
| `discovery_interval` | string | `"5m"` | Deprecated. Use `tailscale.discovery_interval`. |
| `discovery_port` | integer | `8200` | Deprecated. Use `tailscale.discovery_port`. |
| `discovery_prefix` | string | `""` | Deprecated, but discovery reads only this key. The server adds only peers whose hostname starts with this prefix. |

## Settings that are not in the file

You set notifications, schedules, DNS monitors, and packet loss monitors in the web interface. Netronome keeps them in the database. See [Notifications](/configuration/notifications/), [Scheduling](/configuration/scheduling/), and [DNS](/monitoring/dns/).

## Example

```toml title="config.toml"
check_for_updates = true

[database]
type = "sqlite"
path = "netronome.db"

[server]
host = "0.0.0.0"
port = 7575

[logging]
level = "info"

[auth]
whitelist = []
trusted_proxies = []

[session]
session_secret = "<random value>"

[speedtest]
timeout = 30
connections = 0

[speedtest.iperf]
test_duration = 10
parallel_conns = 4
timeout = 60

[speedtest.iperf.ping]
count = 5
interval = 1000
timeout = 10

[speedtest.librespeed]
timeout = 60

[packetloss]
enabled = true
max_concurrent_monitors = 10
privileged_mode = true
mtr_enable_dns = false

[monitor]
enabled = true

[tailscale]
enabled = false
method = "auto"
```
