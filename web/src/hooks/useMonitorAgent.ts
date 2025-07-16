/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  MonitorAgent,
  MonitorStatus,
  SystemInfo,
  PeakStats,
  HardwareStats,
  getMonitorAgentStatus,
  getMonitorAgentNative,
  getMonitorAgentSystemInfo,
  getMonitorAgentHardwareStats,
  getMonitorAgentPeakStats,
  startMonitorAgent,
  stopMonitorAgent,
} from "@/api/monitor";

interface UseMonitorAgentOptions {
  agent: MonitorAgent;
  includeNativeData?: boolean;
  includeSystemInfo?: boolean;
  includePeakStats?: boolean;
  includeHardwareStats?: boolean;
}

export const useMonitorAgent = ({
  agent,
  includeNativeData = false,
  includeSystemInfo = false,
  includePeakStats = false,
  includeHardwareStats = false,
}: UseMonitorAgentOptions) => {
  const queryClient = useQueryClient();

  // Fetch agent status with polling
  const statusQuery = useQuery<MonitorStatus>({
    queryKey: ["monitor-agent-status", agent.id],
    queryFn: () => getMonitorAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 2000 : false, // Poll every 2 seconds if enabled
    enabled: agent.enabled,
  });

  // Fetch native monitor data (optional)
  const nativeDataQuery = useQuery({
    queryKey: ["monitor-agent-native", agent.id],
    queryFn: () => getMonitorAgentNative(agent.id),
    refetchInterval: agent.enabled ? 60000 : false, // Poll every minute
    enabled: agent.enabled && includeNativeData,
  });

  // Fetch system info (optional)
  const systemInfoQuery = useQuery<SystemInfo>({
    queryKey: ["monitor-agent-system", agent.id],
    queryFn: () => getMonitorAgentSystemInfo(agent.id),
    refetchInterval: agent.enabled ? 300000 : false, // Poll every 5 minutes
    enabled: agent.enabled && includeSystemInfo && statusQuery.data?.connected,
  });

  // Fetch peak stats (optional)
  const peakStatsQuery = useQuery<PeakStats>({
    queryKey: ["monitor-agent-peaks", agent.id],
    queryFn: () => getMonitorAgentPeakStats(agent.id),
    refetchInterval: agent.enabled ? 30000 : false, // Poll every 30 seconds
    enabled: agent.enabled && includePeakStats && statusQuery.data?.connected,
  });

  // Fetch hardware stats (optional)
  const hardwareStatsQuery = useQuery<HardwareStats>({
    queryKey: ["monitor-agent-hardware", agent.id],
    queryFn: () => getMonitorAgentHardwareStats(agent.id),
    refetchInterval: agent.enabled ? 30000 : false, // Poll every 30 seconds
    enabled: agent.enabled && includeHardwareStats && statusQuery.data?.connected,
  });

  // Start mutation
  const startMutation = useMutation({
    mutationFn: () => startMonitorAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["monitor-agent-status", agent.id],
      });
    },
  });

  // Stop mutation
  const stopMutation = useMutation({
    mutationFn: () => stopMonitorAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["monitor-agent-status", agent.id],
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
