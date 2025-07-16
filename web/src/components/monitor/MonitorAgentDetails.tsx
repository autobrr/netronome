/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { MonitorUsageTable } from "./MonitorUsageTable";
import { MonitorLiveMonitor } from "./MonitorLiveMonitor";
import { MonitorSystemInfo } from "./MonitorSystemInfo";
import { MonitorBandwidthChart } from "./MonitorBandwidthChart";
import { MonitorHardwareStats } from "./MonitorHardwareStats";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";

interface MonitorAgentDetailsProps {
  agent: MonitorAgent;
}

export const MonitorAgentDetails: React.FC<MonitorAgentDetailsProps> = ({
  agent,
}) => {
  // Use the shared hook for agent status and system info
  const { status, systemInfo, nativeData, hardwareStats } = useMonitorAgent({ 
    agent,
    includeSystemInfo: true,
    includeNativeData: true,
    includeHardwareStats: true,
  });

  // Transform native data for hourly chart
  const hourlyChartData = React.useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.hour) return [];
    
    const hourData = nativeData.interfaces[0].traffic.hour;
    // monitor returns newest first, reverse for chronological order
    return hourData.slice().reverse().map((item) => {
      const date = new Date(
        item.date.year,
        item.date.month - 1,
        item.date.day || 1,
        item.time?.hour || 0,
        item.time?.minute || 0
      );
      
      return {
        time: date.toISOString(),
        rx: item.rx,
        tx: item.tx,
      };
    });
  }, [nativeData]);

  // Transform native data for daily chart
  const dailyChartData = React.useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.day) return [];
    
    const dayData = nativeData.interfaces[0].traffic.day;
    // monitor returns newest first, reverse for chronological order
    return dayData.slice().reverse().map((item) => {
      const date = new Date(
        item.date.year,
        item.date.month - 1,
        item.date.day || 1
      );
      
      return {
        time: date.toISOString(),
        rx: item.rx,
        tx: item.tx,
      };
    });
  }, [nativeData]);

  return (
    <div className="space-y-6">
      {/* Live Monitor */}
      {agent.enabled && status?.connected && status.liveData && (
        <MonitorLiveMonitor 
          liveData={status.liveData}
          thresholds={{
            download: 100 * 1024 * 1024, // 100 MiB/s
            upload: 50 * 1024 * 1024,    // 50 MiB/s
          }}
        />
      )}

      {/* System Information */}
      {agent.enabled && status?.connected && systemInfo && (
        <MonitorSystemInfo systemInfo={systemInfo} />
      )}

      {/* Hardware Stats */}
      {agent.enabled && status?.connected && hardwareStats && (
        <MonitorHardwareStats hardwareStats={hardwareStats} />
      )}

      {/* Hourly Bandwidth Chart */}
      {agent.enabled && hourlyChartData.length > 0 && (
        <MonitorBandwidthChart
          data={hourlyChartData}
          title="Hourly Bandwidth"
          timeFormat="hour"
        />
      )}

      {/* Daily Bandwidth Chart */}
      {agent.enabled && dailyChartData.length > 0 && (
        <MonitorBandwidthChart
          data={dailyChartData}
          title="Daily Bandwidth"
          timeFormat="day"
        />
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

        <MonitorUsageTable agentId={agent.id} />
      </div>
    </div>
  );
};
