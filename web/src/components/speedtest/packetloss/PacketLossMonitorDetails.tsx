/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useEffect } from "react";
import { motion } from "motion/react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { GlobeAltIcon } from "@heroicons/react/24/outline";
import { PacketLossMonitor } from "@/types/types";
import { getPacketLossHistory } from "@/api/packetloss";
import { MonitorStatusCard } from "./components/MonitorStatusCard";
import { MonitorPerformanceChart } from "./components/MonitorPerformanceChart";
import { MonitorResultsTable } from "./components/MonitorResultsTable";
import { MonitorStatus } from "./types/monitorStatus";

interface PacketLossMonitorDetailsProps {
  selectedMonitor: PacketLossMonitor;
  monitorStatuses: Map<number, MonitorStatus>;
  onTraceRoute?: () => void;
}

export const PacketLossMonitorDetails: React.FC<
  PacketLossMonitorDetailsProps
> = ({ selectedMonitor, monitorStatuses, onTraceRoute }) => {
  const queryClient = useQueryClient();

  // Fetch history for selected monitor
  const { data: monitorHistory } = useQuery({
    queryKey: ["packetloss", "history", selectedMonitor.id],
    queryFn: () => getPacketLossHistory(selectedMonitor.id, 100),
    staleTime: 5000, // Consider data stale after 5 seconds
    refetchInterval: false, // Don't refetch automatically
  });

  // Ensure monitorHistory is always an array
  const historyList = monitorHistory || [];

  // Fetch fresh history when monitor is selected
  useEffect(() => {
    // Fetch fresh data and update cache directly
    getPacketLossHistory(selectedMonitor.id, 100)
      .then((freshHistory) => {
        queryClient.setQueryData(
          ["packetloss", "history", selectedMonitor.id],
          freshHistory,
        );
      })
      .catch((error) => {
        console.error("Failed to fetch monitor history:", error);
      });
  }, [selectedMonitor, queryClient]);

  const status = monitorStatuses.get(selectedMonitor.id);
  const showStatus =
    selectedMonitor.enabled &&
    status &&
    (status.isRunning || status.packetLoss !== undefined);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.1 }}
      className="flex-1"
    >
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            {selectedMonitor.name || selectedMonitor.host} Details
          </h2>
          {onTraceRoute && (
            <motion.button
              onClick={onTraceRoute}
              className="flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-xs bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30 hover:bg-amber-500/20"
              title="Run traceroute to this host"
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
            >
              <GlobeAltIcon className="w-3 h-3" />
              <span>Trace Route</span>
            </motion.button>
          )}
        </div>

        {/* Current Status - Only show when actively testing or has recent results */}
        {showStatus && (
          <MonitorStatusCard
            status={status}
            threshold={selectedMonitor.threshold}
          />
        )}

        {/* Performance Chart */}
        <MonitorPerformanceChart
          historyList={historyList}
          selectedMonitorId={selectedMonitor.id}
        />

        {/* Recent Results */}
        <MonitorResultsTable
          historyList={historyList}
          selectedMonitor={selectedMonitor}
        />
      </div>
    </motion.div>
  );
};
