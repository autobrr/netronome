/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Centralized refresh intervals for monitor data fetching
 * All intervals are in milliseconds
 */
export const MONITOR_REFRESH_INTERVALS = {
  // Live status polling - most frequent
  STATUS: 5000, // 5 seconds
  
  // Hardware stats - frequent updates
  HARDWARE_STATS: 30000, // 30 seconds
  
  // Agent list refresh
  AGENTS_LIST: 30000, // 30 seconds
  
  // Native vnstat data
  NATIVE_DATA: 60000, // 1 minute
  
  // System info - less frequent
  SYSTEM_INFO: 300000, // 5 minutes
  
  // Peak stats
  PEAK_STATS: 30000, // 30 seconds
} as const;

// Type for the intervals
export type MonitorRefreshInterval = typeof MONITOR_REFRESH_INTERVALS[keyof typeof MONITOR_REFRESH_INTERVALS];