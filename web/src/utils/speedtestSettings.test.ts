/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import {
  findScheduleServer,
  formatSpeedtestServerName,
  formatSpeedtestServerStorageStatus,
  isLatitude,
  isLongitude,
  normalizeSpeedtestSettings,
  speedtestResultServerKey,
  speedtestServerQueryKey,
  speedtestServerSelectionKey,
  speedtestServerStatusQueryKey,
  speedtestServerQuery,
} from "./speedtestSettings.ts";

test("coordinate settings require finite in-range coordinates", () => {
  assert.equal(isLatitude(-90), true);
  assert.equal(isLatitude(91), false);
  assert.equal(isLongitude(180), true);
  assert.equal(isLongitude(Number.NaN), false);
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
  assert.equal(
    speedtestServerSelectionKey({ global: true }),
    JSON.stringify(speedtestServerQueryKey({ global: true })),
  );
  assert.notDeepEqual(
    speedtestServerQueryKey({}),
    speedtestServerQueryKey({ latitude: 1, longitude: 2 }),
  );
});

test("server status queries distinguish discovery sources and coordinates", () => {
  assert.deepEqual(
    speedtestServerStatusQueryKey({ source: "local", showServerCity: false }),
    ["servers", "speedtest", "status", "local"],
  );
  assert.deepEqual(
    speedtestServerStatusQueryKey({ source: "global", showServerCity: false }),
    ["servers", "speedtest", "status", "global"],
  );
  assert.notDeepEqual(
    speedtestServerStatusQueryKey({
      source: "coordinates",
      latitude: 1,
      longitude: 2,
      showServerCity: false,
    }),
    speedtestServerStatusQueryKey({
      source: "coordinates",
      latitude: 3,
      longitude: 4,
      showServerCity: false,
    }),
  );
});

test("server storage status keeps unavailable distinct from not stored", () => {
  assert.equal(
    formatSpeedtestServerStorageStatus("Global", {
      stored: undefined,
      isLoading: false,
      isError: true,
    }),
    "Stored server status unavailable",
  );
  assert.equal(
    formatSpeedtestServerStorageStatus("Global", {
      stored: false,
      isLoading: false,
      isError: false,
    }),
    "No global servers stored yet",
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

test("schedule lookup respects the saved LibreSpeed catalogue source", () => {
  const baseServer = {
    id: "1",
    name: "",
    host: "",
    location: "",
    distance: 0,
    country: "",
    sponsor: "",
    latitude: 0,
    longitude: 0,
    isIperf: false,
    isLibrespeed: true,
  };
  const servers = [
    { ...baseServer, name: "Custom", isPublic: false },
    { ...baseServer, name: "Public", isPublic: true },
  ];
  const options = {
    enableDownload: true,
    enableUpload: true,
    serverIds: ["1"],
    useIperf: false,
    useLibrespeed: true,
    serverHost: undefined,
    isPublicServer: true,
  };

  assert.equal(findScheduleServer(servers, "1", options)?.name, "Public");
  assert.equal(
    findScheduleServer(servers, "1", { ...options, isPublicServer: false })?.name,
    "Custom",
  );
});
