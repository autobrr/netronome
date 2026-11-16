/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { formatIperfServerEndpoint } from "./iperfEndpoint.ts";

test("joins an IPv4 host and port", () => {
  assert.equal(formatIperfServerEndpoint("192.0.2.10", 5201), "192.0.2.10:5201");
  assert.equal(formatIperfServerEndpoint("192.0.2.10", "5203"), "192.0.2.10:5203");
});

test("joins a hostname and port", () => {
  assert.equal(
    formatIperfServerEndpoint("iperf.example.com", 5201),
    "iperf.example.com:5201",
  );
});

test("brackets a saved IPv6 host so the port stays separate", () => {
  assert.equal(
    formatIperfServerEndpoint("2001:db8::1", 5203),
    "[2001:db8::1]:5203",
  );
  assert.equal(formatIperfServerEndpoint("::1", 5201), "[::1]:5201");
  assert.equal(
    formatIperfServerEndpoint("::ffff:192.0.2.1", 5201),
    "[::ffff:192.0.2.1]:5201",
  );
});

test("does not double bracket an already bracketed IPv6 host", () => {
  assert.equal(
    formatIperfServerEndpoint("[2001:db8::1]", 5203),
    "[2001:db8::1]:5203",
  );
});

test("trims spaces around the host", () => {
  assert.equal(
    formatIperfServerEndpoint("  2001:db8::1  ", 5203),
    "[2001:db8::1]:5203",
  );
});
