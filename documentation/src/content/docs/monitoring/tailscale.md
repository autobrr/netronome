---
title: Tailscale
description: Connect agents to Netronome over Tailscale, and let the server find new agents on your tailnet.
sidebar:
  order: 3
---

With Tailscale, the Netronome server connects to agents over your tailnet. A tailnet is your private Tailscale network. You do not have to open the agent port to other networks.

Tailscale has two uses in Netronome:

- An agent can listen on the tailnet.
- The server can find agents on the tailnet and add them. This is discovery.

## Methods

Netronome can use Tailscale in two ways:

- `host`: Netronome uses the `tailscaled` daemon that already runs on the machine. No new machine appears in the Tailscale admin console.
- `tsnet`: Netronome starts its own Tailscale node in the process. The node is a new machine on your tailnet. This method needs an auth key.

The `method` setting selects one of them:

- `auto`: The default. If `auth_key` has a value, Netronome uses `tsnet`. If not, it uses `host`.
- `host`: Netronome uses the `tailscaled` daemon on the machine.
- `tsnet`: Netronome starts its own node. If `auth_key` is empty, the agent stops with an error, and the server logs an error and does not run discovery.

## Set up an agent

To use Tailscale, start the agent with the `--tailscale` flag:

```bash
# Own Tailscale node: with an auth key, auto selects tsnet
netronome agent --tailscale --tailscale-auth-key tskey-auth-YOUR-KEY

# The tailscaled daemon of the machine
netronome agent --tailscale --tailscale-method host

# Own node with a custom host name
netronome agent --tailscale --tailscale-auth-key tskey-auth-YOUR-KEY --tailscale-hostname "webserver-prod"
```

- `--tailscale` (default `false`): Turns on Tailscale for the agent.
- `--tailscale-method` (default `auto`): `auto`, `host`, or `tsnet`.
- `--tailscale-auth-key` (default empty): The auth key for `tsnet`.
- `--tailscale-hostname` (default `netronome-agent-<hostname>`): The machine name of the `tsnet` node.
- `--tailscale-state-dir` (default `~/.config/netronome/tsnet`): The directory for the state of the `tsnet` node.

Instead of the `--tailscale` flag, you can set `enabled = true` in both `[tailscale]` and `[tailscale.agent]` of the configuration file of the agent:

```toml title="config.toml"
[tailscale]
enabled = true
method = "tsnet"
auth_key = "tskey-auth-YOUR-KEY"

[tailscale.agent]
enabled = true
```

:::note
`enabled = true` in `[tailscale]` alone does not turn on Tailscale for the agent. Use the `--tailscale` flag or `[tailscale.agent]`.
:::

### tsnet mode

The agent starts its own node with the machine name `netronome-agent-<hostname>`, or with the name from `--tailscale-hostname`. It keeps the node state in `state_dir`. The agent listens only on the tailnet, on port `agent_port` of the `[tailscale]` section. The default is `8200`. If `agent_port` is `0`, the agent uses `port` of the `[agent]` section.

With `ephemeral = true`, Tailscale removes the node when it goes offline. To use a Headscale server, set `control_url`.

On Linux, the install script runs the agent as root in this mode. The node state is then in `/root/.config/netronome/tsnet`.

### host mode

The agent uses the `tailscaled` daemon of the machine. The machine name on the tailnet is the name that `tailscaled` already uses. The `--tailscale-hostname` flag has no effect in this mode.

The agent listens on the first Tailscale IP address of the machine, on port `agent_port`. If it cannot use that address, it logs a warning and listens on `host` of the `[agent]` section.

If the agent cannot connect to `tailscaled`, it stops with the error `no running tailscaled found on host`.

### API keys

The Netronome server adds a discovered agent without an API key, and the web interface does not let you set one for it. If the agent has an API key, the server cannot read its data. For an agent that you want the server to discover, leave `api_key` empty. The install script does not set an API key when you select Tailscale.

## Set up the server

Discovery runs on the Netronome server. It needs `enabled = true` in `[tailscale]`, and `enabled = true` in `[monitor]`:

```toml title="config.toml"
[tailscale]
enabled = true
method = "auto"  # auto, host, or tsnet
auth_key = ""    # Required for tsnet mode
hostname = ""    # Optional custom hostname

# Discovery settings
auto_discover = true
discovery_interval = "5m"
discovery_port = 8200
```

