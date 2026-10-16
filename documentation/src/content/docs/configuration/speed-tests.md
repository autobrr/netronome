---
title: Speed tests
description: Configure the Speedtest.net, iperf3, and LibreSpeed tests, their time limits, and their connections.
sidebar:
  order: 4
---

Netronome runs speed tests with three providers:

- Speedtest.net, built into Netronome.
- iperf3, against an iperf3 server that you add. This needs the `iperf3` program on the host that runs Netronome.
- LibreSpeed, against public or custom LibreSpeed servers. This needs the `librespeed-cli` program on the host that runs Netronome.

You select the provider and the server in the web interface. The keys on this page go in `config.toml` or in environment variables.

## Speedtest.net

- `speedtest.timeout` (default `30`): The time limit in seconds for the download test, and a separate time limit for the upload test. For a test that you start in the web interface, it is also the time limit for a full iperf3 test.
- `speedtest.connections` (default `0`): The maximum number of parallel connections for the download test and the upload test.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

With `connections = 0`, the test library chooses:

- Download: one connection for each CPU core.
- Upload: one connection for each CPU core, with a maximum of 8.

A higher value, for example `32`, can give more accurate results on links faster than 1 Gbit/s.

```toml title="config.toml"
[speedtest]
timeout = 30
connections = 32
```

:::caution
Do not set a high `connections` value on a slow upload link. The upload test starts all connections at the same time. On a 10 Mbit/s upload link, 32 connections give an upload result of 0 Mbit/s.
:::

### Servers

The server list shows the Speedtest.net servers, nearest first. Netronome keeps the list for 30 minutes. If you do not select a server, Netronome uses the nearest one.

To use a server that is not in the list, type its ID in the Server ID field and add it. Netronome then asks Speedtest.net for the server with that ID.

Some servers redirect their HTTP address to HTTPS. Before the test starts, Netronome follows the redirects and uses the final address for the full test.

## iperf3

Add an iperf3 server in the web interface with a name, a host, and a port (5201 is the iperf3 default). Before each iperf3 test, Netronome runs `ping` against the server to measure the latency.

- `speedtest.iperf.test_duration` (default `10`): The length of the iperf3 test, in seconds.
- `speedtest.iperf.parallel_conns` (default `4`): The number of parallel TCP connections.
- `speedtest.iperf.timeout` (default `60`): The time limit for the `iperf3` program, in seconds.
- `speedtest.iperf.ping.count` (default `5`): The number of ping packets.
- `speedtest.iperf.ping.interval` (default `1000`): The time between ping packets, in milliseconds.
- `speedtest.iperf.ping.timeout` (default `10`): The time limit for the ping step, in seconds.

If the ping step fails, the iperf3 test continues without a latency value.

```toml title="config.toml"
[speedtest.iperf]
test_duration = 10
parallel_conns = 4
timeout = 60

[speedtest.iperf.ping]
count = 5
interval = 1000
timeout = 10
```

:::note
A test that you start in the web interface also has the `speedtest.timeout` limit, which is 30 seconds by default. If you increase `test_duration`, increase `speedtest.timeout` too.
:::

## LibreSpeed

- `speedtest.librespeed.timeout` (default `60`): The time limit in seconds for a LibreSpeed test that you start in the web interface. If the value in `config.toml` is `0`, Netronome uses 60.

The server list contains two groups:

- Public servers from `librespeed.org`. Netronome keeps this list for 30 minutes.
- Custom servers from the file `librespeed-servers.json`, in the same directory as `config.toml`. If the file does not exist, the list has no custom servers.

The custom file is a JSON array in the `librespeed-cli` server format:

```json title="librespeed-servers.json"
[
  {
    "id": 1,
    "name": "Clouvider - London, UK",
    "server": "http://lon.speedtest.clouvider.net/backend",
    "dlURL": "garbage.php",
    "ulURL": "empty.php",
    "pingURL": "empty.php",
    "getIpURL": "getIP.php"
  }
]
```

## Scheduled tests

A scheduled test does not use `speedtest.timeout` or `speedtest.librespeed.timeout` for the full test. It has a fixed time limit of 5 minutes. The other time limits on this page still apply. To set up schedules, see [Scheduling](/configuration/scheduling/).
