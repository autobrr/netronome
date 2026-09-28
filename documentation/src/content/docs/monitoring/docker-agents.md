---
title: Docker agents
description: Run the Netronome agent in a Docker container to monitor a VPN tunnel or a host interface.
sidebar:
  order: 2
---

You can run the [agent](/monitoring/agents/) in a Docker container, for example to monitor the VPN traffic of a Gluetun container.

The Netronome image contains the `vnstat` command, but it does not run the vnstat daemon. The agent reads the bandwidth data from a vnstat database. You therefore run a separate vnstat container, and the agent reads its database.

Two conditions apply to each setup:

- The agent and vnstat must be in the same network namespace as the interface. A container cannot see the interfaces of the host or of other containers.
- The agent must mount the same vnstat database as the vnstat container, at `/var/lib/vnstat`.

The image starts the server by default. To start the agent, give `agent` as the command.

## Monitor a VPN tunnel

In this example, the agent and vnstat use `network_mode: "service:gluetun"`. They share the network namespace of Gluetun and see its tunnel interface.

<details>
<summary>Compose example for a VPN tunnel</summary>

```yaml title="docker-compose.yml"
services:
  # Gluetun - VPN client container
  gluetun:
    image: qmcgaw/gluetun:latest
    container_name: gluetun
    restart: unless-stopped
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - /path/to/gluetun:/gluetun
    environment:
      - VPN_SERVICE_PROVIDER=your_provider
      - VPN_TYPE=wireguard
      # ... your VPN configuration
    networks:
      monitoring:
        aliases:
          - netronome-vpn-agent  # Allows dashboard to reach agent by name

  # vnstat - collects bandwidth data on the VPN tunnel
  vnstat:
    image: vergoh/vnstat:latest
    container_name: vnstat
    restart: unless-stopped
    network_mode: "service:gluetun"
    depends_on:
      - gluetun
    environment:
      - TZ=
    volumes:
      - /path/to/vnstat:/var/lib/vnstat # Add a mount for the vnstat db

  # Netronome VPN agent - monitors VPN tunnel traffic
  netronome-vpn-agent:
    image: ghcr.io/autobrr/netronome:latest # You could also put the agent bin in a smaller image
    container_name: netronome-vpn-agent
    restart: unless-stopped
    network_mode: "service:gluetun"
    depends_on:
      - gluetun
      - vnstat
    environment:
      - TZ=
      - NETRONOME__AGENT_HOST=0.0.0.0
      - NETRONOME__AGENT_PORT=8200
      - NETRONOME__AGENT_API_KEY=  # Optional: set for authentication
    command:
      - agent
      - --interface
      - tun0  # VPN tunnel interface
    volumes:
      - /path/to/vnstat:/var/lib/vnstat:ro
    cap_add:
      - NET_RAW
      - NET_ADMIN

  # Netronome dashboard - main web interface
  netronome:
    image: ghcr.io/autobrr/netronome:latest
    container_name: netronome
    restart: unless-stopped
    environment:
      - TZ=
      - NETRONOME__HOST=0.0.0.0
      - NETRONOME__PORT=7575
    ports:
      - "7575:7575"
    volumes:
      - /path/to/netronome:/data
    networks:
      - monitoring
    cap_add:
      - NET_RAW
      - NET_ADMIN

networks:
  monitoring:
    driver: bridge
```

</details>

The Gluetun container has the alias `netronome-vpn-agent` on the `monitoring` network. In Netronome, add the agent with the URL `http://netronome-vpn-agent:8200`.

To find the name of the tunnel interface, run this command:

```sh
docker exec gluetun ip -br link
```

Common interface names:

- `tun0`: OpenVPN, or a Gluetun custom provider.
- `wg0`: WireGuard.

## Monitor a host interface

To monitor a physical interface of the host, give the agent and vnstat the host network. Both must see the same interface, and both must use the same vnstat database.

This example is for Linux. On Docker Desktop, the host network is the network of the Docker virtual machine, not the physical interface of the computer.

<details>
<summary>Compose example for a host interface</summary>

