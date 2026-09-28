---
title: System monitoring agents
description: Install the Netronome agent on a server and see its bandwidth, CPU, memory, disks, and temperatures in the Netronome dashboard.
sidebar:
  order: 1
---

An agent is the `netronome` binary in agent mode. It runs on the server that you want to monitor and sends data to your Netronome server over HTTP. One Netronome server can monitor many agents.

The agent gets bandwidth data from [vnstat](https://humdi.net/vnstat/), and it gets system data from the operating system. You add each agent in the web interface on the **Agents** tab. To run the agent in a container, see [Docker agents](/monitoring/docker-agents/).

## Requirements

- Install vnstat on the server, and keep its daemon running. The agent runs the `vnstat` command for all bandwidth data. vnstat 1.x and 2.x both work.
- The Netronome server must be able to connect to the agent port, which is `8200` by default. If you cannot open a port, use [Tailscale](/monitoring/tailscale/).
- On the Netronome server, `[monitor]` must have `enabled = true`. This is the default.

## Install with the script

The install script downloads the latest release, writes a configuration file, and creates a service. It supports Linux and macOS on `x86_64` and `arm64`. On a 32-bit ARM host, install the agent by hand from the `linux_arm` release archive.

Install vnstat before you run the script. If vnstat is missing, the script stops.

On Linux, the script must run as root:

```bash
curl -sL https://netrono.me/install-agent | sudo bash
```

On macOS, run it as your user. The script asks for your password when it needs `sudo`:

```bash
curl -sL https://netrono.me/install-agent | bash
```

The script asks for these values:

1. The network interface to monitor. If you leave it empty, vnstat selects the interface. See [The interface setting](#the-interface-setting).
2. Tailscale, and the Tailscale method. See [Tailscale](/monitoring/tailscale/).
3. The API key. You can generate a random key, type your own key, or use no key. With Tailscale, the script does not ask for a key.
4. The host and the port to listen on. The defaults are `0.0.0.0` and `8200`.
5. The disk mounts to include and to exclude.
6. Automatic daily updates.

At the end, the script shows the URL and the API key of the agent. Keep them. You need them when you add the agent in Netronome.

If the script has no terminal for input, it uses these values: all defaults, no Tailscale, a generated API key, and automatic updates on.

### What the script installs

| Item | Linux | macOS |
|---|---|---|
| Binary | `/opt/netronome/netronome` | `/usr/local/opt/netronome/netronome` |
| Configuration file | `/etc/netronome/agent.toml` | `/usr/local/etc/netronome/agent.toml` |
| Service | systemd unit `netronome-agent` | launchd job `com.netronome.agent` |
| Service user | `netronome` (`root` in Tailscale tsnet mode) | `netronome` |

The configuration file has mode `600`, because it contains the API key.

On Linux, automatic updates use the systemd timer `netronome-agent-update.timer`. It runs once a day, with a random delay of up to 4 hours.

### Script options

- `-u`, `--uninstall`: Stops and removes the service, the update job, the binary, and the configuration.
- `--update`: Runs `netronome update` and restarts the service.
- `--auto-update [true|false]`: Turns automatic daily updates on or off without a prompt.
- `-h`, `--help`: Shows the help.

To give an option, download the script first:

```bash
curl -sL https://netrono.me/install-agent -o install-agent.sh
sudo bash install-agent.sh --update
```

## Set up the agent by hand

Start the agent with the `agent` command:

```bash
# Default settings: listen on 0.0.0.0:8200, no API key
netronome agent

# With an API key
netronome agent --api-key your-secret-key

# Custom address, port, and interface
netronome agent --host 192.168.1.100 --port 8300 --interface eth0

# With a configuration file
netronome agent --config /etc/netronome/agent.toml
```

A flag replaces the value from the configuration file. For all flags, see [CLI](/reference/cli/).

:::caution
Set an API key if other devices can connect to the agent port. Without a key, any device that can connect to the port can read the data of the agent.
:::

## Add the agent in Netronome

1. In the web interface, open the **Agents** tab.
2. Click **Add Agent**.
3. In **Agent Name**, type a name for the server.
4. In **Agent URL**, type the base URL of the agent, for example `http://192.168.1.100:8200`.
5. If the agent has an API key, type it in **API Key**.
6. Click **Create**.

If **Enable monitoring** is on, the server connects to the agent at once. To show the agent on the dashboard, click **Feature on dashboard** in the agent list.

## Agent configuration

The agent reads the `[agent]` section of the configuration file:

```toml title="config.toml"
[agent]
host = "0.0.0.0"
port = 8200
interface = ""
api_key = "your-secret-key"
disk_includes = ["/mnt/storage"]
disk_excludes = ["/boot", "/tmp"]
disable_system_metrics = false

[logging]
level = "info"
```

- `host` (flag `--host`, `-H`, default `0.0.0.0`): The IP address to listen on.
- `port` (flag `--port`, `-p`, default `8200`): The port to listen on.
- `interface` (flag `--interface`, `-i`, default empty): The interface for the live bandwidth data.
- `api_key` (flag `--api-key`, `-k`, default empty): The API key. If empty, the agent does not ask for a key.
- `disk_includes` (flag `--disk-include`, default empty): Mounts to always report.
- `disk_excludes` (flag `--disk-exclude`, default empty): Mounts to never report.
- `disable_system_metrics` (flag `--disable-system-metrics`, default `false`): Sends bandwidth data only.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

The environment variables for the disk lists take a comma-separated list, for example `/mnt/a,/mnt/b`. The `--disk-include` and `--disk-exclude` flags also take a comma-separated list, or you can give the flag more than one time.

To set the log level for one run, use `--log-level` (`-l`) with `trace`, `debug`, `info`, `warn`, or `error`.

### The interface setting

If `interface` has a value, the agent gives it to vnstat with `--iface`. The live bandwidth data and the history then come from that interface only.

If `interface` is empty, the agent does not give an interface to vnstat. The live data then comes from one interface: the `Interface` setting in `vnstat.conf`, or the interface that vnstat selects itself. The history contains all interfaces in the vnstat database.

### Disk filters

The agent reports each mounted file system that passes these rules, in this order:

1. If the mount matches a pattern in `disk_includes`, the agent reports it.
2. If the mount matches a pattern in `disk_excludes`, the agent does not report it.
3. The agent skips system mounts: mounts under `/snap`, `/run`, `/dev`, `/proc`, `/sys`, and the overlay directories of Docker and Podman. It also skips the file system types `tmpfs`, `devtmpfs`, `squashfs`, `overlay`, `proc`, `sysfs`, `cgroup`, and `cgroup2`.
4. The agent skips file systems smaller than 1 GiB.

A mount in `disk_includes` skips rules 2 to 4. The agent reports it even when it is small, special, or a bind mount.

A pattern can have one of three forms:

- An exact path, for example `/mnt/storage`.
- A path that ends in `*`. It matches each mount that starts with the text before the `*`, for example `/mnt/*`.
- A glob pattern with `*`, `?`, or `[]`. It matches the full mount path or the last part of the path.

If two or more mounts show the same device, file system type, and size, the agent reports only one of them. It keeps the mount that is not a bind mount, and then the mount with the shortest path. A bind mount in `disk_includes` stays in the report.

## What the agent reports

### Bandwidth

- Live download and upload speed, from `vnstat --live`.
- The peak download and upload speed since the agent started.
- The traffic history from the vnstat database, by hour, day, and month.

The Netronome server gets the live data as a stream. It gets the traffic history one time each hour.

### System information

- The host name, the kernel version, and the uptime. The uptime is available on Linux and macOS.
- The vnstat version.
- Each network interface that is not a loopback interface: the name, the IPv4 address, and the up or down state. On Linux, the agent also reads the link speed. It marks bridge, Docker, `veth`, `tap`, `tun`, and bond interfaces as virtual and shows no link speed for them.
- The vnstat alias and the total traffic of each interface.

### Hardware

- CPU: the model, the frequency, the physical cores, the threads, the usage, and the load average. The load average is not available on Windows.
- Memory: the total, used, free, available, cached, and buffer memory, and the swap. On Linux, used memory is the total minus the available memory, so the cache does not count as used. On a ZFS system, the agent also reports the size of the ZFS ARC.
- Disks: the path, the device, the file system type, the size, and the used space of each disk that passes the [disk filters](#disk-filters). If the agent can read SMART data, it adds the model and the serial number.
- Temperatures: the sensors of the operating system, and the disk temperatures from SMART.

The Netronome server gets the system information and the hardware data every 30 seconds. It keeps these samples for 2 hours.

### Bandwidth-only mode

With `disable_system_metrics = true`, the agent sends bandwidth data only. It does not collect CPU, memory, disk, or temperature data. The Netronome server detects this and does not ask the agent for the data.

## Temperature monitoring

The agent reads the CPU and board sensors that the operating system shows. It skips readings of 0 °C or less and of more than 200 °C.

Disk temperatures come from SMART. The agent reads SMART data only in these conditions:

- The operating system is Linux or macOS. On Linux, the agent reads SATA and NVMe disks. On macOS, it reads NVMe disks.
- The agent runs as root. SMART needs direct access to the disk device.
- The disk has a temperature sensor.
- The binary contains SMART support. A binary built with the `nosmart` tag has no SMART support. The binaries on the Releases page use this tag. The Docker image and `make build` binaries have SMART support. See [Build from source](/help/build-from-source/).

:::note
The install script runs the agent as the `netronome` user. The only exception is Tailscale tsnet mode on Linux, where the agent runs as root. The `netronome` user cannot read SMART data, so the agent shows no disk temperatures. To get them, run the agent as root.
:::

## Connections and alerts

If the connection to an agent stops, the server tries again. It waits 1 second before the first try and doubles the time after each failure, up to 1 minute.

The server can send a notification when an agent goes offline or comes back online. It can also send an alert when the CPU, memory, disk use, temperature, or bandwidth of an agent goes above a limit. The server sends each of these alerts at most one time per hour for each agent. To set the limits, see [Notifications](/configuration/notifications/).

## Agent endpoints

You can open these URLs in a browser or with `curl` to test an agent.

These paths do not need an API key:

- `/`: The service name, the endpoints, and whether a key is necessary.
- `/netronome/info`: The agent version, the host name, and whether Tailscale is on. Discovery uses it.

These paths need the API key, if the agent has one:

- `/events?stream=live-data`: The live bandwidth stream.
- `/export/historical`: The vnstat history as JSON. Add `?interface=<name>` for one interface.
- `/stats/peaks`: The peak speeds.
- `/system/info`: The system information. Not available in bandwidth-only mode.
- `/system/hardware`: The hardware data. Not available in bandwidth-only mode.
- `/tailscale/status`: The Tailscale status of the agent.

Send the API key in the `X-API-Key` header or in the `apikey` query parameter:

```bash
curl -H "X-API-Key: your-secret-key" http://192.168.1.100:8200/stats/peaks
```
