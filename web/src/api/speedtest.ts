/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Server, SpeedTestResult, Schedule, TestOptions, SpeedUpdate, TimeRange, PaginatedResponse } from '../types/types'

export const fetchServers = async (): Promise<Server[]> => {
  const response = await fetch("/api/servers")
  if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
  const text = await response.text()
  try {
    return JSON.parse(text)
  } catch (parseError) {
    throw new Error(`Invalid JSON response (${(parseError as Error).message}): ${text.substring(0, 100)}...`)
  }
}

export const fetchHistory = async (
  timeRange: TimeRange,
  page: number,
  limit: number = 500
): Promise<PaginatedResponse<SpeedTestResult>> => {
  const response = await fetch(
    `/api/speedtest/history?timeRange=${timeRange}&page=${page}&limit=${limit}`
  );
  return response.json();
}

export const fetchSchedules = async (): Promise<Schedule[]> => {
  const response = await fetch("/api/schedules")
  if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
  const data = await response.json()
  return data || []
}

export const runSpeedTest = async (options: TestOptions): Promise<SpeedTestResult> => {
  try {
    const response = await fetch('/api/speedtest', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(options),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Server error: ${errorText}`);
    }

    const data = await response.json();
    if (!data) {
      throw new Error('Empty response from server');
    }

    return data;
  } catch (error) {
    console.error('Speed test error:', error);
    throw error;
  }
};

export const fetchTestStatus = async (): Promise<SpeedUpdate> => {
  const response = await fetch("/api/speedtest/status", {
    headers: {
      "Cache-Control": "no-cache",
      Pragma: "no-cache",
    },
    credentials: "same-origin",
    cache: "no-store",
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  const contentType = response.headers.get("content-type")
  if (!contentType?.includes("application/json")) {
    throw new Error("Invalid content type")
  }

  return response.json()
}
