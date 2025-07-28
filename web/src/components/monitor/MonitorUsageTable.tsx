/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { MonitorAgent, MonitorUsageSummary } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import { formatBytes } from "@/utils/formatBytes";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface MonitorUsageTableProps {
  agent: MonitorAgent;
}

interface UsagePeriod {
  period: string;
  download: number; // Total bytes downloaded
  upload: number; // Total bytes uploaded
  total: number; // Total bytes
}

export const MonitorUsageTable: React.FC<MonitorUsageTableProps> = ({
  agent,
}) => {
  // Use the shared hook for native monitor data
  const { nativeData, isLoadingNativeData } = useMonitorAgent({
    agent,
    includeNativeData: true,
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

  if (isLoadingNativeData) {
    return (
      <div className="flex h-32 items-center justify-center">
        <p className="text-gray-500 dark:text-gray-400">
          Loading usage data...
        </p>
      </div>
    );
  }

  if (!nativeData && !isLoadingNativeData) {
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
    <Table className="table-fixed" noScroll>
      <TableHeader>
        <TableRow className="border-gray-300 dark:border-gray-800">
          <TableHead className="text-gray-600 dark:text-gray-400 w-[140px]">
            Period
          </TableHead>
          <TableHead className="text-right text-gray-600 dark:text-gray-400 w-[120px]">
            Download
          </TableHead>
          <TableHead className="text-right text-gray-600 dark:text-gray-400 w-[120px]">
            Upload
          </TableHead>
          <TableHead className="text-right text-gray-600 dark:text-gray-400 w-[120px]">
            Total
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {usageData.map((row) => (
          <TableRow
            key={row.period}
            className="border-gray-300/50 dark:border-gray-800/50 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors"
          >
            <TableCell className="text-gray-700 dark:text-gray-300 font-medium w-[140px]">
              {row.period}
            </TableCell>
            <TableCell className="text-right text-gray-700 dark:text-gray-300 font-mono w-[120px]">
              <span className="text-blue-600 dark:text-blue-400">
                {formatBytes(row.download)}
              </span>
            </TableCell>
            <TableCell className="text-right text-gray-700 dark:text-gray-300 font-mono w-[120px]">
              <span className="text-emerald-600 dark:text-emerald-400">
                {formatBytes(row.upload)}
              </span>
            </TableCell>
            <TableCell className="text-right text-gray-700 dark:text-gray-300 font-mono font-medium w-[120px]">
              {formatBytes(row.total)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
