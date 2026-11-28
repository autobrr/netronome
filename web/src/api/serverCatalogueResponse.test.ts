/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { strict as assert } from "node:assert";
import { test } from "node:test";
import { parseServerCatalogueResponse } from "./serverCatalogueResponse.ts";

test("server catalogue responses preserve partial discovery warnings", () => {
  const servers = [{ id: "1" }];

  assert.deepEqual(parseServerCatalogueResponse(servers), {
    servers,
    warnings: [],
  });
  assert.deepEqual(
    parseServerCatalogueResponse({
      servers,
      warnings: ["fetch servers near Brisbane: request timed out"],
    }),
    {
      servers,
      warnings: ["fetch servers near Brisbane: request timed out"],
    },
  );
  assert.throws(
    () => parseServerCatalogueResponse({ servers, warnings: [42] }),
    /Invalid server catalogue response/,
  );
});
