# ADR-0002: The checksums asset has a fixed name

Date: 2026-09-01
Status: accepted
PR: #277

## Context

`netronome update` replaces the binary of the agent or the server with the latest
release. The `go-selfupdate` library downloads the archive, then reads a checksums
file from the same release to make sure that the archive is correct.

The library needs the name of the checksums asset before it looks at the release.
`DetectLatest` records the ID of that asset only when the updater already holds a
validator, and the validator holds the name. A release found without a validator
keeps the value `-1` for the ID, and the download of the checksums then asks GitHub
for asset `-1`, which always answers 404.

The name of the checksums asset was `netronome_<version>_checksums.txt`. The version
is not known before `DetectLatest`, so the correct name was not known before the
updater had to be built. Every self-update failed for this reason.

## Decision

**The checksums asset is `checksums.txt`. The name does not contain the version.**
`checksum.name_template` in `.config/goreleaser.yml` sets this name.

**`cmd/netronome/update.go` writes the same name as a constant** in the
`ChecksumValidator` of the one updater that it builds. With a name that does not
change, the updater holds the validator from the start, and one `DetectLatest` gives
a release that `UpdateTo` can download and check.

The two files must always hold the same string. Nothing in the build finds a
difference between them: a change to one of the two makes the self-update fail on the
machine of the user, after the release, with a message about a validation asset that
the release does not contain. Change the two files together.

## Consequences

The name of the checksums asset in a release is different from the names in the
releases before v0.14.0. A person or a script that downloads
`netronome_<version>_checksums.txt` must use `checksums.txt` for the new releases.
The old releases keep their old names.

An installation from a release before this change cannot update itself. Its binary
looks for the old name, which the new releases do not contain, and it failed with the
`-1` fault before this change. Such an installation needs one manual installation of
a new binary. After that, `netronome update` works.

This decision does not stop a move to `Updater.UpdateSelf`, which does the detection,
the comparison of versions and the update in one call. `UpdateSelf` reads the version
of the running binary as a semantic version, and a development build has the version
`dev`, which is not one. That change is possible, but it is not part of this decision.
