/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import {
  formatSpeedtestServerName,
  normalizeSpeedtestSettings,
  resolveServerReferences,
  selectedServersForKey,
  speedtestResultServerKey,
  speedtestSelectionKey,
  speedtestServerQuery,
} from "./speedtestSettings.ts";

test("coordinate settings require finite in-range coordinates", () => {
  assert.deepEqual(
    normalizeSpeedtestSettings({
      source: "coordinates",
      latitude: -27.4698,
      longitude: 153.0251,
      showServerCity: true,
    }),
    {
      source: "coordinates",
      latitude: -27.4698,
      longitude: 153.0251,
      showServerCity: true,
    },
  );
  assert.deepEqual(
    normalizeSpeedtestSettings({
      source: "coordinates",
      latitude: 91,
      longitude: 153.0251,
      showServerCity: true,
    }),
    { source: "local", showServerCity: true },
  );
});

test("server queries reflect the selected discovery source", () => {
  assert.deepEqual(speedtestServerQuery({ source: "local", showServerCity: false }), {});
  assert.deepEqual(
    speedtestServerQuery({ source: "global", showServerCity: false }),
    { global: true },
  );
  assert.deepEqual(
    speedtestServerQuery({
      source: "coordinates",
      latitude: 1.5,
      longitude: 2.5,
      showServerCity: false,
    }),
    { latitude: 1.5, longitude: 2.5 },
  );
});

test("changing catalogues invalidates the selection used by runs and schedules", () => {
  const localKey = speedtestSelectionKey({ source: "local", showServerCity: false });
  const globalKey = speedtestSelectionKey({ source: "global", showServerCity: false });
  const selection = { key: localKey, servers: [{ id: "123" }] };

  assert.deepEqual(selectedServersForKey(selection, localKey), [{ id: "123" }]);
  assert.deepEqual(selectedServersForKey(selection, globalKey), []);
});

test("saved schedule server IDs survive when the active catalogue cannot resolve them", () => {
  assert.deepEqual(
    resolveServerReferences(["global-123"], [{ id: "local-456", name: "Local" }]),
    [{ id: "global-123" }],
  );
});

test("schedule server resolution keeps colliding provider IDs distinct", () => {
  const collidingServers = [
    { id: "42", name: "Speedtest", isLibrespeed: false },
    { id: "42", name: "LibreSpeed", isLibrespeed: true },
  ];

  assert.deepEqual(
    resolveServerReferences(["42"], collidingServers, (server) => !server.isLibrespeed),
    [{ id: "42", server: collidingServers[0] }],
  );
  assert.deepEqual(
    resolveServerReferences(["42"], collidingServers, (server) => server.isLibrespeed),
    [{ id: "42", server: collidingServers[1] }],
  );
});

test("history labels and keys preserve distinct server identities", () => {
  assert.equal(formatSpeedtestServerName("Example", "Brisbane", true), "Example (Brisbane)");
  assert.equal(formatSpeedtestServerName("Example", "Brisbane", false), "Example");
  assert.equal(
    speedtestResultServerKey({
      testType: "speedtest",
      serverId: "123",
      serverHost: "host",
      serverName: "Example",
    }),
    "speedtest:123",
  );
  assert.equal(
    speedtestResultServerKey({
      testType: "speedtest",
      serverId: "",
      serverHost: "host",
      serverName: "Example",
    }),
    "speedtest:host",
  );
  assert.notEqual(
    speedtestResultServerKey({
      testType: "speedtest",
      serverId: "42",
      serverHost: "shared",
      serverName: "Shared",
    }),
    speedtestResultServerKey({
      testType: "librespeed",
      serverId: "42",
      serverHost: "shared",
      serverName: "Shared",
    }),
  );
});
