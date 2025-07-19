/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ServerIcon,
  ChartBarIcon,
  CpuChipIcon,
  ClockIcon,
  SignalIcon,
  CalendarIcon,
} from "@heroicons/react/24/outline";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faLinux, faApple } from "@fortawesome/free-brands-svg-icons";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { formatBytes } from "@/utils/formatBytes";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";

interface MonitorOverviewTabProps {
  agent: MonitorAgent;
}

const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  
  if (days > 0) {
    return `${days}d ${hours}h`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
};

export const MonitorOverviewTab: React.FC<MonitorOverviewTabProps> = ({
  agent,
}) => {
  const { status, nativeData, hardwareStats, systemInfo, peakStats } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeSystemInfo: true,
    includeHardwareStats: true,
    includePeakStats: true,
  });

  const usage = nativeData ? parseMonitorUsagePeriods(nativeData) : null;

  const isOffline = !status?.connected;

  return (
    <div className="space-y-6">
      {/* Offline Banner */}
      {isOffline && <MonitorOfflineBanner />}

      {/* Key Metrics Grid - Now 2x2 on desktop */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Current Speed */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              Current Speed
            </h3>
            <ServerIcon className="h-5 w-5 text-gray-400" />
          </div>
          {status?.liveData ? (
            <>
              <div className="flex items-center space-x-2 mb-2">
                <ArrowDownIcon className="h-4 w-4 text-blue-500" />
                <span className="text-lg font-bold text-gray-900 dark:text-white">
                  {status.liveData.rx.ratestring}
                </span>
              </div>
              <div className="flex items-center space-x-2">
                <ArrowUpIcon className="h-4 w-4 text-green-500" />
                <span className="text-lg font-bold text-gray-900 dark:text-white">
                  {status.liveData.tx.ratestring}
                </span>
              </div>
            </>
          ) : isOffline ? (
            <p className="text-gray-500 dark:text-gray-400">Offline</p>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>

        {/* Today's Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              Today's Usage
            </h3>
            <ChartBarIcon className="h-5 w-5 text-gray-400" />
          </div>
          {usage?.["Today"] ? (
            <>
              <p className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                {formatBytes(usage["Today"].total)}
              </p>
              <div className="flex items-center space-x-4 text-sm">
                <div className="flex items-center space-x-1">
                  <ArrowDownIcon className="h-3 w-3 text-blue-500" />
                  <span className="text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["Today"].download)}
                  </span>
                </div>
                <div className="flex items-center space-x-1">
                  <ArrowUpIcon className="h-3 w-3 text-green-500" />
                  <span className="text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["Today"].upload)}
                  </span>
                </div>
              </div>
            </>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>


        {/* System Health */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.2 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              System Health
            </h3>
            {systemInfo?.kernel.toLowerCase().includes("darwin") ? (
              <FontAwesomeIcon
                icon={faApple}
                className="h-5 w-5 text-gray-400"
              />
            ) : systemInfo?.kernel.toLowerCase().includes("linux") ? (
              <FontAwesomeIcon
                icon={faLinux}
                className="h-5 w-5 text-gray-400"
              />
            ) : (
              <CpuChipIcon className="h-5 w-5 text-gray-400" />
            )}
          </div>
          {hardwareStats ? (
            <>
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    CPU
                  </span>
                  <span className="text-sm font-medium text-gray-900 dark:text-white">
                    {hardwareStats.cpu.usage_percent.toFixed(1)}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className="h-2 rounded-full transition-all duration-300"
                    style={{
                      width: `${hardwareStats.cpu.usage_percent}%`,
                      backgroundColor:
                        hardwareStats.cpu.usage_percent < 70
                          ? "#10B981"
                          : hardwareStats.cpu.usage_percent < 85
                          ? "#F59E0B"
                          : "#EF4444",
                    }}
                  />
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    Memory
                  </span>
                  <span className="text-sm font-medium text-gray-900 dark:text-white">
                    {hardwareStats.memory.used_percent.toFixed(1)}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className="h-2 rounded-full transition-all duration-300"
                    style={{
                      width: `${hardwareStats.memory.used_percent}%`,
                      backgroundColor:
                        hardwareStats.memory.used_percent < 70
                          ? "#10B981"
                          : hardwareStats.memory.used_percent < 85
                          ? "#F59E0B"
                          : "#EF4444",
                    }}
                  />
                </div>
                {/* Temperature Alert */}
                {hardwareStats.temperature && hardwareStats.temperature.length > 0 && (() => {
                  const hotSensors = hardwareStats.temperature.filter(t => t.temperature > 80);
                  const warmSensors = hardwareStats.temperature.filter(t => t.temperature > 60 && t.temperature <= 80);
                  
                  if (hotSensors.length > 0) {
                    return (
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Temperature
                        </span>
                        <span className="text-sm font-medium text-red-600 dark:text-red-400">
                          {hotSensors.length} Hot
                        </span>
                      </div>
                    );
                  } else if (warmSensors.length > 0) {
                    return (
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Temperature
                        </span>
                        <span className="text-sm font-medium text-amber-600 dark:text-amber-400">
                          {warmSensors.length} Warm
                        </span>
                      </div>
                    );
                  } else {
                    return (
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Temperature
                        </span>
                        <span className="text-sm font-medium text-green-600 dark:text-green-400">
                          Normal
                        </span>
                      </div>
                    );
                  }
                })()}
              </div>
            </>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>
        {/* This Week Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.3 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              This Week
            </h3>
            <CalendarIcon className="h-5 w-5 text-gray-400" />
          </div>
          {usage?.["This week"] ? (
            <>
              <p className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                {formatBytes(usage["This week"].total)}
              </p>
              <div className="flex items-center space-x-4 text-sm">
                <div className="flex items-center space-x-1">
                  <ArrowDownIcon className="h-3 w-3 text-blue-500" />
                  <span className="text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["This week"].download)}
                  </span>
                </div>
                <div className="flex items-center space-x-1">
                  <ArrowUpIcon className="h-3 w-3 text-green-500" />
                  <span className="text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["This week"].upload)}
                  </span>
                </div>
              </div>
            </>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>
      </div>

      {/* Quick Stats Row */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.4 }}
        className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">
            Quick Stats
          </h3>
          <SignalIcon className="h-5 w-5 text-gray-400" />
        </div>
        
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          {/* Uptime */}
          <div className="text-center">
            <ClockIcon className="h-8 w-8 mx-auto mb-2 text-gray-400" />
            <p className="text-sm text-gray-600 dark:text-gray-400">Uptime</p>
            <p className="text-lg font-bold text-gray-900 dark:text-white">
              {systemInfo?.uptime ? formatUptime(systemInfo.uptime) : "N/A"}
            </p>
          </div>

          {/* This Month Usage */}
          <div className="text-center">
            <ChartBarIcon className="h-8 w-8 mx-auto mb-2 text-purple-500" />
            <p className="text-sm text-gray-600 dark:text-gray-400">This Month</p>
            <p className="text-lg font-bold text-gray-900 dark:text-white">
              {usage?.["This Month"] ? formatBytes(usage["This Month"].total) : "N/A"}
            </p>
          </div>

          {/* Peak Download */}
          <div className="text-center">
            <ArrowDownIcon className="h-8 w-8 mx-auto mb-2 text-blue-500" />
            <p className="text-sm text-gray-600 dark:text-gray-400">Peak Down</p>
            <p className="text-lg font-bold text-gray-900 dark:text-white">
              {peakStats?.peak_rx_string || "N/A"}
            </p>
          </div>

          {/* Peak Upload */}
          <div className="text-center">
            <ArrowUpIcon className="h-8 w-8 mx-auto mb-2 text-green-500" />
            <p className="text-sm text-gray-600 dark:text-gray-400">Peak Up</p>
            <p className="text-lg font-bold text-gray-900 dark:text-white">
              {peakStats?.peak_tx_string || "N/A"}
            </p>
          </div>
        </div>
      </motion.div>
    </div>
  );
};