- `enabled` (default `false`): Turns on Tailscale.
- `method` (default `auto`): `auto`, `host`, or `tsnet`.
- `auth_key` (default empty): The auth key for `tsnet`.
- `hostname` (default empty): The machine name of the `tsnet` node. If empty, the server uses `netronome-server-<hostname>`.
- `ephemeral` (default `false`): Tailscale removes the `tsnet` node when it goes offline.
- `state_dir` (default `~/.config/netronome/tsnet`): The directory for the state of the `tsnet` node.
- `control_url` (default empty): The URL of a Headscale server.
- `agent_port` (default `8200`): The port that the agent listens on in the tailnet. Only the agent reads this key.
- `auto_discover` (default `true`): Turns on discovery.
- `discovery_interval` (default `5m`): The time between two discovery runs.
- `discovery_port` (default `8200`): The port where discovery looks for agents.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

### The discovery run

The server runs discovery one time when it starts, and then one time per `discovery_interval`. If the interval is not a valid duration, the server logs a warning and uses 5 minutes.

On each run, the server does these steps:

1. It gets the list of online machines on the tailnet, and it adds itself.
2. It skips each machine that it already monitors as a Tailscale agent.
3. It sends `GET http://<machine-name>:<discovery_port>/netronome/info` to each other machine. The time limit is 3 seconds.
4. If the answer shows a Netronome agent with Tailscale on, the server adds the agent and starts to monitor it.

The server gives the new agent the machine name as its name. The web interface shows the agent as "Tailscale Connected Agent", with the date of discovery. For a discovered agent, you can only turn monitoring on or off.

If the server uses `host` and cannot connect to `tailscaled`, it logs a warning and does not run discovery. The server itself continues to run.

## Troubleshooting

If the server does not discover an agent:

- Make sure that the agent listens on the discovery port, which is `8200` by default.
- Make sure that you started the agent with Tailscale. The agent must show `"using_tailscale": true` on `/netronome/info`.
- Make sure that the server and the agent are on the same tailnet.
- Test the connection from the server: `tailscale ping <agent-hostname>`.
- Test the endpoint from the server: `curl http://<agent-hostname>:8200/netronome/info`.
- Read the server log for lines about Tailscale discovery.

## Docker with a Tailscale sidecar

A sidecar is a second container that gives a service to the main container. In this setup, a Tailscale container runs `tailscaled`, and Netronome uses it with the `host` method. Netronome and Tailscale stay in separate containers.

### Compose file

```yaml title="docker-compose.yml"
services:
  netronome:
    image: ghcr.io/autobrr/netronome:latest
    container_name: netronome
    user: 1000:1000
    restart: unless-stopped
    env_file: .env
    volumes:
      - "./netronome:/data"
      - tailscale-socket:/var/run/tailscale  # Share socket directory
    depends_on:
      - netronome-ts
    network_mode: service:netronome-ts  # Share network namespace

  netronome-ts:
    image: tailscale/tailscale:latest
    container_name: netronome-ts
    hostname: netronome  # This will be the Tailscale hostname
    cap_add:
      - NET_ADMIN
    environment:
      - TS_AUTHKEY=${TS_AUTHKEY}
      - TS_STATE_DIR=/var/lib/tailscale
      - TS_EXTRA_ARGS=${TS_EXTRA_ARGS}
      - TZ=${TZ}
      - TS_SERVE_CONFIG=/config/netronome.json
      - TS_SOCKET=/var/run/tailscale/tailscaled.sock  # Force socket location
    volumes:
      - /dev/net/tun:/dev/net/tun
      - ./config:/config
      - tailscale-data-netronome:/var/lib/tailscale
      - tailscale-socket:/var/run/tailscale  # Share socket directory

volumes:
  tailscale-data-netronome:
  tailscale-socket:  # Named volume for socket sharing
```

The setup depends on three parts of this file:

- `network_mode: service:netronome-ts` puts Netronome in the network namespace of the Tailscale container. Netronome then uses the Tailscale interface.
- The `tailscale-socket` volume is in both containers at `/var/run/tailscale`. Netronome connects to `tailscaled` through this socket.
- `TS_SOCKET=/var/run/tailscale/tailscaled.sock` makes the Tailscale container create the socket at that path.

:::caution
The shared socket gives Netronome full access to the Tailscale daemon. Share the socket only with containers that you trust.
:::

To run more than one Tailscale container, you can put the shared settings in a YAML anchor:

