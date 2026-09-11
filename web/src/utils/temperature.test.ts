/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import type { TemperatureStats } from "../api/monitor.ts";
import { temperatureLevel } from "./temperature.ts";

// Readings from a healthy Proxmox host. smartctl reported zero seconds above
// the warning threshold for both NVMe drives at these values.
const healthy: TemperatureStats[] = [
  { sensor_key: "acpitz", temperature: 27.8 },
  { sensor_key: "r8169_0_400:00", temperature: 39.5, critical: 120 },
  { sensor_key: "nvme_composite", temperature: 39.85, critical: 83.85 },
  { sensor_key: "nvme_sensor_2", temperature: 61.85 },
  { sensor_key: "nvme_composite", temperature: 41.85, critical: 74.85 },
  { sensor_key: "nvme_sensor_1", temperature: 76.8, critical: 65261.85 },
  { sensor_key: "nvme_sensor_2", temperature: 72.85, critical: 65261.85 },
  { sensor_key: "coretemp_package_id_0", temperature: 35, critical: 80 },
  { sensor_key: "nct6687_system", temperature: 38, critical: 39 },
  { sensor_key: "nct6687_m2_1", temperature: 32, critical: 32 },
];

test("a healthy host raises no temperature alert", () => {
  for (const sensor of healthy) {
    assert.equal(
      temperatureLevel(sensor),
      "normal",
      `${sensor.sensor_key} at ${sensor.temperature}C`
    );
  }
});

test("hot sensors still alert", () => {
  const cases: Array<[TemperatureStats, string]> = [
    // Above the drive's own composite warning threshold.
    [{ sensor_key: "nvme_composite", temperature: 78, critical: 74.85 }, "hot"],
    [{ sensor_key: "nvme_sensor_1", temperature: 96, critical: 65261.85 }, "hot"],
    [{ sensor_key: "coretemp_core_0", temperature: 92, critical: 80 }, "hot"],
    [{ sensor_key: "smart_sda", label: "WDC (SDA)", temperature: 55 }, "warm"],
  ];

  for (const [sensor, want] of cases) {
    assert.equal(temperatureLevel(sensor), want, sensor.sensor_key);
  }
});
