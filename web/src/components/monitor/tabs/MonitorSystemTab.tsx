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

      <div className="space-y-6 lg:space-y-0 lg:grid lg:grid-cols-3 lg:gap-6">
        {/* Left Column - System Information */}
        <div className="order-1">
          {systemInfo && (
            <MonitorSystemInfo systemInfo={systemInfo} isOffline={isOffline} />
          )}
        </div>
        
        {/* CPU - order 2 on mobile, middle column on desktop */}
        <div className="order-2 lg:order-2 space-y-6">
          {hardwareStats && (
            <>
              <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="cpu" />
              <div className="hidden lg:block">
                {hardwareStats.temperature && hardwareStats.temperature.length > 0 && (
                  <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="temperature" />
                )}
              </div>
            </>
          )}
        </div>
        
        {/* Memory - order 3 on mobile, right column on desktop */}
        <div className="order-3 lg:order-3 space-y-6">
          {hardwareStats && (
            <>
              <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="memory" />
              <div className="hidden lg:block">
                {hardwareStats.disks.length > 0 && (
                  <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="disk" />
                )}
              </div>
            </>
          )}
        </div>
        
        {/* Temperature Sensors - order 4 on mobile (shows only on mobile) */}
        <div className="order-4 lg:hidden">
          {hardwareStats && hardwareStats.temperature && hardwareStats.temperature.length > 0 && (
            <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="temperature" />
          )}
        </div>
        
        {/* Disk Usage - order 5 on mobile (shows only on mobile) */}
        <div className="order-5 lg:hidden">
          {hardwareStats && hardwareStats.disks.length > 0 && (
            <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="disk" />
          )}
        </div>
      </div>

      {/* Single timestamp at bottom */}
      {(systemInfo || hardwareStats) && (
        <div className="text-center text-xs text-gray-500 dark:text-gray-400 mt-6">
          {hardwareStats?.from_cache || systemInfo?.from_cache ? "Data collected" : "Last updated"}:{" "}
          {new Date((hardwareStats?.updated_at || systemInfo?.updated_at) || Date.now()).toLocaleTimeString()}
          {(hardwareStats?.from_cache || systemInfo?.from_cache) && " (cached)"}
        </div>
      )}
    </div>
  );
};