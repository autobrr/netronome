/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/** Keeps server catalogue requests off PWA runtime caches so explicit refreshes cannot return stale data. */
export const speedtestServersNetworkOnlyRoute = {
  urlPattern: /\/api\/servers(?:\?|$)/,
  handler: "NetworkOnly" as const,
};
