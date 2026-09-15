/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { speedtestServersNetworkOnlyRoute } from "../../vite.config.ts";

test("server catalogue requests bypass PWA runtime caches", () => {
  assert.equal(speedtestServersNetworkOnlyRoute.handler, "NetworkOnly");
  assert.match(
    "https://netronome.example/api/servers?testType=speedtest&refresh=true",
    speedtestServersNetworkOnlyRoute.urlPattern,
  );
  assert.match(
    "https://netronome.example/api/servers/catalogue/status?global=true",
    speedtestServersNetworkOnlyRoute.urlPattern,
  );
  assert.doesNotMatch(
    "https://netronome.example/api/server-status",
    speedtestServersNetworkOnlyRoute.urlPattern,
  );
});
