# Docker Tailscale Sidecar Configuration

This guide explains how to run Netronome with a Tailscale sidecar container for secure networking and agent discovery.

## Overview

When running Netronome in Docker alongside a Tailscale sidecar container, special configuration is needed to enable Tailscale discovery features. The sidecar pattern allows you to:

- Keep Tailscale and Netronome in separate containers
- Share networking between containers
- Enable automatic agent discovery
- Maintain clean separation of concerns

## Prerequisites

- Docker and Docker Compose installed
- A Tailscale account and auth key
- Basic understanding of Docker networking

## Configuration

### Working Example with Docker Compose Anchors

For a cleaner setup when running multiple Tailscale containers, you can use YAML anchors:

```yaml
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

### Basic Configuration

If you're only running Netronome with Tailscale, here's a simpler configuration:

```yaml
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

### Environment Variables (.env file)

```bash
# Tailscale configuration
TS_AUTHKEY=tskey-auth-YOUR-KEY-HERE
TS_STATE_DIR=/var/lib/tailscale
TS_EXTRA_ARGS=--advertise-routes=192.168.1.0/24  # Optional
TZ=America/New_York

# Netronome configuration (optional)
NETRONOME__TAILSCALE_ENABLED=true
NETRONOME__TAILSCALE_METHOD=host  # Use host's tailscaled
```

### Serving with HTTPS via Tailscale Serve

To serve Netronome with HTTPS certificates through Tailscale, create a `netronome.json` file in your config directory:

```json
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

This configuration:
- Enables HTTPS on port 443 with automatic certificates from Tailscale
- Proxies all requests to Netronome running on port 7575
- Keeps the service private to your tailnet (Funnel disabled)

The `${TS_CERT_DOMAIN}` variable is automatically populated by Tailscale with your node's full domain name.

## Key Configuration Points

### 1. Socket Sharing

The most critical part is sharing the Tailscale socket between containers:

```yaml
volumes:
  - tailscale-socket:/var/run/tailscale  # Both containers mount this
```

And forcing the socket location in the Tailscale container:

```yaml
environment:
  - TS_SOCKET=/var/run/tailscale/tailscaled.sock
```

### 2. Network Mode

Use `network_mode: service:netronome-ts` to share the network namespace:

```yaml
netronome:
  network_mode: service:netronome-ts  # Shares network with Tailscale container
```

This allows Netronome to access the Tailscale network interface.

### 3. Netronome Configuration

Configure Netronome to use the host's tailscaled (which is actually in the sidecar):

```toml
# In your config.toml or via environment variables
[tailscale]
enabled = true
method = "host"  # Use host mode to connect to sidecar's tailscaled
auto_discover = true
discovery_interval = "5m"
discovery_port = 8200
```

## Troubleshooting

### Socket Connection Issues

If you see errors like "no running tailscaled found on host":

1. **Verify socket path**: Check that the Tailscale container is creating the socket at `/var/run/tailscale/tailscaled.sock`
   ```bash
   docker exec netronome-ts ls -la /var/run/tailscale/
   ```

2. **Check volume mounting**: Ensure both containers have the socket volume mounted
   ```bash
   docker inspect netronome | grep -A5 Mounts
   docker inspect netronome-ts | grep -A5 Mounts
   ```

3. **Test socket connectivity**: From inside the Netronome container
   ```bash
   docker exec netronome curl --unix-socket /var/run/tailscale/tailscaled.sock http://local-tailscaled.sock/localapi/v0/status
   ```

### Discovery Not Working

If agents aren't being discovered:

1. **Check Tailscale status**:
   ```bash
   docker exec netronome-ts tailscale status
   ```

2. **Verify agents are on discovery port**: Ensure agents are running on port 8200 (or your configured discovery port)

3. **Check logs**: Look for discovery-related messages
   ```bash
   docker logs netronome | grep -i tailscale
   ```

## Alternative Approaches

### Option 1: TSNet Mode (Separate Tailscale Instance)

If you prefer Netronome to have its own Tailscale identity:

```yaml
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

No sidecar needed with this approach.

### Option 2: Host Network Mode

If running on Linux, you can use the host's network and tailscaled:

```yaml
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

## Best Practices

1. **Use named volumes** for the socket directory to ensure proper permissions
2. **Set explicit socket paths** to avoid auto-detection issues
3. **Monitor logs** during initial setup to catch configuration problems early
4. **Test connectivity** before enabling auto-discovery
5. **Use environment variables** for sensitive data like auth keys

## Security Considerations

- The socket sharing grants Netronome full access to the Tailscale daemon
- Consider using read-only mounts where possible
- Use API keys for additional authentication on agents
- Regularly rotate Tailscale auth keys

## Additional Resources

- [Tailscale Docker Documentation](https://tailscale.com/kb/1282/docker)
- [Docker Networking Documentation](https://docs.docker.com/network/)
- [Netronome Tailscale Integration Guide](../README.md#tailscale-integration)