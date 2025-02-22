/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from '@/utils/baseUrl';
import { SpeedTestOptions } from '@/types/speedtest';

export async function getServers() {
  try {
    const response = await fetch(getApiUrl("/servers"))
    if (!response.ok) {
      throw new Error('Failed to fetch servers');
    }
    return await response.json();
  } catch (error) {
    console.error('Error fetching servers:', error);
    throw error;
  }
}

export async function getHistory(timeRange: string, page: number, limit: number) {
  try {
    const response = await fetch(
      getApiUrl(`/speedtest/history?timeRange=${timeRange}&page=${page}&limit=${limit}`)
    );
    if (!response.ok) {
      throw new Error('Failed to fetch history');
    }
    return await response.json();
  } catch (error) {
    console.error('Error fetching history:', error);
    throw error;
  }
}

export async function getSchedules() {
  try {
    const response = await fetch(getApiUrl("/schedules"))
    if (!response.ok) {
      throw new Error('Failed to fetch schedules');
    }
    return await response.json();
  } catch (error) {
    console.error('Error fetching schedules:', error);
    throw error;
  }
}

export async function runSpeedTest(options: SpeedTestOptions) {
  try {
    const response = await fetch(getApiUrl('/speedtest'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(options),
    });
    if (!response.ok) {
      throw new Error('Failed to run speed test');
    }
    return await response.json();
  } catch (error) {
    console.error('Error running speed test:', error);
    throw error;
  }
}

export async function getSpeedTestStatus() {
  try {
    const response = await fetch(getApiUrl("/speedtest/status"), {
      headers: {
        'Cache-Control': 'no-cache',
        'Pragma': 'no-cache'
      }
    });
    if (!response.ok) {
      throw new Error('Failed to get speed test status');
    }
    return await response.json();
  } catch (error) {
    console.error('Error getting speed test status:', error);
    throw error;
  }
}
