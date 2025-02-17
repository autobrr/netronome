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
- [Development Commands](#️-development-commands)
- [Configuration](#️-configuration)
  - [Configuration File](#configuration-file-configtoml)
  - [Environment Variables](#environment-variables)
  - [Database](#database)
  - [Authentication](#authentication)
  - [CLI Commands](#cli-commands)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

- **Speed Testing**

  - Support for both Speedtest.net and iperf3 servers
  - Real-time test progress visualization
  - Detailed latency measurements

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
```

### Environment Variables

| Variable                          | Description                                                       | Default                                      | Required            |
| --------------------------------- | ----------------------------------------------------------------- | -------------------------------------------- | ------------------- |
| `NETRONOME__HOST`                 | Server host                                                       | `127.0.0.1`                                  | No                  |
| `NETRONOME__PORT`                 | Server port                                                       | `7575`                                       | No                  |
| `NETRONOME__GIN_MODE`             | Gin framework mode (`debug`/`release`)                            | `release`                                    | No                  |
| `NETRONOME__DB_TYPE`              | Database type (`sqlite`/`postgres`)                               | `sqlite`                                     | No                  |
| `NETRONOME__DB_PATH`              | SQLite database file path                                         | `./netronome.db`                             | Only for SQLite     |
| `NETRONOME__DB_HOST`              | PostgreSQL host                                                   | `localhost`                                  | Only for PostgreSQL |
| `NETRONOME__DB_PORT`              | PostgreSQL port                                                   | `5432`                                       | Only for PostgreSQL |
| `NETRONOME__DB_USER`              | PostgreSQL user                                                   | `postgres`                                   | Only for PostgreSQL |
| `NETRONOME__DB_PASSWORD`          | PostgreSQL password                                               | -                                            | Only for PostgreSQL |
| `NETRONOME__DB_NAME`              | PostgreSQL database name                                          | `netronome`                                  | Only for PostgreSQL |
| `NETRONOME__DB_SSLMODE`           | PostgreSQL SSL mode                                               | `disable`                                    | Only for PostgreSQL |
| `NETRONOME__IPERF_TEST_DURATION`  | Duration of iPerf tests in seconds                                | `10`                                         | No                  |
| `NETRONOME__IPERF_PARALLEL_CONNS` | Number of parallel iPerf connections                              | `4`                                          | No                  |
| `NETRONOME__SPEEDTEST_TIMEOUT`    | Speedtest timeout in seconds                                      | `30`                                         | No                  |
| `NETRONOME__LOG_LEVEL`            | Log level (`trace`/`debug`/`info`/`warn`/`error`/`fatal`/`panic`) | `info`                                       | No                  |
| `NETRONOME__OIDC_ISSUER`          | OpenID Connect issuer URL                                         | -                                            | Only for OIDC       |
| `NETRONOME__OIDC_CLIENT_ID`       | OpenID Connect client ID                                          | -                                            | Only for OIDC       |
| `NETRONOME__OIDC_CLIENT_SECRET`   | OpenID Connect client secret                                      | -                                            | Only for OIDC       |
| `NETRONOME__OIDC_REDIRECT_URL`    | OpenID Connect redirect URL                                       | http://localhost:7575/api/auth/oidc/callback | Only for OIDC       |
| `NETRONOME__DEFAULT_PAGE`         | Default page number for pagination                                | `1`                                          | No                  |
| `NETRONOME__DEFAULT_PAGE_SIZE`    | Default page size for pagination                                  | `20`                                         | No                  |
| `NETRONOME__MAX_PAGE_SIZE`        | Maximum page size for pagination                                  | `100`                                        | No                  |
| `NETRONOME__DEFAULT_TIME_RANGE`   | Default time range for data queries                               | `1w`                                         | No                  |
| `NETRONOME__DEFAULT_LIMIT`        | Default limit for data queries                                    | `20`                                         | No                  |

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
   - Integration with identity providers (Google, Okta, Auth0, Keycloak, Pocket-ID etc.)
   - Configure via environment variables:
     ```bash
     OIDC_ISSUER=https://pocketid.domain.net
     OIDC_CLIENT_ID=your-client-id
     OIDC_CLIENT_SECRET=your-client-secret
     OIDC_REDIRECT_URL=https://netronome.domain.net/api/auth/oidc/callback
     ```

### CLI Commands

Netronome provides several command-line commands for managing users:

- `generate-config` - Generate a default configuration file
- `serve` - Start the Netronome server
- `create-user` - Create a new user
- `change-password` - Change password for an existing user

Examples:

```bash
# Generate config
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
