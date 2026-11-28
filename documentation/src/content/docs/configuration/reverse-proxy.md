---
title: Reverse proxy
description: Serve Netronome under a subpath behind nginx or another reverse proxy.
sidebar:
  order: 3
---

A reverse proxy is a web server that receives the requests from browsers and sends them to Netronome. You need one to serve Netronome over HTTPS, on a shared domain, or under a subpath such as `/netronome`.

## Server configuration

- `server.host` (default `127.0.0.1`): The address that Netronome listens on. With `127.0.0.1`, only programs on the same host can connect, which is correct when the proxy runs on the same host. In a container, `netronome generate-config` writes `0.0.0.0`.
- `server.port` (default `7575`): The port that Netronome listens on.
- `server.base_url` (default `/`): The subpath. Netronome serves the web interface and the API under it. If the value does not start with `/`, Netronome adds the `/`. You do not have to build the web interface again after you change it.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

## Serve Netronome under a subpath

This example serves Netronome at `https://example.com/netronome/` with nginx on the same host.

1. Set the base URL in `config.toml`:

   ```toml title="config.toml"
   [server]
   host = "127.0.0.1"  # Listen only on localhost, because nginx sends the requests
   port = 7575
   base_url = "/netronome"
   ```

2. Add these location blocks to the `server` block of your nginx configuration:

   ```nginx
   # Redirect /netronome to /netronome/
   location = /netronome {
       return 301 /netronome/;
   }

   location /netronome/ {
       proxy_pass http://127.0.0.1:7575;
       proxy_http_version 1.1;
       proxy_set_header Upgrade $http_upgrade;
       proxy_set_header Connection "upgrade";
       proxy_buffering off;
       proxy_cache off;
       proxy_read_timeout 86400;
   }
   ```

3. Reload nginx, then restart Netronome.

Do not put a path after the port in `proxy_pass`. Netronome expects the full path, with `/netronome` at the start.

## Client addresses and HTTPS

The IP whitelist and HTTPS need two more headers. Add them to the `location /netronome/` block:

```nginx
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

- If you use the [IP whitelist](/configuration/authentication/#ip-whitelist), Netronome needs the real address of each client. Send `X-Forwarded-For`, and add the address of the proxy to `auth.trusted_proxies`. Without `trusted_proxies`, Netronome ignores the header.
- If the proxy serves HTTPS, send `X-Forwarded-Proto`. When the value is `https`, Netronome gives the session cookie the `Secure` flag, and the browser then sends the cookie only over HTTPS.

## OIDC behind a proxy

If you use OIDC with a base URL, the redirect URL must contain the base URL. For the example above, it is `https://example.com/netronome/api/auth/oidc/callback`. For more, see [Authentication](/configuration/authentication/#openid-connect-oidc).
