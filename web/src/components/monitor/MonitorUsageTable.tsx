/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { getMonitorAgentNative } from "@/api/monitor";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import type { MonitorUsageSummary } from "@/api/monitor";
import { formatBytes } from "@/utils/formatBytes";

interface MonitorUsageTableProps {
  agentId: number;
}

interface UsagePeriod {
  period: string;
  download: number; // Total bytes downloaded
  upload: number; // Total bytes uploaded
  total: number; // Total bytes
}

export const MonitorUsageTable: React.FC<MonitorUsageTableProps> = ({
  agentId,
}) => {
  // Fetch native monitor data from agent
  const {
    data: nativeData,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["monitor-agent-native", agentId],
    queryFn: () => getMonitorAgentNative(agentId),
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  // Parse monitor native data into usage periods
  const usageDataFromParser = nativeData
    ? parseMonitorUsagePeriods(nativeData)
    : {};

  // Convert parsed data to table format
  const usageData: UsagePeriod[] = Object.entries(usageDataFromParser).map(
    ([period, data]) => ({
      period,
      download: (data as MonitorUsageSummary).download,
      upload: (data as MonitorUsageSummary).upload,
      total: (data as MonitorUsageSummary).total,
    })
  );

  if (isLoading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <p className="text-gray-500 dark:text-gray-400">
          Loading usage data...
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-32 items-center justify-center">
        <p className="text-red-500 dark:text-red-400">
          Failed to load usage data
        </p>
      </div>
    );
  }

  if (usageData.length === 0) {
    return (
      <div className="flex h-32 items-center justify-center">
        <p className="text-gray-500 dark:text-gray-400">
          No usage data available
        </p>
      </div>
    );
  }

  return (
    <div>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-300 dark:border-gray-800">
            <th className="text-left py-3 px-4 text-gray-400 font-medium">
              Period
            </th>
            <th className="text-right py-3 px-4 text-gray-400 font-medium">
              Download
            </th>
            <th className="text-right py-3 px-4 text-gray-400 font-medium">
              Upload
            </th>
            <th className="text-right py-3 px-4 text-gray-400 font-medium">
              Total
            </th>
          </tr>
        </thead>
        <tbody>
          {usageData.map((row) => (
            <tr
              key={row.period}
              className="border-b border-gray-300/50 dark:border-gray-800/50 last:border-0 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors"
            >
              <td className="py-3 px-4 text-gray-700 dark:text-gray-300 text-left font-medium">
                {row.period}
              </td>
              <td className="py-3 px-4 text-gray-700 dark:text-gray-300 text-right font-mono">
                <span className="text-blue-600 dark:text-blue-400">
                  {formatBytes(row.download)}
                </span>
              </td>
              <td className="py-3 px-4 text-gray-700 dark:text-gray-300 text-right font-mono">
                <span className="text-emerald-600 dark:text-emerald-400">
                  {formatBytes(row.upload)}
                </span>
              </td>
              <td className="py-3 px-4 text-gray-700 dark:text-gray-300 text-right font-mono font-medium">
                {formatBytes(row.total)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
