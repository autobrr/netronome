/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { beforeEach, afterEach, mock, test } from "node:test";
import {
  convertUTCTimeToUser,
  convertUserTimeToUTC,
  formatUTCTimeOfDay,
} from "./timeSettings.ts";

process.env.TZ = "Europe/Oslo";

beforeEach(() => {
  mock.timers.enable({ apis: ["Date"], now: new Date("2026-07-01T10:00:00Z") });
});

afterEach(() => {
  mock.timers.reset();
});

test("a UTC time shows in the summer time of Oslo", () => {
  assert.equal(convertUTCTimeToUser("12:00", "Europe/Oslo"), "14:00");
});

test("a UTC time late in the day moves to the next day in Oslo", () => {
  assert.equal(convertUTCTimeToUser("23:00", "Europe/Oslo"), "01:00");
});

test("a time in a zone west of UTC moves to the previous day", () => {
  assert.equal(convertUTCTimeToUser("02:30", "America/New_York"), "22:30");
});

test("a half-hour offset keeps the minutes", () => {
  assert.equal(convertUTCTimeToUser("08:30", "Asia/Kolkata"), "14:00");
});

test("a time saved in Oslo comes back unchanged", () => {
  for (const time of ["00:00", "01:00", "14:00", "23:30"]) {
    const utc = convertUserTimeToUTC(time, "Europe/Oslo");
    assert.equal(convertUTCTimeToUser(utc, "Europe/Oslo"), time);
  }
});

test("01:00 in Oslo summer time is stored as 23:00 UTC", () => {
  assert.equal(convertUserTimeToUTC("01:00", "Europe/Oslo"), "23:00");
});

test("a stored UTC time shows in the time zone setting", () => {
  assert.equal(
    formatUTCTimeOfDay("12:00", { timezone: "Europe/Oslo", use24HourFormat: true }),
    "14:00",
  );
});

test("a stored UTC time shows in the browser time zone when the setting is auto", () => {
  assert.equal(
    formatUTCTimeOfDay("12:00", { timezone: "auto", use24HourFormat: true }),
    "14:00",
  );
});
