/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";
import { SpeedTestOptions } from "@/types/speedtest";
import type { Server } from "@/types/types";
import type { SpeedtestServerQuery } from "@/utils/speedtestSettings";

/** Durable fetch metadata for one Speedtest.net discovery source. */
export interface SpeedtestServerCatalogueStatus {
  stored: boolean;
  updatedAt?: string;
}

const addSpeedtestServerQuery = (params: URLSearchParams, query: SpeedtestServerQuery) => {
  if (query.global) {
    params.set("global", "true");
  }
  if (query.latitude !== undefined && query.longitude !== undefined) {
    params.set("latitude", query.latitude.toString());
    params.set("longitude", query.longitude.toString());
  }
};

/** Fetches the selected provider's servers and supports abortable, cache-bypassing Speedtest.net discovery. */
export async function getServers(
  testType: string,
  query: SpeedtestServerQuery = {},
  signal?: AbortSignal,
): Promise<Server[]> {
  try {
    const params = new URLSearchParams({ testType });
    addSpeedtestServerQuery(params, query);
    if (query.refresh) {
      params.set("refresh", "true");
    }
    const response = await fetch(getApiUrl(`/servers?${params.toString()}`), { signal });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch servers");
    }
    return await response.json() as Server[];
  } catch (error) {
    console.error("Error fetching servers:", error);
    throw error;
  }
}

/** Reads whether the selected source has been durably fetched without starting discovery. */
export async function getSpeedtestServerCatalogueStatus(
  query: SpeedtestServerQuery,
  signal?: AbortSignal,
): Promise<SpeedtestServerCatalogueStatus> {
  const params = new URLSearchParams();
  addSpeedtestServerQuery(params, query);
  const response = await fetch(
    getApiUrl(`/servers/catalogue/status?${params.toString()}`),
    { signal },
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || "Failed to get server catalogue status");
  }

  const data: unknown = await response.json();
  if (typeof data !== "object" || data === null || !("stored" in data) || typeof data.stored !== "boolean") {
    throw new Error("Invalid server catalogue status response");
  }
  if ("updatedAt" in data && data.updatedAt !== undefined && typeof data.updatedAt !== "string") {
    throw new Error("Invalid server catalogue status response");
  }
  return {
    stored: data.stored,
    updatedAt: "updatedAt" in data ? data.updatedAt : undefined,
  };
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
