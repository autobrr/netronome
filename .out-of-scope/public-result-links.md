# Public result links for speed tests

Netronome does not give a public link to a speed test result on the web site of
the test provider.

## Why this is out of scope

The two providers cannot supply such a link.

**Speedtest.net.** Netronome measures with the `showwin/speedtest-go` library.
That library does not send the result to Ookla, so Ookla has no record of the
test and there is no page to link to. A link needs the official Ookla CLI, and
its licence does not permit us to put that binary in the Docker image. It also
asks the user to accept a licence at the first start, and it is a proprietary
binary for each architecture. The other method, to send the values to the
undocumented Ookla result endpoint from our own code, gives a page for results
that nobody verified, and it is against the terms of Ookla.

**LibreSpeed.** The `librespeed-cli` option `--share` sends the result to a
telemetry server and returns a link. With the default settings that server is
`https://librespeed.org`, with the paths `/results/telemetry.php` and
`/results/`. On 2026-08-30 both paths give HTTP 404. The site itself answers
with HTTP 200, but the telemetry service is not there:

```
POST https://librespeed.org/results/telemetry.php -> 404
GET  https://librespeed.org/results/             -> 404
```

A live test through Netronome with `share_results = true` measured correctly
(904.86 Mbit/s down, 389.55 Mbit/s up) and gave no link. The CLI wrote this to
stderr:

```
Error when sending telemetry data: server returned invalid response: File not found.
```

A link is possible only against a LibreSpeed instance that the user operates
with telemetry enabled. That needs `--telemetry-server` and the related options,
and a user interface for them. This is a large addition for a small group of
users, and it depends on a service that the LibreSpeed project appears to have
retired.

## If this comes back

Two events change the decision:

- The LibreSpeed project puts the telemetry service back, or names a new default
  endpoint. Then PR #216 becomes useful again with a small change.
- A user asks for a link from their own LibreSpeed instance. Then the work is the
  telemetry server settings, not the share flag.

## Prior requests

- #159: "FR: Add public link to speedtest results if server supports it"
- PR #216: "feat(speedtest): add result URL support for LibreSpeed"
