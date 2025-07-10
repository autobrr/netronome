/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { MTRData } from "@/types/types";

export const formatInterval = (seconds: number): string => {
  if (seconds < 60) {
    return `${seconds} second${seconds !== 1 ? "s" : ""}`;
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes} minute${minutes !== 1 ? "s" : ""}`;
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    return `${hours} hour${hours !== 1 ? "s" : ""}`;
  } else {
    const days = Math.floor(seconds / 86400);
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
