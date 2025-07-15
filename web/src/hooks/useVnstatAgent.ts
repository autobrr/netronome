/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  VnstatAgent,
  VnstatStatus,
  getVnstatAgentStatus,
  getVnstatAgentNative,
  startVnstatAgent,
  stopVnstatAgent,
} from "@/api/vnstat";

interface UseVnstatAgentOptions {
  agent: VnstatAgent;
  includeNativeData?: boolean;
}

export const useVnstatAgent = ({
  agent,
  includeNativeData = false,
}: UseVnstatAgentOptions) => {
  const queryClient = useQueryClient();

  // Fetch agent status with polling
  const statusQuery = useQuery<VnstatStatus>({
    queryKey: ["vnstat-agent-status", agent.id],
    queryFn: () => getVnstatAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 2000 : false, // Poll every 2 seconds if enabled
    enabled: agent.enabled,
  });

  // Fetch native vnstat data (optional)
  const nativeDataQuery = useQuery({
    queryKey: ["vnstat-agent-native", agent.id],
    queryFn: () => getVnstatAgentNative(agent.id),
    refetchInterval: agent.enabled ? 60000 : false, // Poll every minute
    enabled: agent.enabled && includeNativeData,
  });

  // Start mutation
  const startMutation = useMutation({
    mutationFn: () => startVnstatAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["vnstat-agent-status", agent.id],
      });
    },
  });

  // Stop mutation
  const stopMutation = useMutation({
    mutationFn: () => stopVnstatAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["vnstat-agent-status", agent.id],
      });
    },
  });

  return {
    status: statusQuery.data,
    nativeData: nativeDataQuery.data,
    isLoadingStatus: statusQuery.isLoading,
    isLoadingNativeData: nativeDataQuery.isLoading,
    startMutation,
    stopMutation,
  };
};
