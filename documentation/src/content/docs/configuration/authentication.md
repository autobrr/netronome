---
title: Authentication
description: Sign in with a local account or an OpenID Connect provider, and let trusted networks skip the sign-in.
sidebar:
  order: 1
---

Netronome has three ways to control access:

- A local account with a username and a password.
- OpenID Connect (OIDC), a standard sign-in through an external identity provider.
- An IP whitelist, a list of networks that do not have to sign in.

You can use them together. For example, you can sign in with OIDC from the internet and skip the sign-in on your home network.

## Local account

On the first visit, the web interface opens the registration page. You can register only while no account exists. After you create the first account, the web interface refuses all new registrations.

You can also manage the account from the command line. Run these commands on the host that runs Netronome, with the same `--config` flag as the server:

```bash
netronome create-user <username>      # Create a user
netronome change-password <username>  # Change the password of a user
```

Each command asks for the password. If the input is not a terminal, the command reads the password from standard input:

```bash
echo 'new-password' | netronome change-password admin
```

For all commands, see [CLI](/reference/cli/).

## Sessions

A session cookie is valid for 24 hours. Netronome renews the cookie each time the web interface checks your session.

Netronome signs the session cookie with the session secret. The `netronome generate-config` command writes a random secret into the configuration file.

- `session.session_secret` (default empty): The secret that signs the session cookie.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

If the secret is empty, Netronome keeps the sessions in memory only. Then all users must sign in again after each restart. Without a secret, Netronome also cannot renew OIDC tokens.

:::caution
Keep the session secret private. A person who has the secret can make a valid session cookie.
:::

## OpenID Connect (OIDC)

To use OIDC, create a client in your identity provider first. Set the redirect URL of the client to `https://<your-host>/api/auth/oidc/callback`. If you use a [base URL](/configuration/reverse-proxy/), put it before `/api`, for example `https://example.com/netronome/api/auth/oidc/callback`.

Then give Netronome the details of the client:

- `oidc.issuer` (default empty): The URL of the identity provider.
- `oidc.client_id` (default empty): The client ID from the identity provider.
- `oidc.client_secret` (default empty): The client secret from the identity provider.
- `oidc.redirect_url` (default empty): The redirect URL of the client.
- `oidc.scopes` (default `openid`, `profile`): The scopes that Netronome asks for.

In `config.toml`:

```toml title="config.toml"
[oidc]
issuer = "https://auth.example.com"
client_id = "netronome"
client_secret = "your-client-secret"
redirect_url = "https://netronome.example.com/api/auth/oidc/callback"
scopes = ["openid", "profile", "offline_access"]
```

With environment variables:

```bash
export NETRONOME__OIDC_ISSUER=https://auth.example.com
export NETRONOME__OIDC_CLIENT_ID=netronome
export NETRONOME__OIDC_CLIENT_SECRET=your-client-secret
export NETRONOME__OIDC_REDIRECT_URL=https://netronome.example.com/api/auth/oidc/callback
```

Netronome turns on OIDC when `issuer` has a value. It reads the provider endpoints from `<issuer>/.well-known/openid-configuration`.

Notes on the scopes:

- `NETRONOME__OIDC_SCOPES` accepts commas or spaces between the scopes.
- If your list does not contain `openid`, Netronome adds it.
- Add `offline_access` if your provider gives refresh tokens only with that scope. With a refresh token and a session secret, Netronome renews the OIDC tokens and the session stays valid.

Netronome shows the name from the `preferred_username` claim. If that claim is empty, it uses `name`, and then `sub`.

With OIDC on, the sign-in page shows a button to sign in with OpenID. The page shows the password form only if a local account exists. With OIDC, you do not need a local account, and the web interface does not open the registration page.

At startup, Netronome tries to connect to the provider 10 times, with 5 seconds between attempts. If all attempts fail, the server starts without OIDC and the sign-in page hides the OIDC button. If a local account exists, you can still sign in with it.

## IP whitelist

A client with an address in a whitelisted network does not have to sign in. Write each network in CIDR notation, for example `192.168.1.0/24` for a network or `192.168.1.10/32` for one address.

- `auth.whitelist` (default empty): The networks that do not have to sign in.
- `auth.trusted_proxies` (default empty): The proxies whose client address headers Netronome reads. See [Behind a reverse proxy](#behind-a-reverse-proxy).

```toml title="config.toml"
[auth]
whitelist = ["127.0.0.1/32", "192.168.1.0/24"]
```

In the environment variable, put commas between the networks and no spaces:

```bash
export NETRONOME__AUTH_WHITELIST=127.0.0.1/32,192.168.1.0/24
```

Netronome ignores an entry that is not valid CIDR and writes a warning to the log.

### Behind a reverse proxy

If Netronome runs behind a reverse proxy, each request comes from the address of the proxy. By default, Netronome ignores the `X-Forwarded-For` and `X-Real-IP` headers, so Netronome compares only the proxy address with the whitelist.

To match the real client addresses, add the address of the proxy to `trusted_proxies`. Netronome then reads the client address from the headers of that proxy only. The list accepts IP addresses and CIDR networks.

```toml title="config.toml"
[auth]
whitelist = ["192.168.1.0/24"]
trusted_proxies = ["172.16.0.0/12"]
```

The proxy must send one of the two headers. For an nginx example, see [Reverse proxy](/configuration/reverse-proxy/).

### Turn off authentication

If a reverse proxy already does the authentication (for example Authelia or TinyAuth), you can whitelist all clients. IPv6 needs its own entry:

```toml title="config.toml"
[auth]
whitelist = ["0.0.0.0/0", "::/0"]
```

```bash
export NETRONOME__AUTH_WHITELIST=0.0.0.0/0,::/0
```

:::danger
Do this only when clients cannot connect to Netronome without the proxy. Bind Netronome to `127.0.0.1` or to a Docker network that only the proxy uses. Any client that can connect to Netronome gets full access.
:::
