/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import {
  WifiIcon,
  ClockIcon,
  ArrowTrendingUpIcon,
  ArrowTrendingDownIcon,
} from "@heroicons/react/24/outline";
import { MetricCard } from "@/components/common/MetricCard";
import { formatRTT } from "../utils/packetLossUtils";
import { MonitorStatus } from "../types/monitorStatus";

interface MonitorStatusCardProps {
  status: MonitorStatus;
  threshold: number;
}

export const MonitorStatusCard: React.FC<MonitorStatusCardProps> = ({
  status,
  threshold,
}) => {
  // Only show testing progress when actively running a test
  if (
    status.isRunning &&
    status.progress !== undefined &&
    status.progress > 0
  ) {
    return (
      <div className="mb-6 p-4 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg border border-gray-300 dark:border-gray-800">
        <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-3">
          Current Status
        </h3>
        <div className="text-center py-4">
          <div className="flex justify-center items-center mb-3">
            <span className="relative inline-flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-blue-500"></span>
            </span>
          </div>
          <p className="text-blue-600 dark:text-blue-400 text-lg font-medium">
            {status.usedMtr
              ? "Running MTR analysis..."
              : "Testing in progress..."}
          </p>
          {status.usedMtr ? (
            <p className="text-gray-600 dark:text-gray-400 mt-2 text-sm">
              Analyzing network path with hop-by-hop statistics
            </p>
          ) : (
            <>
              <p className="text-gray-600 dark:text-gray-400 mt-2">
                {Math.round(status.progress || 0)}% complete
              </p>
              {status.packetsSent && (
                <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                  Packets: {status.packetsRecv || 0} / {status.packetsSent}{" "}
                  received
                </p>
              )}
            </>
          )}
        </div>
      </div>
    );
  }

  // Show last test results
  if (
    !status.isRunning &&
    !status.isComplete &&
    status.packetLoss !== undefined
  ) {
    return (
      <div className="mb-6 p-4 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg border border-gray-300 dark:border-gray-800">
        <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-3">
          Current Status
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            icon={<WifiIcon className="w-5 h-5 text-blue-500" />}
            title="Packet Loss"
            value={status.packetLoss?.toFixed(1) || "0"}
            unit="%"
            status={(status.packetLoss || 0) > threshold ? "error" : "success"}
          />
          <MetricCard
            icon={<ClockIcon className="w-5 h-5 text-emerald-500" />}
            title="Avg RTT"
            value={formatRTT(status.avgRtt || 0).replace("ms", "")}
            unit="ms"
            status="normal"
          />
          <MetricCard
            icon={<ArrowTrendingDownIcon className="w-5 h-5 text-green-500" />}
            title="Min RTT"
            value={formatRTT(status.minRtt || 0).replace("ms", "")}
            unit="ms"
            status="normal"
          />
          <MetricCard
            icon={<ArrowTrendingUpIcon className="w-5 h-5 text-yellow-500" />}
            title="Max RTT"
            value={formatRTT(status.maxRtt || 0).replace("ms", "")}
            unit="ms"
            status="normal"
          />
        </div>
      </div>
    );
  }

  return null;
};
