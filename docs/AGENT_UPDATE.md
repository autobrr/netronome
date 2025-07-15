# Netronome Agent Update Documentation

## Overview

The Netronome agent now includes a built-in self-update mechanism that allows you to easily update to the latest version without manual intervention.

## Update Methods

### 1. Using the Built-in Update Command

The simplest way to update your Netronome agent:

```bash
# Check current version
/opt/netronome/netronome version

# Update to latest version
/opt/netronome/netronome update

# If running as a systemd service, restart it
systemctl restart netronome-agent
```

### 2. Using the Install Script

If you installed using the one-liner script, you can also update using:

```bash
curl -sL https://raw.githubusercontent.com/autobrr/netronome/main/scripts/install-agent.sh | bash -s -- --update
```

This will automatically:

- Download and install the latest version
- Restart the systemd service if it's running

### 3. Manual Update

For manual updates, you can:

1. Download the latest release from GitHub
2. Stop the service: `systemctl stop netronome-agent`
3. Replace the binary in `/opt/netronome/`
4. Start the service: `systemctl start netronome-agent`

## Version Information

The update command includes several safety features:

- **Version Comparison**: Only updates if a newer version is available
- **Checksum Validation**: Verifies the integrity of downloaded files
- **Atomic Updates**: The old binary is only replaced after successful download
- **Error Handling**: Graceful failure with clear error messages

## Build-time Version Information

When building from source, you can include version information:

```bash
go build -ldflags "-X main.Version=v1.0.0 -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/netronome
```

## Troubleshooting

### Permission Denied

If you get a permission error, ensure you're running the update command with appropriate permissions:

```bash
sudo /opt/netronome/netronome update
```

### Service Not Restarting

After an update, the service needs to be manually restarted:

```bash
systemctl restart netronome-agent
```

### Checking Update Status

You can verify the update was successful by checking the version:

```bash
/opt/netronome/netronome version
```

## Auto-Update Configuration (Future Enhancement)

While not currently implemented, future versions may include automatic update checking via cron:

```bash
# Example crontab entry (not yet implemented)
0 3 * * * /opt/netronome/netronome update --quiet && systemctl restart netronome-agent
```