```yaml title="docker-compose.yml"
x-tailscale-base: &tailscale-base
  image: tailscale/tailscale:latest
  cap_add:
    - NET_ADMIN
  restart: unless-stopped
  networks:
    - tailscale_network

services:
  netronome:
    image: ghcr.io/autobrr/netronome:latest
    container_name: netronome
    user: 1000:1000
    restart: unless-stopped
    env_file: .env
    volumes:
      - "./netronome:/data"
      - tailscale-socket:/var/run/tailscale
    depends_on:
      - netronome-ts
    network_mode: service:netronome-ts

  netronome-ts:
    <<: *tailscale-base
    container_name: netronome-ts
    hostname: netronome
    environment:
      - TS_AUTHKEY=${TS_AUTHKEY}
      - TS_STATE_DIR=${TS_STATE_DIR}
      - TS_EXTRA_ARGS=${TS_EXTRA_ARGS}
      - TZ=${TZ}
      - TS_SERVE_CONFIG=/config/netronome.json
      - TS_SOCKET=/var/run/tailscale/tailscaled.sock
    volumes:
      - /dev/net/tun:/dev/net/tun
      - ${BASE_DOCKER_DATA_PATH}/config:/config
      - tailscale-data-netronome:/var/lib/tailscale
      - tailscale-socket:/var/run/tailscale

volumes:
  tailscale-data-netronome:
  tailscale-socket:

networks:
  tailscale_network:
    ipam:
      config:
        - subnet: 172.19.0.0/16
```

### Environment file

Put the settings in a `.env` file next to the compose file:

```dotenv title=".env"
# Tailscale configuration
TS_AUTHKEY=tskey-auth-YOUR-KEY-HERE
TS_STATE_DIR=/var/lib/tailscale
TS_EXTRA_ARGS=--advertise-routes=192.168.1.0/24  # Optional
TZ=America/New_York

# Netronome configuration
NETRONOME__TAILSCALE_ENABLED=true
NETRONOME__TAILSCALE_METHOD=host  # Use the tailscaled of the sidecar
```

Discovery is on by default when `NETRONOME__TAILSCALE_ENABLED` is `true`. In a configuration file, the same settings are:

```toml title="config.toml"
[tailscale]
enabled = true
method = "host"  # Use host mode to connect to sidecar's tailscaled
auto_discover = true
discovery_interval = "5m"
discovery_port = 8200
```

### HTTPS with Tailscale Serve

Tailscale Serve can give Netronome an HTTPS address with a certificate on your tailnet. Create `netronome.json` in the `./config` directory:

```json title="netronome.json"
{
    "TCP": {
        "443": {
            "HTTPS": true
        }
    },
    "Web": {
        "${TS_CERT_DOMAIN}:443": {
            "Handlers": {
                "/": {
                    "Proxy": "http://127.0.0.1:7575"
                }
            }
        }
    },
    "AllowFunnel": {
        "${TS_CERT_DOMAIN}:443": false
    }
}
```

With this file, Tailscale serves HTTPS on port 443 with its own certificate and sends all requests to Netronome on port 7575. Funnel is off, so only devices on your tailnet can open the address.

Tailscale replaces `${TS_CERT_DOMAIN}` with the full domain name of the node.

### Troubleshooting the sidecar

If the log shows `no running tailscaled found on host`, Netronome cannot connect to the socket:

1. Make sure that the Tailscale container creates the socket:

   ```bash
   docker exec netronome-ts ls -la /var/run/tailscale/
   ```

2. Make sure that both containers mount the socket volume:

   ```bash
   docker inspect netronome | grep -A5 Mounts
   docker inspect netronome-ts | grep -A5 Mounts
   ```

If the server does not discover agents:

1. Show the Tailscale status:

   ```bash
   docker exec netronome-ts tailscale status
   ```

2. Make sure that the agents listen on the discovery port, which is `8200` by default.
3. Read the Netronome log:

   ```bash
   docker logs netronome | grep -i tailscale
   ```

### Other Docker setups

To give Netronome its own Tailscale node, use `tsnet`. You need no sidecar:

```yaml title="docker-compose.yml"
netronome:
  image: ghcr.io/autobrr/netronome:latest
  environment:
    - NETRONOME__TAILSCALE_ENABLED=true
    - NETRONOME__TAILSCALE_METHOD=tsnet
    - NETRONOME__TAILSCALE_AUTH_KEY=tskey-auth-YOUR-KEY
    - NETRONOME__TAILSCALE_HOSTNAME=netronome-monitor
  volumes:
    - "./netronome:/data"
  ports:
    - 7575:7575
```

The image sets `HOME` to `/data`, so the node state goes to `/data/.config/netronome/tsnet` in the data volume.

On Linux, Netronome can also use the host network and the `tailscaled` of the host. Mount the socket directory of the host:

```yaml title="docker-compose.yml"
netronome:
  image: ghcr.io/autobrr/netronome:latest
  network_mode: host
  environment:
    - NETRONOME__TAILSCALE_ENABLED=true
    - NETRONOME__TAILSCALE_METHOD=host
  volumes:
    - "./netronome:/data"
    - /var/run/tailscale:/var/run/tailscale:ro  # Mount host's socket read-only
```

For more information, see the [Tailscale Docker documentation](https://tailscale.com/kb/1282/docker).
