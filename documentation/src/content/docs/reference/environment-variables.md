---
title: Environment variables
description: All NETRONOME__ environment variables, grouped by configuration section, with the default value of each.
sidebar:
  order: 2
---

Each key in `config.toml` has an environment variable. The name starts with `NETRONOME__`, with two underscores. An environment variable overrides the value in the configuration file. For the meaning of each key, see [Configuration file](/reference/configuration-file/).

## Rules

- Netronome ignores a variable with an empty value. `NETRONOME__TAILSCALE_AUTH_KEY` is the only exception: an empty value clears the key.
- A boolean accepts `true`, `false`, `1`, `0`, `t`, and `f`. Netronome ignores a value that it cannot parse, and keeps the value from the file.
- A list uses commas between the items, for example `127.0.0.1/32,10.0.0.0/8`.
- At startup, Netronome loads a `.env` file from the working directory, if one exists.

## Top level

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__CHECK_FOR_UPDATES` | `check_for_updates` | `true` |

## Database

For PostgreSQL, see [Database](/configuration/database/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__DB_TYPE` | `database.type` | `sqlite` |
| `NETRONOME__DB_PATH` | `database.path` | `netronome.db` |
| `NETRONOME__DB_HOST` | `database.host` | `localhost` |
| `NETRONOME__DB_PORT` | `database.port` | `5432` |
| `NETRONOME__DB_USER` | `database.user` | `postgres` |
| `NETRONOME__DB_PASSWORD` | `database.password` | empty |
| `NETRONOME__DB_NAME` | `database.dbname` | `netronome` |
| `NETRONOME__DB_SSLMODE` | `database.sslmode` | `disable` |

## Server

