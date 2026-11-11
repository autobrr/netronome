/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { afterEach, beforeEach, mock, test } from "node:test";
import {
  exactIntervalFromTimes,
  exactTimesFromInterval,
  formatInterval,
} from "./packetLossUtils.ts";

const initialTZ = process.env.TZ;

beforeEach(() => {
  process.env.TZ = "Europe/Oslo";
  mock.timers.enable({ apis: ["Date"], now: new Date("2026-07-01T10:00:00Z") });
});

afterEach(() => {
  mock.timers.reset();
  process.env.TZ = initialTZ;
});

test("a stored UTC exact time shows in the time zone of the user", () => {
  assert.equal(formatInterval("exact:12:00", "Europe/Oslo"), "daily at 14:00");
});

test("several exact times show as a count", () => {
  assert.equal(formatInterval("exact:12:00,23:00", "Europe/Oslo"), "daily at 2 times");
});

test("a duration interval does not change", () => {
  assert.equal(formatInterval("5m", "Europe/Oslo"), "5 minutes");
});

test("an edit form shows the stored UTC exact times in the time zone of the user", () => {
  assert.deepEqual(exactTimesFromInterval("exact:12:00, 23:00", "Europe/Oslo"), ["14:00", "01:00"]);
});

test("a saved form stores the exact times in UTC", () => {
  assert.equal(exactIntervalFromTimes(["14:00", "01:00"], "Europe/Oslo"), "exact:12:00,23:00");
});

test("an unchanged save keeps the stored times across a daylight saving change", () => {
  process.env.TZ = "America/New_York";
  mock.timers.setTime(new Date("2026-11-02T01:00:00Z").getTime());
  const stored = "exact:05:00";
  const shown = exactTimesFromInterval(stored, "America/New_York");
  assert.equal(exactIntervalFromTimes(shown, "America/New_York", stored), stored);
});

test("a new time in an edited monitor is converted to UTC", () => {
  assert.equal(
    exactIntervalFromTimes(["14:00", "15:00"], "Europe/Oslo", "exact:12:00"),
    "exact:12:00,13:00",
  );
});
