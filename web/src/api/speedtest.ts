/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";
import { SpeedTestOptions } from "@/types/speedtest";

export async function getServers(testType: string) {
  try {
    const response = await fetch(getApiUrl(`/servers?testType=${testType}`));
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch servers");
    }
    return await response.json();
  } catch (error) {
    console.error("Error fetching servers:", error);
    throw error;
  }
}

export async function getHistory(
  timeRange: string,
  page: number,
  limit: number
) {
  try {
    const response = await fetch(
      getApiUrl(
        `/speedtest/history?timeRange=${timeRange}&page=${page}&limit=${limit}`
      )
    );
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch history");
    }
    return await response.json();
  } catch (error) {
    console.error("Error fetching history:", error);
    throw error;
  }
}

export async function getSchedules() {
  try {
    const response = await fetch(getApiUrl("/schedules"));
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch schedules");
    }
    return await response.json();
  } catch (error) {
    console.error("Error fetching schedules:", error);
    throw error;
  }
}

export async function runSpeedTest(options: SpeedTestOptions) {
  try {
    const response = await fetch(getApiUrl("/speedtest"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(options),
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to run speed test");
    }
    return await response.json();
  } catch (error) {
    console.error("Error running speed test:", error);
    throw error;
  }
}

export async function getPublicHistory(
  timeRange: string,
  page: number,
  limit: number
) {
  try {
    const response = await fetch(
      getApiUrl(
        `/speedtest/public/history?timeRange=${timeRange}&page=${page}&limit=${limit}`
      )
    );
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch public history");
    }
    return await response.json();
  } catch (error) {
    console.error("Error fetching public history:", error);
    throw error;
  }
}

export async function getSpeedTestStatus() {
  try {
    const response = await fetch(getApiUrl("/speedtest/status"), {
      headers: {
        "Cache-Control": "no-cache",
        Pragma: "no-cache",
      },
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to get speed test status");
    }
    return await response.json();
  } catch (error) {
    console.error("Error getting speed test status:", error);
    throw error;
  }
}

export async function runTraceroute(host: string) {
  try {
    const response = await fetch(
      getApiUrl(`/traceroute?host=${encodeURIComponent(host)}`)
    );
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to run traceroute");
    }
    return await response.json();
  } catch (error) {
    console.error("Error running traceroute:", error);
    throw error;
  }
}

export async function getTracerouteStatus() {
  try {
    const response = await fetch(getApiUrl("/traceroute/status"));
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to get traceroute status");
    }
    return await response.json();
  } catch (error) {
    console.error("Error getting traceroute status:", error);
    throw error;
  }
}
