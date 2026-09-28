---
title: Packet loss monitoring
description: Test the packet loss and the round-trip time to a host on a schedule, with MTR or ping.
sidebar:
  order: 4
---

A packet loss monitor sends probes to a host on a schedule. It records the packet loss and the round-trip time (RTT), which is the time a probe needs to go to the host and come back. If the Netronome server has MTR, the monitor also records each hop on the route.

You manage packet loss monitors on the **Traceroute** tab of the web interface.

## Create a monitor

1. Open the **Traceroute** tab, and click **Monitors**.
2. Click **Add**.
3. In **Host**, type a host name or an IP address, for example `8.8.8.8`.
4. In **Name**, type a name for the monitor.
5. In **Schedule Type**, select how the monitor runs:
   - Select **Interval** to run the monitor at a fixed interval. In **Check Interval**, select an interval from 1 minute to 24 hours. The default is 30 minutes.
   - Select **Exact Time** to run the monitor each day at fixed times. In **Test Times**, select one or more full hours.
6. In **Packets per Test**, type the number of probes for each test, from 1 to 100. The default is 10.
7. In **Alert Threshold (% packet loss)**, type the loss that gives an alert. The default is 5.
8. Click **Create Monitor**.

The scheduler starts due tests one time per minute. For more information about schedules, see [Scheduling](/configuration/scheduling/).

You can also create a monitor from a traceroute result. Click **Monitor This Host** below the result.

In the monitor list, **Stop Monitor** turns a monitor off. **Start Monitor** turns it on again and runs a test at once.

## Test methods

Netronome uses MTR when the `mtr` command is in the `PATH`. If not, it uses ping.

### MTR

MTR sends probes to each hop on the route to the host. Netronome runs it with these settings:

- IPv4 only.
- One probe cycle per second.
- The number of cycles is **Packets per Test**.

The packet loss of the test is the loss at the last hop, which is the host. Netronome also records the loss and the RTT of each hop. If you set up [GeoIP](/configuration/geoip/), each hop also shows its country and its network owner (ASN).

If MTR fails, Netronome runs the test again with ping.

### Ping

Ping sends ICMP echo requests to the host only, so the result has no hops. Each test has a time limit of 2 seconds per probe. If the time limit ends before Netronome gets a result, the test records 100% packet loss.

## Privileges

To send ICMP probes, a program needs raw sockets. On Linux, raw sockets need root or the `CAP_NET_RAW` capability.

With `privileged_mode = true`, which is the default, Netronome tries ICMP first:

- If MTR fails in ICMP mode, Netronome runs MTR again in UDP mode (`mtr -u`).
- If ping fails in ICMP mode, Netronome runs ping again in unprivileged mode.

With `privileged_mode = false`, Netronome uses UDP mode for MTR and unprivileged mode for ping from the start.

The results table shows `ICMP` or `UDP` next to each MTR result, so you can see which mode ran.

## Configuration

The `[packetloss]` section of the configuration file controls the service:

```toml title="config.toml"
[packetloss]
enabled = true
max_concurrent_monitors = 10
privileged_mode = true
mtr_enable_dns = false
```

- `enabled` (default `true`): Turns on packet loss monitoring. If `false`, the server does not start the service or its API.
- `max_concurrent_monitors` (default `10`): A limit for monitors that run at the same time. The current version reads this value, but it does not apply the limit to scheduled tests or to **Start Monitor**.
- `privileged_mode` (default `true`): Tries ICMP first. See [Privileges](#privileges).
- `mtr_enable_dns` (default `false`): Shows host names instead of IP addresses for the hops.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

## Notifications

Netronome gives each monitor a state after each test:

- Down: The packet loss is 100%.
- Threshold exceeded: The packet loss is more than **Alert Threshold**, but less than 100%.
- OK: The packet loss is at or below **Alert Threshold**.

Netronome sends a notification only when the state changes. A change to Down or to Threshold exceeded gives an alert. A change from one of these states back to OK gives a recovery notification. To select the notifications, see [Notifications](/configuration/notifications/).

## 0% packet loss with lost hops

A result can show 0% packet loss for the host and 100% loss at a hop between. Many routers block ICMP replies to themselves, or send fewer of them, but they still forward the traffic. The loss at the last hop tells you whether packets get to the host.
