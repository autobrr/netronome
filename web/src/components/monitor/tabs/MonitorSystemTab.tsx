/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { MonitorSystemInfo } from "../MonitorSystemInfo";
import { MonitorHardwareStats } from "../MonitorHardwareStats";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";
import { ServerIcon } from "@heroicons/react/24/outline";

interface MonitorSystemTabProps {
  agent: MonitorAgent;
}

export const MonitorSystemTab: React.FC<MonitorSystemTabProps> = ({ agent }) => {
  const { status, systemInfo, hardwareStats } = useMonitorAgent({
    agent,
    includeSystemInfo: true,
    includeHardwareStats: true,
  });

  const isOffline = !status?.connected;
  const isLoading = !systemInfo && !hardwareStats;

  if (isLoading) {
    return (
      <div className="text-center py-12">
        <p className="text-lg text-gray-500 dark:text-gray-400">Loading system information...</p>
      </div>
    );
  }

  const hasData = systemInfo || hardwareStats;
  if (!hasData) {
    return (
      <div className="text-center py-12">
        <ServerIcon className="h-12 w-12 text-gray-400 mx-auto mb-4" />
        <p className="text-lg text-gray-500 dark:text-gray-400">No System Information Available</p>
        <p className="text-sm text-gray-400 dark:text-gray-500 mt-2">
          The agent may have just been added or hasn't collected any data yet.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Offline Banner */}
      {isOffline && <MonitorOfflineBanner message="Showing cached system information. Real-time data unavailable." />}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* System Information - Left Column */}
        <div>
          {systemInfo && (
            <MonitorSystemInfo systemInfo={systemInfo} isOffline={isOffline} />
          )}
        </div>
        
        {/* Hardware Statistics - Right Column */}
        <div>
          {hardwareStats && (
            <MonitorHardwareStats hardwareStats={hardwareStats} />
          )}
        </div>
      </div>
    </div>
  );
};