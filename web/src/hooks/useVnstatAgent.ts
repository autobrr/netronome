/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  VnstatAgent,
  VnstatStatus,
  SystemInfo,
  PeakStats,
  HardwareStats,
  getVnstatAgentStatus,
  getVnstatAgentNative,
  startVnstatAgent,
  stopVnstatAgent,
} from "@/api/vnstat";

interface UseVnstatAgentOptions {
  agent: VnstatAgent;
  includeNativeData?: boolean;
  includeSystemInfo?: boolean;
  includePeakStats?: boolean;
  includeHardwareStats?: boolean;
}

export const useVnstatAgent = ({
  agent,
  includeNativeData = false,
  includeSystemInfo = false,
  includePeakStats = false,
  includeHardwareStats = false,
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

  // Debug logging for query conditions
  console.log("System info query conditions:", {
    agentEnabled: agent.enabled,
    includeSystemInfo,
    statusConnected: statusQuery.data?.connected,
    shouldBeEnabled: agent.enabled && includeSystemInfo && statusQuery.data?.connected
  });

  // Fetch system info (optional)
  const systemInfoQuery = useQuery<SystemInfo>({
    queryKey: ["vnstat-agent-system", agent.id],
    queryFn: async () => {
      console.log("Fetching system info...");
      // Direct fetch from agent URL with auth if needed
      const url = new URL(agent.url);
      const systemUrl = `${url.protocol}//${url.host}/system/info`;
      
      const headers: HeadersInit = {};
      if (agent.apiKey) {
        headers["X-API-Key"] = agent.apiKey;
      }

      const response = await fetch(systemUrl, { 
        method: 'GET',
        headers,
        mode: 'cors',
        credentials: 'omit', // Don't send cookies
      });
      if (!response.ok) {
        throw new Error("Failed to fetch system info");
      }
      const data = await response.json();
      console.log("System info data:", data);
      return data;
    },
    refetchInterval: agent.enabled ? 300000 : false, // Poll every 5 minutes
    enabled: agent.enabled && includeSystemInfo && statusQuery.data?.connected,
  });

  // Fetch peak stats (optional)
  const peakStatsQuery = useQuery<PeakStats>({
    queryKey: ["vnstat-agent-peaks", agent.id],
    queryFn: async () => {
      // Direct fetch from agent URL with auth if needed
      const url = new URL(agent.url);
      const peaksUrl = `${url.protocol}//${url.host}/stats/peaks`;
      
      const headers: HeadersInit = {};
      if (agent.apiKey) {
        headers["X-API-Key"] = agent.apiKey;
      }

      const response = await fetch(peaksUrl, { headers });
      if (!response.ok) {
        throw new Error("Failed to fetch peak stats");
      }
      return response.json();
    },
    refetchInterval: agent.enabled ? 30000 : false, // Poll every 30 seconds
    enabled: agent.enabled && includePeakStats && statusQuery.data?.connected,
  });

  // Fetch hardware stats (optional)
  const hardwareStatsQuery = useQuery<HardwareStats>({
    queryKey: ["vnstat-agent-hardware", agent.id],
    queryFn: async () => {
      // Direct fetch from agent URL with auth if needed
      const url = new URL(agent.url);
      const hardwareUrl = `${url.protocol}//${url.host}/system/hardware`;
      
      const headers: HeadersInit = {};
      if (agent.apiKey) {
        headers["X-API-Key"] = agent.apiKey;
      }

      const response = await fetch(hardwareUrl, { 
        method: 'GET',
        headers,
        mode: 'cors',
        credentials: 'omit', // Don't send cookies
      });
      if (!response.ok) {
        throw new Error("Failed to fetch hardware stats");
      }
      const data = await response.json();
      console.log("Hardware stats data:", data);
      return data;
    },
    refetchInterval: agent.enabled ? 30000 : false, // Poll every 30 seconds
    enabled: agent.enabled && includeHardwareStats && statusQuery.data?.connected,
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
    systemInfo: systemInfoQuery.data,
    peakStats: peakStatsQuery.data,
    hardwareStats: hardwareStatsQuery.data,
    isLoadingStatus: statusQuery.isLoading,
    isLoadingNativeData: nativeDataQuery.isLoading,
    isLoadingSystemInfo: systemInfoQuery.isLoading,
    isLoadingPeakStats: peakStatsQuery.isLoading,
    isLoadingHardwareStats: hardwareStatsQuery.isLoading,
    startMutation,
    stopMutation,
  };
};
