/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import valid from "semver/functions/valid.js";

export function formatVersion(version: string): string {
  if (version.startsWith("v") || valid(version) === null) return version;
  return `v${version}`;
}
