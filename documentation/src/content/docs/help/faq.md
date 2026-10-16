---
title: FAQ and troubleshooting
description: Answers to common questions and fixes for common problems.
sidebar:
  order: 1
---

## Getting started

### What is the default user name and password?

Netronome has no default user. On the first visit, Netronome shows a registration page, and you create the first user there. See [First run](/getting-started/first-run/#create-your-user).

### I forgot my password. How do I reset it?

Run this command on the server, with the same `--config` that the server uses:

```bash
netronome change-password <username>
```

### How do I open Netronome from other devices on my network?

Set the listen address to `0.0.0.0`. In `config.toml`, set `host = "0.0.0.0"` in the `[server]` section. You can also set `NETRONOME__HOST=0.0.0.0`. The default is `127.0.0.1`, which accepts connections from the same computer only.

### Do I need all the external tools?

No. Speedtest.net tests, DNS monitors, and packet loss monitors work without external tools. You need:

- `iperf3` for iperf3 tests.
- `librespeed-cli` for LibreSpeed tests.
- `traceroute` for the Traceroute tab.
- `mtr` for packet loss at each hop.
- `vnstat` on agents for bandwidth data.

See [External tools](/getting-started/installation/#external-tools).

### How do I stop the update notice?

Set `check_for_updates = false` at the top level of `config.toml`, or set `NETRONOME__CHECK_FOR_UPDATES=false`. Development builds do not check for updates.

## Common problems

### I cannot open the web interface of the Docker container

The container has no configuration file, so the server listens on `127.0.0.1` inside the container. Create a configuration file with `generate-config` in the container, or set `NETRONOME__HOST=0.0.0.0`. See the Docker steps in [Installation](/getting-started/installation/).

### The server stops with "address already in use"

Another program uses port `7575`. Stop that program, or give Netronome a different port with `port` in the `[server]` section or with `NETRONOME__PORT`.

### A test fails with "librespeed-cli not found" or "iperf3 not found"

The program is not in the `PATH` of the user that runs Netronome. Install it, and restart Netronome if you changed `PATH`. See [External tools](/getting-started/installation/#external-tools).

### The LibreSpeed server list is empty

Netronome gets the public server list from LibreSpeed.org. If the list does not load, make sure that the server can connect to the internet. You can also add your own servers in `librespeed-servers.json` in the same folder as `config.toml`. See [Speed tests](/configuration/speed-tests/).

### Speed tests are slower than expected

1. Test to a server that is near you.
2. Make sure that other devices do not use the connection during the test.
3. Try a different test type. Some ISPs slow down some test services.
4. For iperf3, make sure that the iperf3 server can send and receive at the speed of your connection.

For the number of connections and other test settings, see [Speed tests](/configuration/speed-tests/).

## Temperatures

### Why do I not see disk temperatures?

CPU and NVMe temperatures work in all builds. SATA and HDD temperatures need SMART data. The binaries on the Releases page have no SMART support, and the agent must run as root. For all the conditions, see [Temperature monitoring](/monitoring/agents/#temperature-monitoring).

## Network diagnostics

### Why does a hop show 100% packet loss when the total loss is 0%?

Many routers drop or limit ICMP replies to themselves, but they still forward traffic to the next hop. The loss at the last hop is the real loss between you and the host. See [Packet loss](/monitoring/packet-loss/#0-packet-loss-with-lost-hops).

### Packet loss monitors do not use MTR

Netronome uses MTR only when it finds `mtr` in the `PATH`. Without it, monitors use ping and record only the total loss. On Windows, the WinMTRCmd file must have the name `mtr.exe`. In Docker, MTR needs the `NET_RAW` capability. See [Packet loss](/monitoring/packet-loss/).

## Agents

### Netronome does not find my Tailscale agent

1. Make sure that the agent listens on the discovery port. The default is `8200`.
2. Make sure that the server and the agent are on the same tailnet.
3. From the server, run `tailscale ping <agent-hostname>`.
4. From the server, run `curl http://<agent-hostname>:8200/netronome/info`. The agent replies with information about itself.

See [Tailscale](/monitoring/tailscale/).

## Themes

### Why are the premium themes missing?

A build from source has the default theme only. The premium themes are in the release binaries and the Docker image, and a one-time license unlocks them in Settings > Themes & License. See [Build from source](/help/build-from-source/#premium-themes-in-source-builds).
