# Netronome Domain Context

The vocabulary of this codebase. When a word here has a definition, use that word
with that meaning, in code, in issues, and in commit messages.

## The two halves

**Server** (or **instance**) — the process that `netronome serve` starts. It holds
the database, serves the web interface, runs the scheduler, and polls the agents.
There is one server in a deployment.

**Agent** — the process that `netronome agent` starts on another machine. It is
the same binary with a different subcommand. It reports the interface bandwidth,
the system metrics, the disks, and the SMART data. It has no database. It keeps
no schedule. The server asks; the agent answers.

> **Careful:** _server_ also means a **speed test server**, which is a remote
> endpoint that a test measures against (an Ookla server, a LibreSpeed server, an
> iperf3 server). The two meanings sit next to each other in the speed test code.
> Write _speed test server_ or _instance_ when the context does not make the
> meaning clear.

## Speed tests

**Provider** — the implementation that performs a test: `speedtest` (speedtest.net
through the `showwin/speedtest-go` library), `librespeed` (the `librespeed-cli`
subprocess), or `iperf3`. Each provider is a runner behind one interface.
`TestType` on a result holds the provider name.

**Test options** — what one test measures: the download, the upload, the ping, the
jitter, the packet loss. A test can enable any combination.

**Schedule** — an interval, a set of speed test server IDs, and a set of test
options. The scheduler runs the test on the server instance. An agent cannot hold
a schedule. See issue #98.

## Monitors

The word **monitor** has two meanings in this codebase. Do not mix them.

**Packet loss monitor** (`PacketLossMonitor`) — a host, an interval, and a
threshold. The server pings the host and records the loss. Its states are the
normal state, `down` (100% loss), and `recovered`.

**Agent monitoring** (`internal/monitor`) — the client of the server that polls the
agents and stores their bandwidth and their system metrics. `MonitorAgent` is an
agent as the server stores it, not a monitor in the packet loss sense.

**Bandwidth** — the interface counters that an agent reads with `vnstat`. This is
the traffic that the machine passes. It is not a speed test.

**Agent client** — the module in `internal/monitor` through which the server
fetches from the HTTP endpoints of an agent. It owns the base URL rule, the API
key header, the shared HTTP client, and the timeouts. The proxy handlers are its
callers, and the poller becomes a caller later. The live-data SSE stream is not
part of it.

**DNS monitor** (`DNSMonitor`) — a resolver, an interval, a query, and a protocol
(UDP, TCP, or DoT). The server sends the query to the resolver and records the
response time and the response code. A check fails on a timeout or an error
response code. Its states are the normal state, `down`, and `recovered`. The DNS
monitor measures the resolver, not the record. It does not check that an answer
is correct.

## Notifications

Four tables, and the relation between them is the design:

- **Channel** — a destination. One shoutrrr URL, one name, enabled or not.
- **Event** — a thing that can happen, in a category (`speedtest`, `packetloss`,
  `agent`) with a type (`complete`, `download_low`, `monitor_down`, `cpu_high`,
  and so on). The events are seeded rows. A user does not create them.
- **Rule** — one channel joined to one event. This is where a user makes a
  decision: enabled or not, the threshold value, the threshold operator. A rule
  is the unit of configuration, so anything a user sets for one destination and
  one event belongs on the rule.
- **History** — what the server sent, and whether it succeeded.

## Other terms

**Tailscale node** — Netronome can join a tailnet with `tsnet`, as the server or as
an agent, and can find the agents on the tailnet. This is not the same as running
next to the Tailscale daemon of the host.

**Threshold** — a number that an event compares against, with an operator.
A speed below a threshold, a temperature above one. Not every event has one;
`SupportsThreshold` on the event says which.
