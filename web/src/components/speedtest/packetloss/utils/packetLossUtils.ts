/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { MTRData } from "../../../../types/types.ts";
import {
  convertUTCTimeToUser,
  convertUserTimeToUTC,
  getUserTimezone,
} from "../../../../utils/timeSettings.ts";

const EXACT_PREFIX = "exact:";

const storedExactTimes = (interval: string | undefined): string[] =>
  interval?.startsWith(EXACT_PREFIX)
    ? interval.substring(EXACT_PREFIX.length).split(",").map((t) => t.trim())
    : [];

/**
 * The stored UTC times of an `exact:` interval, as HH:MM in userTimezone.
 */
export const exactTimesFromInterval = (
  interval: string,
  userTimezone: string,
): string[] =>
  storedExactTimes(interval).map((time) => convertUTCTimeToUser(time, userTimezone));

/**
 * Builds the stored `exact:` interval (UTC) from HH:MM times in userTimezone.
 * A time that previousInterval already holds keeps its stored UTC value, so
 * an unchanged save does not move it when the offset changed since the load.
 */
export const exactIntervalFromTimes = (
  times: string[],
  userTimezone: string,
  previousInterval?: string,
): string => {
  const storedByShown = new Map(
    storedExactTimes(previousInterval).map((utc) => [
      convertUTCTimeToUser(utc, userTimezone),
      utc,
    ]),
  );
  const utcTimes = times.map(
    (time) => storedByShown.get(time) ?? convertUserTimeToUTC(time, userTimezone),
  );
  return `${EXACT_PREFIX}${utcTimes.join(",")}`;
};

export const formatInterval = (
  interval: number | string,
  userTimezone: string = getUserTimezone(),
): string => {
  // Handle string intervals
  if (typeof interval === "string") {
    // Handle exact time format
    if (interval.startsWith(EXACT_PREFIX)) {
      const times = storedExactTimes(interval);
      if (times.length === 1) {
        return `daily at ${convertUTCTimeToUser(times[0], userTimezone)}`;
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
