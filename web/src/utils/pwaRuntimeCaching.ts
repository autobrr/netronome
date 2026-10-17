/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/** Keeps server catalogue requests off PWA runtime caches so explicit refreshes cannot return stale data. */
export const speedtestServersNetworkOnlyRoute = {
  urlPattern: /\/api\/servers(?:\/catalogue\/status)?(?:\?|$)/,
  handler: "NetworkOnly" as const,
};

/** Keeps exports off PWA runtime caches, so a large file streams to disk and is not stored twice. */
export const exportNetworkOnlyRoute = {
  urlPattern: /\/api\/export\//,
  handler: "NetworkOnly" as const,
};

/** Keeps a link to the API, such as an export download, from getting the app shell. */
export const apiNavigateFallbackDenylist = [/\/api\//];
