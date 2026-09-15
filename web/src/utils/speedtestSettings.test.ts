/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { MutationObserver, QueryClient } from "@tanstack/react-query";
import {
  formatSpeedtestServerName,
  normalizeSpeedtestSettings,
  resolveServerReferences,
  selectedServersForKey,
  speedtestResultServerKey,
  speedtestSelectionKey,
  speedtestServerQueryKey,
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

test("refresh requests do not deduplicate onto ordinary catalogue loads", async () => {
  const queryClient = new QueryClient();
  const calls: string[] = [];
  let markNormalStarted!: () => void;
  let releaseNormal!: () => void;
  const normalStarted = new Promise<void>((resolve) => {
    markNormalStarted = resolve;
  });
  const normalBlocked = new Promise<void>((resolve) => {
    releaseNormal = resolve;
  });

  const normalRequest = queryClient.fetchQuery({
    queryKey: speedtestServerQueryKey({}),
    queryFn: async () => {
      calls.push("normal");
      markNormalStarted();
      await normalBlocked;
      return ["cached"];
    },
  });
  await normalStarted;

  const refreshMutation = new MutationObserver(queryClient, {
    mutationFn: async () => {
      calls.push("refresh");
      return ["fresh"];
    },
  });
  const refreshRequest = refreshMutation.mutate();
  releaseNormal();

  const [, refreshed] = await Promise.all([normalRequest, refreshRequest]);
  assert.deepEqual(calls, ["normal", "refresh"]);
  assert.deepEqual(refreshed, ["fresh"]);
});

test("cancelled refreshes do not republish prior mutation data", async () => {
  const queryClient = new QueryClient();
  const catalogueQueryKey = speedtestServerQueryKey({});
  let successfulRefreshes = 0;
  const refreshMutation = new MutationObserver<
    string[],
    Error,
    { signal: AbortSignal; result?: string[] }
  >(queryClient, {
    mutationFn: async ({ signal, result }) => {
      if (result) return result;
      await new Promise<void>((_resolve, reject) => {
        if (signal.aborted) {
          reject(new Error("refresh aborted"));
          return;
        }
        signal.addEventListener("abort", () => reject(new Error("refresh aborted")), {
          once: true,
        });
      });
      return [];
    },
    onSuccess: (servers) => {
      successfulRefreshes++;
      queryClient.setQueryData(catalogueQueryKey, servers);
    },
  });

  await refreshMutation.mutate({ signal: new AbortController().signal, result: ["old"] });
  queryClient.setQueryData(catalogueQueryKey, ["current"]);

  const controller = new AbortController();
  const cancelledRefresh = refreshMutation.mutate({ signal: controller.signal });
  controller.abort();

  await assert.rejects(cancelledRefresh, /refresh aborted/);
  assert.deepEqual(queryClient.getQueryData(catalogueQueryKey), ["current"]);
  assert.equal(successfulRefreshes, 1);
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
