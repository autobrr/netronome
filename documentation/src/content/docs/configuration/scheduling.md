---
title: Scheduling
description: Run speed tests on a fixed interval or at exact times of the day.
sidebar:
  order: 5
---

A schedule runs a speed test at set times. You create schedules in the Schedule Manager on the Speed Test tab of the web interface. Schedules have no keys in `config.toml`.

A schedule uses one of two types: an interval or exact times.

## Interval

An interval schedule runs the test again after a fixed time. The web interface offers these intervals:

| Label in the web interface | Value |
|---|---|
| Every 5 Minutes | `5m` |
| Every 15 Minutes | `15m` |
| Every 30 Minutes | `30m` |
| Every Hour | `1h` |
| Every 6 Hours | `6h` |
| Every 12 Hours | `12h` |
| Every Day | `24h` |
| Every Week | `7d` |

Netronome accepts Go duration strings, for example `30s`, `5m`, or `1h`. It also accepts `d` for days and `w` for weeks, for example `7d` or `2w`.

Netronome adds a random delay of 1 to 300 seconds to each run. With the delay, two schedules with the same interval start at different times in most cases.

## Exact times

An exact-time schedule runs the test each day at the times that you select. The value starts with `exact:`, then has one or more times in `HH:MM` format with commas between them:

```text
exact:14:30          # Each day at 14:30 UTC
exact:00:00,12:00    # Each day at 00:00 and 12:00 UTC
```

The web interface offers full hours, from 00:00 to 23:00. Netronome stores the times in UTC. The web interface converts the times that you select from your time zone to UTC. To change your time zone, open Settings, then Time & Timezone.

Netronome adds a random delay of 1 to 60 seconds to each run.

## How runs start

- Netronome checks for due schedules one time each minute. A run can thus start up to one minute after its time, plus the random delay.
- After a restart, Netronome does not run the tests that it missed. It calculates the next run from the current time.
- For the time limit of a scheduled test, see [Speed tests](/configuration/speed-tests/#scheduled-tests).
- Packet loss monitors and DNS monitors use the same interval formats, but without the random delay.
