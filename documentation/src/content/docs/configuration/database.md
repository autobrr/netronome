---
title: Database
description: Store Netronome data in SQLite, the default, or in PostgreSQL.
sidebar:
  order: 2
---

Netronome keeps its data in SQLite by default. It also supports PostgreSQL. At each start, Netronome applies the database migrations that it has not applied yet. A migration is a step that creates or changes the tables.

- `database.type` (default `sqlite`): The database type. The value is `sqlite` or `postgres`.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

## SQLite

SQLite needs no setup. Netronome creates the database file and its directory at the first start.

- `database.path` (default `netronome.db`): The path of the database file. A relative `path` in the configuration file starts from the directory of that file. With the default configuration file, the database is at `~/.config/netronome/netronome.db`. A relative path in `NETRONOME__DB_PATH` starts from the working directory.

## PostgreSQL

Create an empty database and a user for Netronome first. Then set `type` to `postgres` and give the connection details:

- `database.host` (default `localhost`): The host name or IP address of the PostgreSQL server.
- `database.port` (default `5432`): The port of the PostgreSQL server.
- `database.user` (default `postgres`): The PostgreSQL user.
- `database.password` (default empty): The password of the PostgreSQL user.
- `database.dbname` (default `netronome`): The name of the database.
- `database.sslmode` (default `disable`): The PostgreSQL SSL mode, for example `disable`, `require`, or `verify-full`.

In `config.toml`:

```toml title="config.toml"
[database]
type = "postgres"
host = "localhost"
port = 5432
user = "netronome"
password = "your-password"
dbname = "netronome"
sslmode = "disable"
```

With environment variables:

```bash
export NETRONOME__DB_TYPE=postgres
export NETRONOME__DB_HOST=localhost
export NETRONOME__DB_PORT=5432
export NETRONOME__DB_USER=netronome
export NETRONOME__DB_PASSWORD=your-password
export NETRONOME__DB_NAME=netronome
```

The repository has a Docker Compose file that starts Netronome with PostgreSQL 17: `distrib/docker/docker-compose.postgres.yml`. For Docker setup, see [Installation](/getting-started/installation/).

:::note
Netronome does not copy data between SQLite and PostgreSQL. If you change `type`, Netronome starts with an empty database.
:::

## Delete old results

To make the database smaller, delete old results in the web interface:

1. Open Settings, then Data.
2. Under Purge History, select the age of the records to delete: older than 7 days, 30 days, 90 days, 1 year, or everything.
3. Select Purge, then confirm.

The purge deletes speed test results, packet loss results, and DNS results. You cannot undo it. On SQLite, Netronome then runs `VACUUM`, so that the file on disk becomes smaller.
