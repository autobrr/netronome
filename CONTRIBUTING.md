# Contributing to Netronome

Thanks for considering a contribution. I maintain this in spare time, so clear,
focused PRs help a lot. Please skim this before opening an issue or PR — it saves
us both time. ❤️

> [!NOTE]
> I’m not trying to be difficult. I just want to keep the project healthy and
> reviewable.

## AI Usage

AI assistance is allowed, but the PR must be carefully reviewed and edited by a
human. If a change “smells AI‑generated” (broad refactors, unrelated edits,
drifting scope), it will be closed. Keep diffs tight, explain intent, and verify
behavior.

## Quick Guide

If you want to contribute:
- Pick a clear, scoped issue.
- Keep the change focused on the issue.
- Run relevant tests.
- Open a PR with a short rationale and how you tested.

## Scope Rules

Please avoid:
- Unrelated refactors or file cleanup.
- Reformatting or renaming that is not required for the change.
- Mass edits that weren’t discussed first.

## Package Manager

We use `pnpm`. Do not add `package-lock.json` or switch tooling.

## Build & Tests

Run the appropriate checks for what you changed:
- `pnpm -C web lint`
- `pnpm -C web build`
- Add other commands as applicable.

If you didn’t run tests, say why in the PR.

## Pull Requests

PRs should include:
- Summary of what changed and why.
- How it was tested.
- Screenshots for UI changes (if applicable).
