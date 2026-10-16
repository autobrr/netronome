---
title: DNS monitoring
description: Send a DNS query to a resolver on a schedule and record its response time and response code.
sidebar:
  order: 5
---

A DNS monitor sends one DNS query to a resolver on a schedule. A resolver is the DNS server that answers the query, for example your router, a Pi-hole, or `1.1.1.1`. Netronome records the response time and the response code of each check, and it tells you when the resolver stops answering.

A DNS monitor measures the resolver. Netronome does not compare the answer with an expected value.

You manage DNS monitors on the **DNS** tab of the web interface. DNS monitoring needs no configuration file settings.

## Create a monitor

1. Open the **DNS** tab.
2. Click **Add**. The **New DNS Monitor** dialog opens.
3. Optional: click a preset to fill in the resolver and the protocol. The presets are Cloudflare, Google, Quad9, and OpenDNS, each over UDP and over DNS over TLS.
4. In **Resolver**, type the IP address or the host name of the resolver. To use a port that is not the default, add `:port`, for example `dns.example.com:5353`.
5. In **Name**, type a name for the monitor. If you selected a preset and leave the field empty, Netronome uses the name of the preset.
6. In **Protocol**, select **UDP**, **TCP**, or **DNS over TLS**.
7. In **Query**, type the domain name to ask for. The default is `google.com`.
8. In **Record Type**, select `A`, `AAAA`, `CNAME`, `MX`, `NS`, or `TXT`. The default is `A`.
9. In **Check Interval**, select the time between two checks, from 1 minute to 24 hours. The default is 1 minute.
10. Click **Create Monitor**.

A new monitor runs its first check at the next minute of the scheduler. For more information, see [Scheduling](/configuration/scheduling/).

## Protocols and ports

| Protocol | Default port |
|---|---|
| UDP | 53 |
| TCP | 53 |
| DNS over TLS | 853 |

For DNS over TLS, Netronome makes sure that the TLS certificate of the resolver matches the host part of **Resolver**. If the certificate does not match, the check fails. Type the host name of the resolver, for example `dns.google`. An IP address works only if the certificate contains that address.

## Failed checks

Each check has a time limit of 5 seconds. A check fails in these conditions:

- The resolver does not answer in 5 seconds.
- The connection fails, for example because of a network error or a TLS error.
- The resolver answers with an error code, for example `SERVFAIL`, `REFUSED`, or `NXDOMAIN`.

A check is successful only when the response code is `NOERROR`.

:::note
A query for a domain that does not exist gives `NXDOMAIN`, and the check fails. Use a domain that exists in **Query**.
:::

## Monitor states

Netronome gives each monitor a state after each check:

- OK: The check is successful, and the monitor was not down.
- Down: The check failed.
- Recovered: The first successful check after the monitor was down. After the next successful check, the state goes back to OK.

A monitor that has not run yet has no state.

## Notifications

Netronome sends a notification when a monitor goes down and when it recovers. It does not send a notification again while the monitor stays down. The notification contains the name of the monitor, the resolver, and the error or the response time.

To select these notifications, use the DNS category on the [Notifications](/configuration/notifications/) page.

## Monitor details

Click a monitor to see its details:

- Latency: The response time of the last check, in milliseconds.
- Code: The response code of the last check, for example `NOERROR` or `SERVFAIL`.
- State: OK, Down, or Recovered.
- Latency History: A chart of the response time of the last checks. A failed check leaves a gap in the line.

To delete old DNS results, see [Delete old results](/configuration/database/#delete-old-results).
