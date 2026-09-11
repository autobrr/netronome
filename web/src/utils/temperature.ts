/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { TemperatureStats } from "@/api/monitor";

type TemperatureLevel = "normal" | "warm" | "hot";

const SMART_LIMITS = { warm: 50, hot: 60 };
// NVMe temperature sensors 1-8 are vendor spot sensors on the controller die
// and the NAND. The drive applies its warning and critical limits to the
// composite value only, and a spot sensor 20-30 C above the composite is
// normal.
const NVME_SPOT_LIMITS = { warm: 85, hot: 95 };
const DEFAULT_LIMITS = { warm: 60, hot: 80 };

const isSmartSensor = (sensor: TemperatureStats): boolean => {
  const key = sensor.sensor_key.toLowerCase();
  const label = (sensor.label || "").toLowerCase();
  return (
    key.includes("smart_") || label.includes("hdd") || label.includes("ssd")
  );
};

// Some sensors report a useless limit: NVMe spot sensors report 65261.85 C,
// and board sensors report a limit that tracks the current reading.
const reportedLimit = (sensor: TemperatureStats): number | null =>
  sensor.critical && sensor.critical > 0 && sensor.critical < 1000
    ? sensor.critical
    : null;

export function temperatureLimits(sensor: TemperatureStats): {
  warm: number;
  hot: number;
} {
  const key = sensor.sensor_key.toLowerCase();

  if (isSmartSensor(sensor)) return SMART_LIMITS;
  if (key.startsWith("nvme") && key.includes("_sensor_"))
    return NVME_SPOT_LIMITS;

  if (key.startsWith("nvme") && key.endsWith("_composite")) {
    // The composite limit is the drive's own warning threshold, and it can
    // sit below the default limits.
    const limit = reportedLimit(sensor);
    if (limit !== null) return { warm: limit - 10, hot: limit };
  }

  return DEFAULT_LIMITS;
}

export function temperatureLevel(sensor: TemperatureStats): TemperatureLevel {
  const limits = temperatureLimits(sensor);
  if (sensor.temperature > limits.hot) return "hot";
  if (sensor.temperature > limits.warm) return "warm";
  return "normal";
}
