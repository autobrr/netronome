/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { Server } from "../types/types.ts";

/** A retained catalogue update plus any source-level discovery warnings. */
export interface SpeedtestServerCatalogueResult {
  servers: Server[];
  warnings: string[];
}

/** Normalizes full and partial server-list responses at the API boundary. */
export const parseServerCatalogueResponse = (
  data: unknown,
): SpeedtestServerCatalogueResult => {
  if (Array.isArray(data)) {
    return { servers: data as Server[], warnings: [] };
  }
  if (
    typeof data === "object" &&
    data !== null &&
    "servers" in data &&
    Array.isArray(data.servers) &&
    "warnings" in data &&
    Array.isArray(data.warnings) &&
    data.warnings.every((warning) => typeof warning === "string")
  ) {
    return { servers: data.servers as Server[], warnings: data.warnings };
  }
  throw new Error("Invalid server catalogue response");
};
