---
title: GeoIP
description: Show the country and the network owner of each hop in traceroute and MTR results.
sidebar:
  order: 7
---

GeoIP adds two values to each hop in traceroute results and in MTR results of packet loss monitors:

- The country, as a flag.
- The ASN (autonomous system number) and the name of the network that owns the address, for example `AS13335 Cloudflare, Inc.`.

GeoIP is optional. Without it, Netronome works in the same way, but the hops show no country and no ASN. Netronome reads the free GeoLite2 databases from MaxMind.

## Configuration

- `geoip.country_database_path` (default empty): The path of the GeoLite2 Country database file.
- `geoip.asn_database_path` (default empty): The path of the GeoLite2 ASN database file.

Each key also has an environment variable. See [Environment variables](/reference/environment-variables/).

You can set one path or both. Each database gives its own value.

## Set up GeoIP

1. Make a free account at [MaxMind](https://www.maxmind.com/en/geolite2/signup).
2. Download the GeoLite2 Country database and the GeoLite2 ASN database in `.mmdb` format.
3. Put the files on the host that runs Netronome.
4. Add the paths to `config.toml`:

   ```toml title="config.toml"
   [geoip]
   country_database_path = "/path/to/GeoLite2-Country.mmdb"
   asn_database_path = "/path/to/GeoLite2-ASN.mmdb"
   ```

5. Restart Netronome.

At startup, Netronome writes `GeoIP Country database loaded successfully` and `GeoIP ASN database loaded successfully` to the log. If it cannot open a file, it writes a warning with the path and continues without that database.

In Docker, put the files in a mounted volume and use the path inside the container.

:::note
Netronome does not update the databases. To get new data, download the files again and restart Netronome.
:::
