/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/context/auth";
import { getServerVersion, serverVersionQueryKey } from "@/api/version";

export function useServerVersion(): string | null {
  const { isAuthenticated } = useAuth();
  const { data } = useQuery({
    queryKey: serverVersionQueryKey,
    queryFn: getServerVersion,
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });

  return isAuthenticated ? (data?.version ?? null) : null;
}
