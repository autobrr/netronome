# Shoutrrr fork migration: follow-ups

Netronome moved from `containrrr/shoutrrr v0.8.0` to `nicholas-fedor/shoutrrr v0.16.3` in
issue #126. This file tracks the work that the migration PR did not include.

## Follow-ups

- **ntfy shim removal.** The shim (`internal/notifications/ntfy.go`) survives for one reason:
  it logs and ignores query keys that it cannot parse, while the fork errors on the first bad
  key. Stored URLs predate this parsing and must continue to work. Removal needs a
  URL-sanitizing migration first. The migrator is count-based, so the migration needs a
  strictly-higher number.
- **Shim gap:** `parseNtfyURL` accepts the `disabletls` key. `sendNtfy` honors it now. If the
  fork adds more keys with send-time behavior, the shim must mirror them.
- **Upstream PR:** the fork discards the service error in `initService`
  (`pkg/router/router.go`). A PR that adds `%w` would make our direct-init workaround in
  `ValidateNotificationURL` unnecessary. Not filed yet, and we found no existing issue.
- **qui:** bump `autobrr/qui` from fork v0.16.1 to v0.16.2 or later. That version adds
  HTTP-client injection (PR #1058).
- **autobrr:** `autobrr/autobrr` is still on containrrr v0.8.0. It is a second migration
  candidate.
- **Allowlist asymmetry:** the allowlist in
  `web/src/components/settings/notifications/ChannelDetails.tsx` blocks *edits* of channels
  with unlisted schemes, but free-text *adds* work. Decide whether to relax it.

## Open questions

1. Do any users have stored `teams://` channels? They break silently after the migration.
   The release notes carry the callout. Ask in Discord if reports come in.
2. The matrix `Initialize` performs a network login in both library versions. Because of
   this, `ValidateNotificationURL` makes a blocking outbound request that an attacker can
   influence. This is not a regression, but we did not confirm a rate limit on that route.
   Examine it separately.
