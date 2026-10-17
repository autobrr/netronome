---
title: Export results
description: Download speed test, packet loss, and DNS results as a CSV file or a JSON file.
sidebar:
  order: 4
---

An export is a download of the results of one dataset, as CSV or JSON. A result is one stored measurement: a speed test result, a packet loss result, or a DNS result. You can export all results, or only the results in a date range.

## Export from the web interface

1. Click **Settings**.
2. Select **Data**.
3. In **Export Results**, select the dataset in **Dataset**.
4. Select **CSV** or **JSON** in **Format**.
5. To set a start day, select it in **From**.
6. To set an end day, select it in **To**.
7. Click **Download**.

The browser saves one file, for example `netronome-speedtests-2026-10-17.csv`. The file name contains the dataset and the date of the export.

The days use the time zone of your browser. The file contains the results from the start of the **From** day to the end of the **To** day. If you leave a day empty, the range has no limit on that side.

If the export fails during the download, the browser shows the download as failed. Do not use a file from a failed download, because it is not complete.

## Export with a script

Send a `GET` request to this endpoint:

```text
/api/export/<dataset>?format=<format>&from=<timestamp>&to=<timestamp>
```

- `dataset`: `speedtests`, `packetloss`, or `dns`.
- `format`: `csv` or `json`. This parameter is necessary.
- `from` (optional): an RFC 3339 timestamp, for example `2026-10-17T00:00:00Z`. The export includes a result at this time.
- `to` (optional): an RFC 3339 timestamp. The export does not include a result at this time.

If a timestamp has a positive offset, write the `+` as `%2B`, for example `2026-10-17T00:00:00%2B02:00`. In a query string, a `+` that is not encoded is a space.

If Netronome uses a base URL, put the base URL in front of `/api`.

The endpoint needs authentication. A script must send the `session` cookie of a signed-in user, or it must run on an address in the IP whitelist. For more information, see [Authentication](/configuration/authentication/#ip-whitelist).

This example exports the DNS results of October 2026 as JSON:

```bash
curl -o dns.json "http://netronome.local:7575/api/export/dns?format=json&from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z"
```

If the dataset, the format, or a timestamp is not valid, the endpoint returns status 400 and a JSON object with an `error` message. If the export fails after the first byte, the server closes the connection before the end of the file.

## File format

CSV and JSON contain the same columns. A CSV file starts with a header row. A JSON file contains an array of objects, with one key for each column.

- The rows are in the order of `created_at`, from the oldest to the newest.
- The timestamps are RFC 3339 in UTC, for example `2026-10-17T08:30:00Z`.
- A missing value is an empty CSV cell or a JSON `null`.
- In a CSV file, a text cell that starts with `=`, `+`, `-`, `@`, a tab, or a carriage return starts with `'`. This prevents a spreadsheet from running the text as a formula. Number cells and JSON values do not change.

The columns stay the same after an update of Netronome. A new release can add a column. If a release renames or removes a column, or changes a unit or the order of the rows, the release notes tell you.

### Speed tests

| Column | Value |
| --- | --- |
| `id` | The ID of the result. |
| `created_at` | The time of the test. |
| `test_type` | The provider that ran the test: `speedtest`, `librespeed`, or `iperf3`. |
| `server_name` | The name of the speed test server. |
| `server_id` | The ID of the speed test server. |
| `server_host` | The host of the speed test server. |
| `server_city` | The city of the speed test server. |
| `download_mbps` | The download speed in Mbps. |
| `upload_mbps` | The upload speed in Mbps. |
| `latency_ms` | The latency in milliseconds. If Netronome cannot read the stored latency, the cell is empty. |
| `jitter_ms` | The jitter in milliseconds. |
| `is_scheduled` | `true` if a schedule ran the test, `false` if a user started it. |

### Packet loss

| Column | Value |
| --- | --- |
| `id` | The ID of the result. |
| `created_at` | The time of the check. |
| `monitor_id` | The ID of the packet loss monitor. |
| `monitor_name` | The name of the monitor. |
| `monitor_host` | The host that the monitor pings. |
| `packet_loss_percent` | The packet loss in percent. |
| `min_rtt_ms` | The shortest round-trip time in milliseconds. |
| `avg_rtt_ms` | The average round-trip time in milliseconds. |
| `max_rtt_ms` | The longest round-trip time in milliseconds. |
| `std_dev_rtt_ms` | The standard deviation of the round-trip time in milliseconds. |
| `packets_sent` | The number of packets that Netronome sent. |
| `packets_received` | The number of packets that came back. |
| `used_mtr` | `true` if the check used MTR. |
| `hop_count` | The number of hops that MTR found. |

### DNS

| Column | Value |
| --- | --- |
| `id` | The ID of the result. |
| `created_at` | The time of the check. |
| `monitor_id` | The ID of the DNS monitor. |
| `monitor_name` | The name of the monitor. |
| `resolver` | The resolver host of the monitor, with an optional port. |
| `protocol` | The protocol of the monitor: `udp`, `tcp`, or `dot`. |
| `query` | The domain name that the monitor asks for. |
| `record_type` | The record type that the monitor asks for, for example `A`. |
| `response_time_ms` | The response time in milliseconds. |
| `response_code` | The DNS response code, for example `NOERROR`. |
| `success` | `true` if the check succeeded. |
| `error` | The error message of a failed check. |

The `resolver`, `protocol`, `query`, and `record_type` columns show the configuration of the monitor at the time of the export.
