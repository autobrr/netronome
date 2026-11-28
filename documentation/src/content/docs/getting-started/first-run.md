---
title: First run
description: Start Netronome, create your user, run your first speed test, and set it up for common use cases.
sidebar:
  order: 3
---

This page starts after [Installation](/getting-started/installation/). You have the `netronome` binary and a configuration file from `netronome generate-config`.

## Start the server

If you did not install a service, start the server by hand:

```bash
netronome serve
```

Without `--config`, Netronome looks for a configuration file in this order:

1. `~/.config/netronome/config.toml`
2. `netronome/config.toml` in the configuration folder of your operating system
3. `config.toml` in the current folder

If Netronome finds no file, it starts with the default values. The SQLite database file `netronome.db` goes in the same folder as the configuration file.

By default, the server listens on `127.0.0.1:7575`. Only the same computer can open it.

## Open the web interface from other devices

To open Netronome from other devices on your network, set the listen address to `0.0.0.0` in `config.toml`:

```toml title="config.toml"
[server]
host = "0.0.0.0"
```

You can also set the environment variable `NETRONOME__HOST=0.0.0.0`. Restart Netronome after the change.

To run Netronome behind nginx or another reverse proxy, see [Reverse proxy](/configuration/reverse-proxy/).

## Create your user

Netronome has no default user name or password.

1. Open `http://localhost:7575` in your browser.
2. On the registration page, enter a user name and a password. The password must have at least 4 characters.
3. Netronome opens the login page. Log in with the new user.

The registration page works only while the database has no user. After you create the first user, registration closes.

To add a user or to change a password, use the command line on the server:

```bash
netronome create-user <username>
netronome change-password <username>
```

For password input from a script, OIDC login, and the IP whitelist, see [Authentication](/configuration/authentication/).

## Run your first speed test

1. Open the Speed Test tab.
2. Open Server Selection.
3. Select the test type: Speedtest, iperf3, or Librespeed.
4. Select a server from the list. For iperf3, add your iperf3 server first.
5. Click Run.

The page shows the progress of the test while it runs. After the test, the result goes into the history on the Dashboard tab.

To run tests on a schedule, add a schedule in the Schedule Manager on the Speed Test tab. See [Scheduling](/configuration/scheduling/).

## Common use cases

### Home network

Monitor the quality of your internet connection:

1. Schedule a speed test every hour. See [Scheduling](/configuration/scheduling/).
2. Add a packet loss monitor to a public address, for example `1.1.1.1` or `8.8.8.8`. See [Packet loss](/monitoring/packet-loss/).
3. Add a DNS monitor for the resolver that your network uses. See [DNS monitors](/monitoring/dns/).
4. Add a notification for slow speed tests and high packet loss. See [Notifications](/configuration/notifications/).

### Multiple sites

Monitor the connections between offices or data centers:

1. Install an agent at each site. See [Agents](/monitoring/agents/).
2. Run an iperf3 server at each site, and add the servers to Netronome for iperf3 tests between sites.
3. Connect the server and the agents through Tailscale, so that no agent port is open to the internet. See [Tailscale](/monitoring/tailscale/).
4. Add notifications for agents that go offline and for slow speed tests.

### Server health

Watch the resources of your servers:

1. Install an agent on each server. See [Agents](/monitoring/agents/).
2. Watch CPU, memory, disk use, and temperatures on the Agents tab.
3. Add notifications for high CPU, high memory, low disk space, and high temperature.
4. Install `vnstat` on each server to record bandwidth use over time.
