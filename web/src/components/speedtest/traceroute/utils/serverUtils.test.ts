/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { getTracerouteServerSelectionKey } from "./serverUtils.ts";

test("traceroute selections use the active catalogue key for their server type", () => {
  const speedtestKey = '["servers","speedtest","catalogue",{"global":true}]';

  assert.equal(
    getTracerouteServerSelectionKey({ isIperf: false, isLibrespeed: false }, speedtestKey),
    speedtestKey,
  );
  assert.equal(
    getTracerouteServerSelectionKey({ isIperf: true, isLibrespeed: false }, speedtestKey),
    "iperf",
  );
  assert.equal(
    getTracerouteServerSelectionKey({ isIperf: false, isLibrespeed: true }, speedtestKey),
    "librespeed",
  );
  assert.equal(getTracerouteServerSelectionKey(null, speedtestKey), speedtestKey);
});
