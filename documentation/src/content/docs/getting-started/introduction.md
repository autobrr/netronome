---
title: Introduction
description: What Netronome is, what it does, and how its parts work together.
sidebar:
  order: 1
---

Netronome is a network performance testing and monitoring tool with a web interface. It runs speed tests, traces routes, watches for packet loss and DNS failures, and collects system data from other servers. It keeps the results in a database, so you can see how your network changes over time.

Netronome is one binary. The binary contains the server, the web interface, and the agent. You can use it for one home connection or for servers at many sites.

## Features

### Speed tests

- Speedtest.net tests need no extra tool.
- iperf3 tests run against iperf3 servers that you add in the web interface.
- LibreSpeed tests run against the public LibreSpeed servers, or against your own servers in a `librespeed-servers.json` file.
- The web interface shows the progress of a test while it runs, and a chart of all past results.
- The scheduler runs tests at an interval (for example, every hour) or at exact times of day. It adds a small random delay to each run so that tests with the same schedule do not start at the same moment.

See [Speed tests](/configuration/speed-tests/) and [Scheduling](/configuration/scheduling/).

### Network diagnostics

- Traceroute shows each hop between Netronome and a host.
- Packet loss monitors ping a host on a schedule and record the loss. If you install `mtr`, they also record the loss at each hop.
- With GeoIP databases, hops show a country flag and the network (ASN) that owns the address.

See [Packet loss](/monitoring/packet-loss/) and [GeoIP](/configuration/geoip/).

### DNS monitors

A DNS monitor sends one DNS query to a resolver on a schedule. You choose the resolver, the name to look up, the record type (`A`, `AAAA`, `CNAME`, `MX`, `NS`, or `TXT`), and the protocol (UDP, TCP, or DNS over TLS). Netronome records the response time and the response code. It sends a notification when a resolver stops answering and when it recovers.

See [DNS monitors](/monitoring/dns/).

### Agents

An agent is Netronome running in agent mode (`netronome agent`) on another server. The server connects to the agent and shows this data on the Agents tab:

- CPU, memory, and disk use.
- Temperatures from CPU, NVMe, and other sensors.
- Bandwidth use from `vnstat`.

The server can find agents on your Tailscale network by itself. See [Agents](/monitoring/agents/) and [Tailscale](/monitoring/tailscale/).

### Notifications

Netronome sends notifications through Shoutrrr. Supported services include Discord, Telegram, Slack, email, Pushover, and ntfy. You choose which events send a notification, for example:

- A speed test fails, or its download or upload speed is below a limit.
- Packet loss goes above a limit, or a monitored host goes down.
- An agent goes offline, or its CPU, memory, disk, or temperature goes past a limit.

See [Notifications](/configuration/notifications/).

### Web interface

- Light, dark, and system appearance modes.
- A default theme, plus a set of premium themes that a one-time license unlocks. You activate the license in Settings > Themes & License.
- A public dashboard at `/public` that shows your speed test history without a login. The share button on the dashboard gives you the link.
- A purge function in Settings > Data that deletes old speed test and packet loss results.
- An update check. Netronome looks for a new release every two hours and shows a notice in the web interface when one is available. You can turn it off with `check_for_updates = false`.

### Authentication

Netronome has a built-in user login, OpenID Connect (OIDC) login, and an IP whitelist. See [Authentication](/configuration/authentication/).

## How it works

Netronome has three parts:

1. The server. You start it with `netronome serve`. It runs the web interface and the API on port `7575`, runs the scheduled tests and monitors, and sends the notifications.
2. The agents. You start one with `netronome agent` on each server that you want to monitor. An agent listens on port `8200`. The server connects to each agent and reads its live system and bandwidth data. Agents are optional.
3. The database. The server keeps users, results, schedules, and settings in SQLite by default. SQLite needs no setup. For larger installations, you can use PostgreSQL. See [Database](/configuration/database/).

Some features run external programs, for example `iperf3`, `librespeed-cli`, `mtr`, and `vnstat`. See [External tools](/getting-started/installation/#external-tools).
