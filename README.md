<h1 align="center">Netronome</h1>

<p align="center">
  <img src=".github/assets/netronome_dashboard.png" alt="Netronome">
</p>

Netronome (Network Metronome) is a modern network speed testing and monitoring tool with a clean, intuitive web interface. It offers both scheduled and on-demand speed tests with detailed visualizations and historical tracking.

## 📑 Table of Contents

- [Features](#-features)
- [Getting Started](#-getting-started)
  - [Swizzin](#swizzin)
  - [Linux Generic](#linux-generic)
  - [Docker Installation](#docker-installation)
- [Configuration](#️-configuration)
  - [Configuration File](#configuration-file-configtoml)
  - [Environment Variables](#environment-variables)
  - [Database](#database)
  - [Authentication](#authentication)
  - [Notifications](#notifications)
  - [CLI Commands](#cli-commands)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

- **Speed Testing**

  - Support for Speedtest.net, iperf3 servers, and LibreSpeed
  - Real-time test progress visualization
  - Latency and jitter measurements

- **Monitoring**

  - Interactive historical data charts
  - Customizable time ranges (1d, 3d, 1w, 1m, all)

- **Scheduling & Automation**

  - Automated speed tests with flexible scheduling

- **Modern Interface**

  - Clean, responsive design
  - Dark mode optimized
  - Real-time updates
  - Interactive charts and visualizations

- **Flexible Authentication**
  - Built-in user authentication
  - OpenID Connect support

## 🚀 Getting Started

### Swizzin

```bash
sudo box install netronome
```

### Linux Generic

Download the latest release, or download the source code and build it yourself using make build.

```bash
wget $(curl -s https://api.github.com/repos/autobrr/netronome/releases/latest | grep download | grep linux_x86_64 | cut -d\" -f4)
```

#### Unpack

Run with root or sudo. If you do not have root, or are on a shared system, place the binaries somewhere in your home directory like ~/.bin.

```bash
tar -C /usr/local/bin -xzf netronome*.tar.gz
```

This will extract both netronome and netronomectl to /usr/local/bin. Note: If the command fails, prefix it with sudo and re-run again.
Systemd (Recommended)

On Linux-based systems, it is recommended to run netronome as a sort of service with auto-restarting capabilities, in order to account for potential downtime. The most common way is to do it via systemd.

You will need to create a service file in /etc/systemd/system/ called netronome.service.

```bash
touch /etc/systemd/system/netronome@.service
```

Then place the following content inside the file (e.g. via nano/vim/ed):

```
[Unit]
Description=netronome service for %i
After=syslog.target network-online.target

[Service]
Type=simple
User=%i
Group=%i
ExecStart=/usr/bin/netronome --config=/home/%i/.config/netronome/config.toml

[Install]
WantedBy=multi-user.target
```

Start the service. Enable will make it startup on reboot.

```bash
systemctl enable -q --now --user netronome@$USER
```

By default, the configuration is set to listen on 127.0.0.1. While netronome works fine as is exposed to the internet, it is recommended to use a reverse proxy like nginx, caddy or traefik.

If you are not running a reverse proxy change host in the config.toml to 0.0.0.0.

### Docker Installation

For containerized deployment see [docker-compose.yml](docker-compose.yml) and [docker-compose.postgres.yml](docker-compose.postgres.yml).

## ⚙️ Configuration

### Configuration File (config.toml)

Netronome can be configured using a TOML file. Generate a default configuration:

```bash
netronome generate-config
```

This will create a `config.toml` file with default settings:

```toml
# Netronome Configuration

[database]
type = "sqlite"
path = "netronome.db"

[server]
host = "127.0.0.1"
port = 7575
base_url = ""

[logging]
level = "info"

[oidc]
issuer = ""
client_id = ""
client_secret = ""
redirect_url = ""

[speedtest]
timeout = 30

[speedtest.iperf]
test_duration = 10
parallel_conns = 4
timeout = 30
enable_udp = false
udp_bandwidth = "100M"

[speedtest.iperf.ping]
count = 5
interval = 200
timeout = 10

[notifications]
enabled = false
webhook_url = ""
ping_threshold = 30
upload_threshold = 200
download_threshold = 200
```

#### LibreSpeed Configuration

Netronome supports LibreSpeed servers for speed testing. A `librespeed-servers.json` file should be placed in the same directory as your configuration file. This file contains the LibreSpeed server definitions.

Example `librespeed-servers.json`:

```json
[
  {
    "id": 1,
    "name": "Clouvider - London, UK",
    "server": "http://lon.speedtest.clouvider.net/backend",
    "dlURL": "garbage.php",
    "ulURL": "empty.php",
    "pingURL": "empty.php",
    "getIpURL": "getIP.php"
  },
  {
    "id": 2,
    "name": "Your Custom Server",
    "server": "http://your-server.example.com/backend",
    "dlURL": "garbage.php",
    "ulURL": "empty.php",
    "pingURL": "empty.php",
    "getIpURL": "getIP.php"
  }
]
```

**Note:** When using Docker, the LibreSpeed CLI tool (`librespeed-cli`) is automatically included in the container.

### Environment Variables

| Variable                                      | Description                                                       | Default                                      | Required               |
| --------------------------------------------- | ----------------------------------------------------------------- | -------------------------------------------- | ---------------------- |
| `NETRONOME__HOST`                             | Server host                                                       | `127.0.0.1`                                  | No                     |
| `NETRONOME__PORT`                             | Server port                                                       | `7575`                                       | No                     |
| `NETRONOME__GIN_MODE`                         | Gin framework mode (`debug`/`release`)                            | `release`                                    | No                     |
| `NETRONOME__DB_TYPE`                          | Database type (`sqlite`/`postgres`)                               | `sqlite`                                     | No                     |
| `NETRONOME__DB_PATH`                          | SQLite database file path                                         | `./netronome.db`                             | Only for SQLite        |
| `NETRONOME__DB_HOST`                          | PostgreSQL host                                                   | `localhost`                                  | Only for PostgreSQL    |
| `NETRONOME__DB_PORT`                          | PostgreSQL port                                                   | `5432`                                       | Only for PostgreSQL    |
| `NETRONOME__DB_USER`                          | PostgreSQL user                                                   | `postgres`                                   | Only for PostgreSQL    |
| `NETRONOME__DB_PASSWORD`                      | PostgreSQL password                                               | -                                            | Only for PostgreSQL    |
| `NETRONOME__DB_NAME`                          | PostgreSQL database name                                          | `netronome`                                  | Only for PostgreSQL    |
| `NETRONOME__DB_SSLMODE`                       | PostgreSQL SSL mode                                               | `disable`                                    | Only for PostgreSQL    |
| `NETRONOME__IPERF_TEST_DURATION`              | Duration of iPerf tests in seconds                                | `10`                                         | No                     |
| `NETRONOME__IPERF_PARALLEL_CONNS`             | Number of parallel iPerf connections                              | `4`                                          | No                     |
| `NETRONOME__IPERF_TIMEOUT`                    | Timeout for iperf3 tests in seconds                               | `60`                                         | No                     |
| `NETRONOME__IPERF_ENABLE_UDP`                 | Enable UDP mode for jitter testing                                | `false`                                      | No                     |
| `NETRONOME__IPERF_UDP_BANDWIDTH`              | Bandwidth limit for UDP tests (e.g., "100M")                      | `100M`                                       | No                     |
| `NETRONOME__IPERF_PING_COUNT`                 | Number of ping packets to send for iperf3 tests                   | `5`                                          | No                     |
| `NETRONOME__IPERF_PING_INTERVAL`              | Interval between ping packets in milliseconds for iperf3 tests    | `200`                                        | No                     |
| `NETRONOME__IPERF_PING_TIMEOUT`               | Timeout for ping tests in seconds for iperf3 tests                | `10`                                         | No                     |
| `NETRONOME__SPEEDTEST_TIMEOUT`                | Speedtest timeout in seconds                                      | `30`                                         | No                     |
| `NETRONOME__LOG_LEVEL`                        | Log level (`trace`/`debug`/`info`/`warn`/`error`/`fatal`/`panic`) | `info`                                       | No                     |
| `NETRONOME__OIDC_ISSUER`                      | OpenID Connect issuer URL                                         | -                                            | Only for OIDC          |
| `NETRONOME__OIDC_CLIENT_ID`                   | OpenID Connect client ID                                          | -                                            | Only for OIDC          |
| `NETRONOME__OIDC_CLIENT_SECRET`               | OpenID Connect client secret                                      | -                                            | Only for OIDC          |
| `NETRONOME__OIDC_REDIRECT_URL`                | OpenID Connect redirect URL                                       | http://localhost:7575/api/auth/oidc/callback | Only for OIDC          |
| `NETRONOME__DEFAULT_PAGE`                     | Default page number for pagination                                | `1`                                          | No                     |
| `NETRONOME__DEFAULT_PAGE_SIZE`                | Default page size for pagination                                  | `20`                                         | No                     |
| `NETRONOME__MAX_PAGE_SIZE`                    | Maximum page size for pagination                                  | `100`                                        | No                     |
| `NETRONOME__DEFAULT_TIME_RANGE`               | Default time range for data queries                               | `1w`                                         | No                     |
| `NETRONOME__DEFAULT_LIMIT`                    | Default limit for data queries                                    | `20`                                         | No                     |
| `NETRONOME__SESSION_SECRET`                   | Session secret for authentication                                 | -                                            | No                     |
| `NETRONOME__NOTIFICATIONS_ENABLED`            | Enable or disable notifications                                   | `false`                                      | No                     |
| `NETRONOME__NOTIFICATIONS_WEBHOOK_URL`        | Webhook URL for notifications                                     | -                                            | Only for Notifications |
| `NETRONOME__NOTIFICATIONS_PING_THRESHOLD`     | Ping threshold in ms for notifications                            | `30`                                         | No                     |
| `NETRONOME__NOTIFICATIONS_UPLOAD_THRESHOLD`   | Upload threshold in Mbps for notifications                        | `200`                                        | No                     |
| `NETRONOME__NOTIFICATIONS_DOWNLOAD_THRESHOLD` | Download threshold in Mbps for notifications                      | `200`                                        | No                     |
| `NETRONOME__NOTIFICATIONS_DISCORD_MENTION_ID` | Discord user/role ID to mention on alerts                         | -                                            | No                     |
| `NETRONOME__AUTH_WHITELIST`                   | Whitelist for authentication                                      | -                                            | No                     |

### Database

Netronome supports two database backends:

1. **SQLite** (Default)

   - No additional setup required

2. **PostgreSQL**
   - Configure via:
     ```bash
     NETRONOME_DB_TYPE=postgres
     NETRONOME_DB_HOST=localhost
     NETRONOME_DB_PORT=5432
     NETRONOME_DB_USER=postgres
     NETRONOME_DB_PASSWORD=your-password
     NETRONOME_DB_NAME=netronome
     NETRONOME_DB_SSLMODE=disable
     ```

### Authentication

Netronome supports two authentication methods:

1. **Built-in Authentication**

   - Username/password authentication
   - Default option if no OIDC is configured

2. **OpenID Connect (OIDC)**

   - Integration with identity providers (Pocket-ID, Authelia, Authentik, Keycloak etc.)
   - PKCE support
   - Configure via environment variables:
     ```bash
     OIDC_ISSUER=https://pocketid.domain.net
     OIDC_CLIENT_ID=your-client-id
     OIDC_CLIENT_SECRET=your-client-secret
     OIDC_REDIRECT_URL=https://netronome.domain.net/api/auth/oidc/callback
     ```

3. **IP Whitelisting**
   - Bypass authentication for specific network ranges or IP addresses.
   - Configure in `config.toml` using CIDR notation:
     ```toml
     [auth]
     whitelist = ["127.0.0.1/32"]
     ```

### Notifications

Netronome can send notifications to a webhook URL after each speed test. This is useful for integrating with services like Discord, Slack, or any other service that accepts webhooks.

To enable notifications, you need to set the following in your `config.toml` or as environment variables:

```toml
[notifications]
enabled = true
webhook_url = "your-webhook-url"
ping_threshold = 30
upload_threshold = 200
download_threshold = 200
discord_mention_id = ""
```

- `enabled` - Enable or disable notifications
- `webhook_url` - The webhook URL to send notifications to
- `ping_threshold` - The ping threshold in ms. If the ping is higher than this value, a notification will be sent.
- `upload_threshold` - The upload threshold in Mbps. If the upload speed is lower than this value, a notification will be sent.
- `download_threshold` - The download threshold in Mbps. If the download speed is lower than this value, a notification will be sent.
- `discord_mention_id` - Optional. A Discord user ID or role ID to mention when an alert is triggered. For example, `123456789012345678` for a user or `&123456789012345678` for a role.

### CLI Commands

Netronome provides several command-line commands:

- `generate-config` - Generate a default configuration file to `~/.config/netronome/config.toml`
- `serve` - Starts the Netronome server
- `create-user` - Create a new user
- `change-password` - Change password for an existing user

Examples:

```bash
netronome generate-config

# Create a new user (interactive)
netronome --config /home/username/.config/netronome/config.toml create-user username

# Create a new user (non-interactive)
echo "password123" | netronome --config /home/username/.config/netronome/config.toml create-user username

# Change user password (interactive)
netronome --config /home/username/.config/netronome/config.toml change-password username

# Change user password (non-interactive)
echo "newpassword123" | netronome --config /home/username/.config/netronome/config.toml change-password username
```

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.
