/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Animation configuration for Motion components
 * Moved outside component to prevent re-creation
 */
export const SPRING_TRANSITION = {
  type: "spring" as const,
  stiffness: 500,
  damping: 30,
} as const;

/**
 * Default display count for servers list
 */
export const DEFAULT_SERVER_DISPLAY_COUNT = 4;

/**
 * Server list increment when loading more
 */
export const SERVER_DISPLAY_INCREMENT = 4;

/**
 * Default traceroute configuration
 */
export const DEFAULT_TRACEROUTE_CONFIG = {
  totalHops: 30,
  initialProgress: 0,
  pollInterval: 1000, // 1 second
} as const;

/**
 * Tab mode types for the traceroute interface
 */
export type TabMode = "traceroute" | "monitors";

/**
 * Default tab mode
 */
export const DEFAULT_TAB_MODE: TabMode = "traceroute";

/**
 * LocalStorage key for tab mode persistence
 */
export const TAB_MODE_STORAGE_KEY = "netronome-packetloss-tab-mode";

/**
 * Default values for traceroute status polling
 */
export const TRACEROUTE_POLLING = {
  interval: 1000, // 1 second
  enabled: true,
} as const;

/**
 * Copy success notification duration (milliseconds)
 */
export const COPY_SUCCESS_DURATION = 2000;

/**
 * Table column configurations
 */
export const TABLE_COLUMNS = {
  hop: "Hop",
  host: "Host",
  provider: "Provider",
  rtt1: "RTT 1",
  rtt2: "RTT 2",
  rtt3: "RTT 3",
  average: "Average",
} as const;

