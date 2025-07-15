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

/**
 * CSS class constants for consistent styling
 */
// TODO: GET RID OF THIS SHIT
export const STYLES = {
  card: "bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800",
  input:
    "px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50",
  button: {
    primary:
      "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700",
    secondary:
      "bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300/80 dark:border-gray-800/80 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-800 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800",
    disabled:
      "bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500 border-gray-200 dark:border-gray-700 cursor-not-allowed",
    running:
      "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-300 dark:border-emerald-500/30 cursor-not-allowed",
  },
  table: {
    header: "border-b border-gray-800",
    headerCell: "text-center py-3 px-2 text-gray-400 font-medium",
    row: "border-b border-gray-300/50 dark:border-gray-800/50 last:border-0 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors",
    cell: "py-3 px-2 text-gray-700 dark:text-gray-300 text-center",
    providerCell: "py-3 px-2 min-w-[200px] max-w-[200px] text-center",
    monoCell: "py-3 px-2 text-center font-mono",
  },
  mobileCard: "bg-gray-200/50 dark:bg-gray-800/50 rounded-lg p-4",
  progressContainer:
    "mb-6 p-4 backdrop-blur-sm bg-blue-500/10 border border-blue-500/30 rounded-lg",
  errorContainer:
    "mb-6 p-4 backdrop-blur-sm bg-red-500/10 border border-red-500/30 rounded-lg",
} as const;
