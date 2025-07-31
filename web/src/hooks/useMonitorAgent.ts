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
import { MONITOR_REFRESH_INTERVALS } from "@/constants/monitorRefreshIntervals";
import { showToast } from "@/components/common/Toast";

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
    refetchInterval: agent.enabled ? MONITOR_REFRESH_INTERVALS.STATUS : false,
    staleTime: MONITOR_REFRESH_INTERVALS.STATUS / 2, // Consider data fresh for half the refetch interval
    enabled: agent.enabled,
  });

  // Fetch native monitor data (optional)
  const nativeDataQuery = useQuery({
    queryKey: ["monitor-agent-native", agent.id],
    queryFn: () => getMonitorAgentNative(agent.id),
    refetchInterval: agent.enabled
      ? MONITOR_REFRESH_INTERVALS.NATIVE_DATA
      : false,
    staleTime: MONITOR_REFRESH_INTERVALS.NATIVE_DATA, // Keep data fresh for the full interval
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes even when unused
    enabled: agent.enabled && includeNativeData,
  });

  // Fetch system info (optional) - always fetch if requested, regardless of connection status
  const systemInfoQuery = useQuery<SystemInfo>({
    queryKey: ["monitor-agent-system", agent.id],
    queryFn: () => getMonitorAgentSystemInfo(agent.id),
    refetchInterval: agent.enabled
      ? MONITOR_REFRESH_INTERVALS.SYSTEM_INFO
      : false,
    staleTime: MONITOR_REFRESH_INTERVALS.SYSTEM_INFO / 2, // Consider data fresh for half the refetch interval
    enabled: agent.enabled && includeSystemInfo,
  });

  // Fetch peak stats (optional)
  const peakStatsQuery = useQuery<PeakStats>({
    queryKey: ["monitor-agent-peaks", agent.id],
    queryFn: () => getMonitorAgentPeakStats(agent.id),
    refetchInterval: agent.enabled
      ? MONITOR_REFRESH_INTERVALS.PEAK_STATS
      : false,
    staleTime: MONITOR_REFRESH_INTERVALS.PEAK_STATS / 2, // Consider data fresh for half the refetch interval
    enabled: agent.enabled && includePeakStats && statusQuery.data?.connected,
  });

  // Fetch hardware stats (optional) - always fetch if requested, regardless of connection status
  const hardwareStatsQuery = useQuery<HardwareStats>({
    queryKey: ["monitor-agent-hardware", agent.id],
    queryFn: () => getMonitorAgentHardwareStats(agent.id),
    refetchInterval: agent.enabled
      ? MONITOR_REFRESH_INTERVALS.HARDWARE_STATS
      : false,
    staleTime: MONITOR_REFRESH_INTERVALS.HARDWARE_STATS, // Keep data fresh for the full interval
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes even when unused
    enabled: agent.enabled && includeHardwareStats,
  });

  // Start mutation
  const startMutation = useMutation({
    mutationFn: () => startMonitorAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["monitor-agent-status", agent.id],
      });
      showToast("Agent started", "success", {
        description: `${agent.name} is now active`,
      });
    },
    onError: (error: Error) => {
      showToast("Failed to start agent", "error", {
        description: error.message || "Unable to start the monitoring agent",
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
      showToast("Agent stopped", "success", {
        description: `${agent.name} has been stopped`,
      });
    },
    onError: (error: Error) => {
      showToast("Failed to stop agent", "error", {
        description: error.message || "Unable to stop the monitoring agent",
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
