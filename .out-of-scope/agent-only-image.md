# Agent-only Docker image

Netronome ships one container image. There is no second, smaller image that
contains only the agent.

## Why this is out of scope

The agent is not a separate program. It is a subcommand of the same binary:

```go
// cmd/netronome/main.go
Use:   "agent",
```

The published image can already run it. The compose service only needs a
different command:

```yaml
services:
  netronome-agent:
    image: ghcr.io/autobrr/netronome:latest
    command: agent
    network_mode: host
```

A second image would give a smaller download, because the agent does not need
`iperf3`, `mtr`, `traceroute`, `sqlite`, `librespeed-cli`, or the web assets.
The cost is a second build matrix, a second set of tags, and a second release
path to keep correct for every architecture, forever. The size that this saves
is not worth that maintenance for the devices in question, which must run
`vnstat` in any case.

If the image size becomes a true problem on a specific device, open an issue
with the device and the disk space that is available. Numbers change the
decision.

## Prior requests

- #141: "Separate Agent docker container"
