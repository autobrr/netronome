/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { MTRData } from "@/types/types";

export const formatInterval = (interval: number | string): string => {
  // Handle string intervals
  if (typeof interval === "string") {
    // Handle exact time format
    if (interval.startsWith("exact:")) {
      const times = interval.substring(6).split(",");
      if (times.length === 1) {
        return `daily at ${times[0]}`;
      } else {
        return `daily at ${times.length} times`;
      }
    }

    // Handle duration format (e.g., "1m", "1h", "1d")
    const match = interval.match(/^(\d+)([smhd])$/);
    if (match) {
      const value = parseInt(match[1]);
      const unit = match[2];
      switch (unit) {
        case "s":
          return `${value} second${value !== 1 ? "s" : ""}`;
        case "m":
          return `${value} minute${value !== 1 ? "s" : ""}`;
        case "h":
          return `${value} hour${value !== 1 ? "s" : ""}`;
        case "d":
          return `${value} day${value !== 1 ? "s" : ""}`;
      }
    }

    // Fallback for unknown string format
    return interval;
  }

  // Handle numeric intervals (legacy support)
  if (interval < 60) {
    return `${interval} second${interval !== 1 ? "s" : ""}`;
  } else if (interval < 3600) {
    const minutes = Math.floor(interval / 60);
    return `${minutes} minute${minutes !== 1 ? "s" : ""}`;
  } else if (interval < 86400) {
    const hours = Math.floor(interval / 3600);
    return `${hours} hour${hours !== 1 ? "s" : ""}`;
  } else {
    const days = Math.floor(interval / 86400);
    return `${days} day${days !== 1 ? "s" : ""}`;
  }
};

export const formatRTT = (rtt: number): string => {
  if (rtt === 0) return "0";
  return rtt.toFixed(1);
};

export const parseMTRData = (
  mtrDataStr: string | undefined,
): MTRData | null => {
  if (!mtrDataStr) return null;
  try {
    return JSON.parse(mtrDataStr);
  } catch (e) {
    console.error("Failed to parse MTR data:", e);
    return null;
  }
};