```yaml title="docker-compose.yml"
services:
  netronome:
    image: ghcr.io/autobrr/netronome:latest
    container_name: netronome
    command: ["serve"]
    environment:
      - NETRONOME__HOST=0.0.0.0
      - NETRONOME__PORT=7575
      - TZ=UTC
    ports:
      - "7575:7575"
    volumes:
      - ./netronome/config:/config
      - ./netronome/data:/data
    restart: unless-stopped
    networks:
      - netronome

  # The agent needs the host network to see the host interfaces.
  # Do not add a "ports" entry. It is not compatible with the host network.
  netronome-agent:
    image: ghcr.io/autobrr/netronome:latest
    container_name: netronome-agent
    network_mode: host
    environment:
      - NETRONOME__AGENT_API_KEY=${NETRONOME_AGENT_API_KEY}
    command:
      - agent
      - --interface
      - <interface>
    volumes:
      - ./netronome/vnstat:/var/lib/vnstat
    restart: unless-stopped

  vnstat:
    image: ghcr.io/vergoh/vnstat
    container_name: vnstat
    network_mode: host
    cap_add:
      - NET_ADMIN
      - NET_RAW
    environment:
      - VNSTAT_Interfaces=<interface>
      - TZ=UTC
    volumes:
      - ./netronome/vnstat:/var/lib/vnstat
    restart: unless-stopped

networks:
  netronome:
    driver: bridge
```

</details>

Replace `<interface>` with the name of the host interface, for example `eth0`.

The `netronome` container runs as the image user `netronome`, and that user must own `./netronome/data`. If Docker creates the folder, root owns it. Before you start the stack the first time, run `chown` in the container as root:

```bash
docker compose run --rm --user root --entrypoint chown netronome netronome:netronome /data
```

The agent is on the host network, but the dashboard is on a bridge network. The dashboard therefore cannot find the agent by its container name. Add the agent with the IP address of the host, for example `http://192.168.1.10:8200`.

Give the API key to the agent with `NETRONOME__AGENT_API_KEY`, not with the `--api-key` flag. Any user of the host can read the command line of a container in `docker inspect` and in the process list.

Keep the key out of the compose file. Put it in a `.env` file next to the compose file, and keep that file out of Git:

```dotenv title=".env"
NETRONOME_AGENT_API_KEY=your-key-here
```

## Missing interfaces

vnstat records only the interfaces in its database. The default of `AlwaysAddNewInterfaces` is 0, so the daemon does not add an interface that appears later. The `vergoh/vnstat` image adds the interfaces only when it creates the database. An interface that was down or absent at that time stays unrecorded.

The agent then shows no bandwidth data for that interface. The agent log shows `No interface matching "eth1" found in database`.

To add the interface:

1. Show the interfaces in the vnstat database:

   ```sh
   docker exec vnstat vnstat
   ```

2. Show the interfaces that the container can see:

   ```sh
   docker exec vnstat vnstat --iflist
   ```

3. Add the missing interface:

   ```sh
   docker exec vnstat vnstat --add -i eth1
   ```

The daemon reads the new interface at its next save, because `RescanDatabaseOnSave` is on by default. This takes up to 5 minutes (`SaveInterval`), or up to 30 minutes when the interface is down (`OfflineSaveInterval`). To apply it at once, restart the container.

To record each new interface without this procedure, set `AlwaysAddNewInterfaces 1` in `/etc/vnstat.conf`.

If the interface is not in the `--iflist` output, the container cannot see it. Put the agent and vnstat in the same network namespace as the interface.

## Limit the monitored interfaces

By default, vnstat records all interfaces that it finds, for example `eth0` and `tun0`. To record only the VPN tunnel, remove the other interfaces from the database:

```sh
# Remove unwanted interfaces from vnstat
docker exec vnstat vnstat --remove -i eth0 --force

# Show the interfaces that vnstat still records
docker exec vnstat vnstat
```

The `--interface` flag of the agent selects the interface for the live data and the history. The history without `--interface` contains all interfaces in the vnstat database. See [The interface setting](/monitoring/agents/#the-interface-setting).
