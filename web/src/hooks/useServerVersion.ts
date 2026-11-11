/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/context/auth";
import { getServerVersion, serverVersionQueryKey } from "@/api/version";

const SERVER_VERSION_POLL_INTERVAL = 5 * 60 * 1000;

export function useServerVersion(): string | null {
  const { isAuthenticated } = useAuth();
  const { data, isError } = useQuery({
    queryKey: serverVersionQueryKey,
    queryFn: getServerVersion,
    enabled: isAuthenticated,
    staleTime: SERVER_VERSION_POLL_INTERVAL,
    refetchInterval: SERVER_VERSION_POLL_INTERVAL,
  });

  if (!isAuthenticated || isError) return null;
  return data?.version ?? null;
}