For `BASE_URL`, see [Reverse proxy](/configuration/reverse-proxy/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__HOST` | `server.host` | `127.0.0.1` |
| `NETRONOME__PORT` | `server.port` | `7575` |
| `NETRONOME__BASE_URL` | `server.base_url` | `/` |
| `NETRONOME__GIN_MODE` | `server.gin_mode` | empty (`release`) |

Netronome removes quotes around the value of `NETRONOME__BASE_URL`.

## Logging

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__LOG_LEVEL` | `logging.level` | `info` |

## Authentication

For the whitelist and trusted proxies, see [Authentication](/configuration/authentication/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__AUTH_WHITELIST` | `auth.whitelist` | empty |
| `NETRONOME__AUTH_TRUSTED_PROXIES` | `auth.trusted_proxies` | empty |
| `NETRONOME__SESSION_SECRET` | `session.session_secret` | empty |

Do not put spaces after the commas in `NETRONOME__AUTH_WHITELIST`. Netronome does not remove them, and an entry with a space is not a valid CIDR network.

## OIDC

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__OIDC_ISSUER` | `oidc.issuer` | empty |
| `NETRONOME__OIDC_CLIENT_ID` | `oidc.client_id` | empty |
| `NETRONOME__OIDC_CLIENT_SECRET` | `oidc.client_secret` | empty |
| `NETRONOME__OIDC_REDIRECT_URL` | `oidc.redirect_url` | empty |
| `NETRONOME__OIDC_SCOPES` | `oidc.scopes` | empty (`openid`, `profile`) |

`NETRONOME__OIDC_SCOPES` accepts commas or spaces between the scopes.

## Speed tests

For the test types, see [Speed tests](/configuration/speed-tests/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__SPEEDTEST_TIMEOUT` | `speedtest.timeout` | `30` |
| `NETRONOME__SPEEDTEST_CONNECTIONS` | `speedtest.connections` | `0` |
| `NETRONOME__IPERF_TEST_DURATION` | `speedtest.iperf.test_duration` | `10` |
| `NETRONOME__IPERF_PARALLEL_CONNS` | `speedtest.iperf.parallel_conns` | `4` |
| `NETRONOME__IPERF_TIMEOUT` | `speedtest.iperf.timeout` | `60` |
| `NETRONOME__IPERF_PING_COUNT` | `speedtest.iperf.ping.count` | `5` |
| `NETRONOME__IPERF_PING_INTERVAL` | `speedtest.iperf.ping.interval` | `1000` |
| `NETRONOME__IPERF_PING_TIMEOUT` | `speedtest.iperf.ping.timeout` | `10` |
| `NETRONOME__LIBRESPEED_TIMEOUT` | `speedtest.librespeed.timeout` | `60` |

## Pagination

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__DEFAULT_PAGE` | `pagination.default_page` | `1` |
| `NETRONOME__DEFAULT_TIME_RANGE` | `pagination.default_time_range` | `1w` |
| `NETRONOME__DEFAULT_LIMIT` | `pagination.default_limit` | `20` |

## GeoIP

For the setup, see [GeoIP](/configuration/geoip/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__GEOIP_COUNTRY_DATABASE_PATH` | `geoip.country_database_path` | empty |
| `NETRONOME__GEOIP_ASN_DATABASE_PATH` | `geoip.asn_database_path` | empty |

## Packet loss

For the setup, see [Packet loss](/monitoring/packet-loss/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__PACKETLOSS_ENABLED` | `packetloss.enabled` | `true` |
| `NETRONOME__PACKETLOSS_MAX_CONCURRENT_MONITORS` | `packetloss.max_concurrent_monitors` | `10` |
| `NETRONOME__PACKETLOSS_PRIVILEGED_MODE` | `packetloss.privileged_mode` | `true` |
| `NETRONOME__PACKETLOSS_MTR_ENABLE_DNS` | `packetloss.mtr_enable_dns` | `false` |

## Monitor

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__MONITOR_ENABLED` | `monitor.enabled` | `true` |

## Agent

The `netronome agent` command reads these variables. For the setup, see [Agents](/monitoring/agents/) and [Docker agents](/monitoring/docker-agents/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__AGENT_HOST` | `agent.host` | `0.0.0.0` |
| `NETRONOME__AGENT_PORT` | `agent.port` | `8200` |
| `NETRONOME__AGENT_INTERFACE` | `agent.interface` | empty. See [The interface setting](/monitoring/agents/#the-interface-setting). |
| `NETRONOME__AGENT_API_KEY` | `agent.api_key` | empty |
| `NETRONOME__AGENT_DISK_INCLUDES` | `agent.disk_includes` | empty |
| `NETRONOME__AGENT_DISK_EXCLUDES` | `agent.disk_excludes` | empty |
| `NETRONOME__AGENT_DISABLE_SYSTEM_METRICS` | `agent.disable_system_metrics` | `false` |

## Tailscale

For the setup, see [Tailscale](/monitoring/tailscale/).

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__TAILSCALE_ENABLED` | `tailscale.enabled` | `false` |
| `NETRONOME__TAILSCALE_METHOD` | `tailscale.method` | `auto` |
| `NETRONOME__TAILSCALE_AUTH_KEY` | `tailscale.auth_key` | empty |
| `NETRONOME__TAILSCALE_HOSTNAME` | `tailscale.hostname` | empty |
| `NETRONOME__TAILSCALE_EPHEMERAL` | `tailscale.ephemeral` | `false` |
| `NETRONOME__TAILSCALE_STATE_DIR` | `tailscale.state_dir` | `~/.config/netronome/tsnet` |
| `NETRONOME__TAILSCALE_CONTROL_URL` | `tailscale.control_url` | empty |
| `NETRONOME__TAILSCALE_AGENT_PORT` | `tailscale.agent_port` and `tailscale.agent.port` | `8200` |
| `NETRONOME__TAILSCALE_AUTO_DISCOVER` | `tailscale.auto_discover` | `true` |
| `NETRONOME__TAILSCALE_DISCOVERY_INTERVAL` | `tailscale.discovery_interval` | `5m` |
| `NETRONOME__TAILSCALE_DISCOVERY_PORT` | `tailscale.discovery_port` | `8200` |
| `NETRONOME__TAILSCALE_DISCOVERY_PREFIX` | `tailscale.discovery_prefix` | empty |

The discovery filter does not read `tailscale.discovery_prefix`. To filter discovered agents by hostname prefix, use `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_PREFIX`.

### Deprecated Tailscale variables

These variables are deprecated but still work. The `MONITOR` variables also set the matching new key.

| Variable | Key | Default |
|---|---|---|
| `NETRONOME__TAILSCALE_PREFER_HOST` | `tailscale.prefer_host` | `false` |
| `NETRONOME__TAILSCALE_AGENT_ENABLED` | `tailscale.agent.enabled` | `false` |
| `NETRONOME__TAILSCALE_MONITOR_AUTO_DISCOVER` | `tailscale.monitor.auto_discover` | `true` |
| `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_INTERVAL` | `tailscale.monitor.discovery_interval` | `5m` |
| `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_PORT` | `tailscale.monitor.discovery_port` | `8200` |
| `NETRONOME__TAILSCALE_MONITOR_DISCOVERY_PREFIX` | `tailscale.monitor.discovery_prefix` | empty |

If `NETRONOME__TAILSCALE_PREFER_HOST` is `true` and `NETRONOME__TAILSCALE_METHOD` is not set, Netronome uses the `host` method.

## Premium theme licenses

These two variables have no key in `config.toml`. Release builds contain the organization ID, so they do not read `NETRONOME__POLAR_ORG_ID`.

| Variable | Default | Description |
|---|---|---|
| `NETRONOME__POLAR_ORG_ID` | empty | The Polar organization ID for premium theme licenses. A build from source without this value turns premium themes off. |
| `NETRONOME__POLAR_ENVIRONMENT` | empty (production) | The Polar API to use: `production`, `sandbox`, or `development`. |
