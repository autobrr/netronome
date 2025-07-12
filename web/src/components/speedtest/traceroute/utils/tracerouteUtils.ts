/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { TracerouteHop } from "@/types/types";

/**
 * Utility function to extract hostname from any server host value
 * Handles URLs, hostnames with ports, and plain hostnames/IPs
 */
export const extractHostname = (hostValue: string): string => {
  let hostname = hostValue;

  // Extract hostname from URL if it's a full URL (LibreSpeed servers)
  if (hostname.startsWith("http://") || hostname.startsWith("https://")) {
    try {
      const url = new URL(hostname);
      hostname = url.hostname;
    } catch {
      console.warn("Failed to parse URL:", hostname);
    }
  }

  // Strip port from hostname if present (iperf3 and other servers)
  if (hostname.includes(":")) {
    hostname = hostname.split(":")[0];
  }

  return hostname;
};

/**
 * Format RTT value for display
 * Returns "*" for zero values (timeouts), otherwise formats as "X.Xms"
 */
export const formatRTT = (rtt: number): string => {
  if (rtt === 0) return "*";
  return `${rtt.toFixed(1)}ms`;
};

/**
 * Calculate average RTT from three RTT measurements
 * Filters out invalid (zero) RTT values and returns average of valid ones
 */
export const getAverageRTT = (hop: {
  rtt1: number;
  rtt2: number;
  rtt3: number;
  timeout: boolean;
}): number => {
  if (hop.timeout) return 0;
  const validRTTs = [hop.rtt1, hop.rtt2, hop.rtt3].filter((rtt) => rtt > 0);
  if (validRTTs.length === 0) return 0;
  return validRTTs.reduce((sum, rtt) => sum + rtt, 0) / validRTTs.length;
};

/**
 * Get CSS color classes for RTT display based on RTT column
 */
export const getRTTColorClass = (
  timeout: boolean,
  rttColumn: "rtt1" | "rtt2" | "rtt3" | "average",
): string => {
  if (timeout) return "text-gray-500";

  switch (rttColumn) {
    case "rtt1":
      return "text-emerald-600 dark:text-emerald-400";
    case "rtt2":
      return "text-yellow-600 dark:text-yellow-400";
    case "rtt3":
      return "text-orange-600 dark:text-orange-400";
    case "average":
      return "text-gray-700 dark:text-gray-300";
    default:
      return "text-gray-700 dark:text-gray-300";
  }
};

/**
 * Format hop data for display in table/cards
 */
export const formatHopData = (hop: TracerouteHop) => {
  return {
    number: hop.number,
    host: hop.timeout ? "Timeout" : hop.host,
    provider: hop.as || "—",
    countryCode: hop.countryCode,
    location: hop.location,
    rtt1: formatRTT(hop.rtt1),
    rtt2: formatRTT(hop.rtt2),
    rtt3: formatRTT(hop.rtt3),
    average: hop.timeout ? "*" : formatRTT(getAverageRTT(hop)),
    timeout: hop.timeout,
  };
};
