/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { getVnstatAgentUsage } from "@/api/vnstat";

interface VnstatUsageTableProps {
  agentId: number;
}

interface UsagePeriod {
  period: string;
  download: number; // Total bytes downloaded
  upload: number; // Total bytes uploaded
  total: number; // Total bytes
}

export const VnstatUsageTable: React.FC<VnstatUsageTableProps> = ({
  agentId,
}) => {
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";

    const units = ["B", "KB", "MB", "GB", "TB", "PB"];
    const k = 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${units[i]}`;
  };

  // Fetch usage data from API
  const {
    data: usageDataFromAPI,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["vnstat-agent-usage", agentId],
    queryFn: () => getVnstatAgentUsage(agentId),
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  // Convert API data to table format
  const usageData: UsagePeriod[] = usageDataFromAPI
    ? Object.entries(usageDataFromAPI)
        .map(([period, data]) => ({
          period,
          download: data.download,
          upload: data.upload,
          total: data.total,
        }))
        .sort((a, b) => {
          // Define the desired order
          const order = [
            "This Hour",
            "Last Hour",
            "Today",
            "This Month",
            "All Time",
          ];
          const indexA = order.indexOf(a.period);
          const indexB = order.indexOf(b.period);
          return indexA - indexB;
        })
    : [];

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
          <tr className="border-b border-gray-800">
            <th className="text-left py-3 px-2 text-gray-400 font-medium">
              Period
            </th>
            <th className="text-right py-3 px-2 text-gray-400 font-medium">
              Download
            </th>
            <th className="text-right py-3 px-2 text-gray-400 font-medium">
              Upload
            </th>
            <th className="text-right py-3 px-2 text-gray-400 font-medium">
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
              <td className="py-3 px-2 text-gray-700 dark:text-gray-300 text-left font-medium">
                {row.period}
              </td>
              <td className="py-3 px-2 text-gray-700 dark:text-gray-300 text-right font-mono">
                <span className="text-blue-600 dark:text-blue-400">
                  {formatBytes(row.download)}
                </span>
              </td>
              <td className="py-3 px-2 text-gray-700 dark:text-gray-300 text-right font-mono">
                <span className="text-emerald-600 dark:text-emerald-400">
                  {formatBytes(row.upload)}
                </span>
              </td>
              <td className="py-3 px-2 text-gray-700 dark:text-gray-300 text-right font-mono font-medium">
                {formatBytes(row.total)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
