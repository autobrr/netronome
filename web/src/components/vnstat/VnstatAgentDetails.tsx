/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ServerStackIcon } from "@heroicons/react/24/solid";
import { Button } from "@/components/ui/Button";
import { VnstatUsageTable } from "./VnstatUsageTable";
import { VnstatLiveMonitor } from "./VnstatLiveMonitor";
import {
  VnstatAgent,
  VnstatStatus,
  getVnstatAgentStatus,
  startVnstatAgent,
  stopVnstatAgent,
} from "@/api/vnstat";

interface VnstatAgentDetailsProps {
  agent: VnstatAgent;
}

export const VnstatAgentDetails: React.FC<VnstatAgentDetailsProps> = ({
  agent,
}) => {
  const queryClient = useQueryClient();

  // Fetch agent status
  const { data: status } = useQuery<VnstatStatus>({
    queryKey: ["vnstat-agent-status", agent.id],
    queryFn: () => getVnstatAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 2000 : false, // Poll every 2 seconds if enabled
  });

  // Start/stop mutations
  const startMutation = useMutation({
    mutationFn: () => startVnstatAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["vnstat-agent-status", agent.id],
      });
    },
  });

  const stopMutation = useMutation({
    mutationFn: () => stopVnstatAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["vnstat-agent-status", agent.id],
      });
    },
  });

  const handleToggleMonitoring = () => {
    if (agent.enabled && status?.connected) {
      stopMutation.mutate();
    } else {
      startMutation.mutate();
    }
  };

  return (
    <div className="space-y-6">
      {/* Agent Info Card */}
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <div className="flex items-start justify-between">
          <div className="flex items-start space-x-4 flex-1">
            <ServerStackIcon className="h-10 w-10 text-gray-400 mt-1" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-3 mb-2">
                <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                  {agent.name}
                </h3>
                <div
                  className={`h-2 w-2 rounded-full ${
                    !agent.enabled
                      ? "bg-gray-400"
                      : status?.connected
                        ? "bg-green-500"
                        : "bg-red-500"
                  }`}
                />
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {!agent.enabled
                    ? "Disabled"
                    : status?.connected
                      ? "Connected"
                      : "Disconnected"}
                </span>
              </div>

              <div className="space-y-1">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  <span className="font-medium">URL:</span>{" "}
                  {agent.url.replace(/\/events\?stream=live-data$/, "")}
                </p>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  <span className="font-medium">Retention:</span>{" "}
                  {agent.retentionDays || 365} days
                </p>
                {agent.enabled && status?.liveData && (
                  <div className="flex items-center space-x-4 mt-2">
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      <span className="font-medium">↓</span>{" "}
                      {status.liveData.rx.ratestring}
                    </span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      <span className="font-medium">↑</span>{" "}
                      {status.liveData.tx.ratestring}
                    </span>
                  </div>
                )}
              </div>
            </div>
          </div>

          <Button
            onClick={handleToggleMonitoring}
            disabled={startMutation.isPending || stopMutation.isPending}
            className={
              agent.enabled && status?.connected
                ? "bg-red-500/10 hover:bg-red-500/20 text-red-600 dark:text-red-400 border-red-500/20"
                : "bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
            }
          >
            {agent.enabled && status?.connected ? "Stop" : "Start"}
          </Button>
        </div>
      </div>

      {/* Live Monitor */}
      {agent.enabled && status?.connected && status.liveData && (
        <VnstatLiveMonitor liveData={status.liveData} />
      )}

      {/* Data Usage Table */}
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <div className="mb-4">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">
            Data Usage Summary
          </h3>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Total bandwidth usage by time period
          </p>
        </div>

        <VnstatUsageTable agentId={agent.id} />
      </div>
    </div>
  );
};
