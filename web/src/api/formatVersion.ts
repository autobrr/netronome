/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

const releaseNumber = /^v?\d+\.\d+\.\d+$/;

export function formatVersion(version: string): string {
  if (!releaseNumber.test(version) || version.startsWith("v")) return version;
  return `v${version}`;
}
