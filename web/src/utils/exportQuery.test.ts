/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { exportQuery } from "./exportQuery.ts";

process.env.TZ = "Europe/Oslo";

test("empty dates export everything", () => {
  assert.equal(exportQuery("csv", "", ""), "format=csv");
});

test("a picked day starts and ends at local midnight", () => {
  const query = new URLSearchParams(exportQuery("json", "2026-10-17", "2026-10-17") ?? "");
  assert.equal(query.get("format"), "json");
  assert.equal(query.get("from"), "2026-10-16T22:00:00.000Z");
  assert.equal(query.get("to"), "2026-10-17T22:00:00.000Z");
});

test("the end day keeps its length across a daylight saving change", () => {
  // Norway leaves summer time on October 25, 2026, so that day has 25 hours.
  const query = new URLSearchParams(exportQuery("csv", "", "2026-10-25") ?? "");
  assert.equal(query.get("from"), null);
  assert.equal(query.get("to"), "2026-10-25T23:00:00.000Z");
});

test("the end day rolls over into the next month", () => {
  const query = new URLSearchParams(exportQuery("csv", "", "2026-10-31") ?? "");
  assert.equal(query.get("to"), "2026-10-31T23:00:00.000Z");
});

test("a year below 100 stays that year", () => {
  const query = new URLSearchParams(exportQuery("csv", "0050-06-01", "") ?? "");
  assert.match(query.get("from") ?? "", /^0050-05-31T/);
});

test("a day that the server cannot read gives no query", () => {
  // Date inputs accept years up to 275760, and Go reads only four-digit years.
  assert.equal(exportQuery("csv", "10000-01-01", ""), null);
  assert.equal(exportQuery("csv", "", "275760-09-13"), null);
});
