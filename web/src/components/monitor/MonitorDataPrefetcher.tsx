/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { MonitorAgent } from "@/api/monitor";
import { 
  getMonitorAgentNative, 
  getMonitorAgentSystemInfo, 
  getMonitorAgentHardwareStats, 
  getMonitorAgentPeakStats 
} from "@/api/monitor";

interface MonitorDataPrefetcherProps {
  agents: MonitorAgent[];
}

// This component pre-fetches data for all enabled agents
export const MonitorDataPrefetcher: React.FC<MonitorDataPrefetcherProps> = ({ agents }) => {
  const queryClient = useQueryClient();

  useEffect(() => {
    // Filter to only enabled agents
    const enabledAgents = agents.filter(agent => agent.enabled);
    
    if (enabledAgents.length === 0) return;

    // Pre-fetch all data types that tabs will need for all enabled agents
    const allPrefetchPromises: Promise<unknown>[] = [];

    enabledAgents.forEach(agent => {
      allPrefetchPromises.push(
        queryClient.prefetchQuery({
          queryKey: ["monitor-agent-native", agent.id],
          queryFn: () => getMonitorAgentNative(agent.id),
          staleTime: 60000, // 1 minute
        }),
        queryClient.prefetchQuery({
          queryKey: ["monitor-agent-system", agent.id],
          queryFn: () => getMonitorAgentSystemInfo(agent.id),
          staleTime: 300000, // 5 minutes
        }),
        queryClient.prefetchQuery({
          queryKey: ["monitor-agent-hardware", agent.id],
          queryFn: () => getMonitorAgentHardwareStats(agent.id),
          staleTime: 30000, // 30 seconds
        }),
        queryClient.prefetchQuery({
          queryKey: ["monitor-agent-peaks", agent.id],
          queryFn: () => getMonitorAgentPeakStats(agent.id),
          staleTime: 30000, // 30 seconds
        })
      );
    });

    // Execute all prefetch operations
    Promise.all(allPrefetchPromises).catch((error) => {
      console.warn("Failed to prefetch monitor data:", error);
    });
  }, [agents, queryClient]);

  // This component doesn't render anything
  return null;
};