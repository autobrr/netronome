/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { formatVersion } from "./formatVersion.ts";

test("release numbers show with a v prefix", () => {
  assert.equal(formatVersion("v0.10.0"), "v0.10.0");
  assert.equal(formatVersion("0.10.0"), "v0.10.0");
});

test("other build values show as reported", () => {
  assert.equal(formatVersion("dev"), "dev");
  assert.equal(formatVersion("pr-352"), "pr-352");
  assert.equal(formatVersion("(devel)"), "(devel)");
  assert.equal(
    formatVersion("v0.9.0-3-g336f529-dirty"),
    "v0.9.0-3-g336f529-dirty",
  );
  assert.equal(formatVersion("0.10.0-rc.1"), "0.10.0-rc.1");
  assert.equal(formatVersion("336f529"), "336f529");
});
