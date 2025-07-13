/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { VnstatUsageTable } from "./VnstatUsageTable";
import { VnstatLiveMonitor } from "./VnstatLiveMonitor";
import { VnstatAgent, VnstatStatus, getVnstatAgentStatus } from "@/api/vnstat";

interface VnstatAgentDetailsProps {
  agent: VnstatAgent;
}

export const VnstatAgentDetails: React.FC<VnstatAgentDetailsProps> = ({
  agent,
}) => {
  // Fetch agent status
  const { data: status } = useQuery<VnstatStatus>({
    queryKey: ["vnstat-agent-status", agent.id],
    queryFn: () => getVnstatAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 2000 : false, // Poll every 2 seconds if enabled
  });

  return (
    <div className="space-y-6">
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
