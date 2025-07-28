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
  CircleStackIcon,
  FireIcon,
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
  const { status, nativeData, hardwareStats, systemInfo } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeSystemInfo: true,
    includeHardwareStats: true,
    includePeakStats: false,
  });

  const usage = nativeData ? parseMonitorUsagePeriods(nativeData) : null;

  const isOffline = !status?.connected;

  return (
    <div className="space-y-6">
      {/* Offline Banner */}
      {isOffline && <MonitorOfflineBanner />}

      {/* System Identity Bar */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-4 shadow-lg border border-gray-200 dark:border-gray-800"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            {/* Hostname Icon */}
            <ServerIcon className="h-8 w-8 text-gray-600 dark:text-gray-400" />

            {/* System Info */}
            <div className="flex-1">
              <div className="flex items-baseline space-x-3">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {systemInfo?.hostname || agent.name}
                </h2>
              </div>
              {hardwareStats?.cpu && (
                <div className="flex items-center space-x-1.5 mt-0.5">
                  <CpuChipIcon className="h-4 w-4 text-gray-500 dark:text-gray-500" />
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {hardwareStats.cpu.model ? (
                      <>
                        {hardwareStats.cpu.model
                          .replace(/\s+/g, " ")
                          .trim()
                          .split("@")[0]
                          .trim()}
                        <span className="text-gray-500 dark:text-gray-500 mx-1">
                          •
                        </span>
                      </>
                    ) : null}
                    {hardwareStats.cpu.cores}{" "}
                    {hardwareStats.cpu.cores === 1 ? "core" : "cores"}
                    {hardwareStats.cpu.threads &&
                      hardwareStats.cpu.threads !== hardwareStats.cpu.cores && (
                        <>, {hardwareStats.cpu.threads} threads</>
                      )}
                    {systemInfo?.kernel && (
                      <>
                        {systemInfo.kernel.toLowerCase().includes("darwin") ? (
                          <FontAwesomeIcon
                            icon={faApple}
                            className="h-4 w-4 text-gray-500 dark:text-gray-500 ml-2 mr-1"
                          />
                        ) : systemInfo.kernel
                            .toLowerCase()
                            .includes("linux") ? (
                          <FontAwesomeIcon
                            icon={faLinux}
                            className="h-4 w-4 text-gray-500 dark:text-gray-500 ml-2 mr-1"
                          />
                        ) : null}
                        <span>{systemInfo.kernel}</span>
                      </>
                    )}
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Uptime */}
          <div className="flex items-center space-x-2 pr-4">
            <ClockIcon className="h-5 w-5 text-gray-400" />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {systemInfo?.uptime
                ? `Up ${formatUptime(systemInfo.uptime)}`
                : "N/A"}
            </span>
          </div>
        </div>
      </motion.div>

      {/* Network Performance Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Current Speed */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              Current Speed
            </h3>
            <SignalIcon className="h-5 w-5 text-gray-400" />
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
          transition={{ duration: 0.3, delay: 0.15 }}
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

        {/* This Week Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.2 }}
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

        {/* This Month Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.25 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              This Month
            </h3>
            <CalendarIcon className="h-5 w-5 text-gray-400" />
          </div>
          {usage?.["This Month"] ? (
            <>
              <p className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                {formatBytes(usage["This Month"].total)}
              </p>
              <div className="flex items-center space-x-4 text-sm">
                <div className="flex items-center space-x-1">
                  <ArrowDownIcon className="h-3 w-3 text-blue-500" />
                  <span className="text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["This Month"].download)}
                  </span>
                </div>
                <div className="flex items-center space-x-1">
                  <ArrowUpIcon className="h-3 w-3 text-green-500" />
                  <span className="text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["This Month"].upload)}
                  </span>
                </div>
              </div>
            </>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>
      </div>

      {/* Resource Monitors Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* CPU Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.3 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              CPU Usage
            </h3>
            <CpuChipIcon className="h-5 w-5 text-gray-400" />
          </div>
          {hardwareStats?.cpu ? (
            <div className="space-y-3">
              <div className="flex items-baseline justify-between">
                <span className="text-3xl font-bold text-gray-900 dark:text-white">
                  {hardwareStats.cpu.usage_percent.toFixed(1)}%
                </span>
                {hardwareStats.cpu.load_avg &&
                  hardwareStats.cpu.load_avg[0] && (
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      Load: {hardwareStats.cpu.load_avg[0].toFixed(2)}
                    </span>
                  )}
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                <div
                  className="h-3 rounded-full transition-all duration-500 ease-out"
                  style={{
                    width: `${Math.min(hardwareStats.cpu.usage_percent, 100)}%`,
                    backgroundColor:
                      hardwareStats.cpu.usage_percent < 50
                        ? "#34d399"
                        : hardwareStats.cpu.usage_percent < 85
                        ? "#d97706"
                        : "#EF4444",
                  }}
                />
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {hardwareStats.cpu.threads} threads @{" "}
                {hardwareStats.cpu.frequency?.toFixed(0) || "?"} MHz
              </p>
            </div>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>

        {/* Memory Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.35 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              Memory Usage
            </h3>
            <CircleStackIcon className="h-5 w-5 text-gray-400" />
          </div>
          {hardwareStats?.memory ? (
            <div className="space-y-3">
              <div className="flex items-baseline justify-between">
                <span className="text-3xl font-bold text-gray-900 dark:text-white">
                  {hardwareStats.memory.used_percent.toFixed(1)}%
                </span>
                <span className="text-sm text-gray-500 dark:text-gray-400">
                  {formatBytes(hardwareStats.memory.available)} free
                </span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                <div
                  className="h-3 rounded-full transition-all duration-500 ease-out"
                  style={{
                    width: `${Math.min(
                      hardwareStats.memory.used_percent,
                      100
                    )}%`,
                    backgroundColor:
                      hardwareStats.memory.used_percent < 50
                        ? "#34d399"
                        : hardwareStats.memory.used_percent < 85
                        ? "#d97706"
                        : "#EF4444",
                  }}
                />
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {formatBytes(hardwareStats.memory.used)} /{" "}
                {formatBytes(hardwareStats.memory.total)}
              </p>
            </div>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>

        {/* Disk Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.4 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              Primary Disk
            </h3>
            <ServerIcon className="h-5 w-5 text-gray-400" />
          </div>
          {hardwareStats?.disks && hardwareStats.disks.length > 0 ? (
            (() => {
              const primaryDisk =
                hardwareStats.disks.find((d) => d.path === "/") ||
                hardwareStats.disks[0];
              return (
                <div className="space-y-3">
                  <div className="flex items-baseline justify-between">
                    <span className="text-3xl font-bold text-gray-900 dark:text-white">
                      {primaryDisk.used_percent.toFixed(1)}%
                    </span>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      {formatBytes(primaryDisk.free)} free
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                    <div
                      className="h-3 rounded-full transition-all duration-500 ease-out"
                      style={{
                        width: `${Math.min(primaryDisk.used_percent, 100)}%`,
                        backgroundColor:
                          primaryDisk.used_percent < 70
                            ? "#10b981"
                            : primaryDisk.used_percent < 85
                            ? "#eab308"
                            : "#ef4444",
                      }}
                    />
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {primaryDisk.path} • {formatBytes(primaryDisk.used)} /{" "}
                    {formatBytes(primaryDisk.total)}
                  </p>
                </div>
              );
            })()
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>
      </div>

      {/* Temperature Alert (only if notable) */}
      {hardwareStats?.temperature &&
        hardwareStats.temperature.length > 0 &&
        (() => {
          const hotSensors = hardwareStats.temperature.filter(
            (t) => t.temperature > 80
          );
          const warmSensors = hardwareStats.temperature.filter(
            (t) => t.temperature > 60 && t.temperature <= 80
          );

          if (hotSensors.length > 0 || warmSensors.length > 0) {
            return (
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.25 }}
                className={`rounded-xl p-4 shadow-lg border ${
                  hotSensors.length > 0
                    ? "bg-red-50/95 dark:bg-red-900/20 border-red-200 dark:border-red-800"
                    : "bg-amber-50/95 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800"
                }`}
              >
                <div className="flex items-center space-x-3">
                  <FireIcon
                    className={`h-6 w-6 ${
                      hotSensors.length > 0
                        ? "text-red-600 dark:text-red-400"
                        : "text-amber-600 dark:text-amber-400"
                    }`}
                  />
                  <div className="flex-1">
                    <h4
                      className={`font-medium ${
                        hotSensors.length > 0
                          ? "text-red-900 dark:text-red-100"
                          : "text-amber-900 dark:text-amber-100"
                      }`}
                    >
                      Temperature Warning
                    </h4>
                    <div className="flex flex-wrap gap-3 mt-1">
                      {[...hotSensors, ...warmSensors].map((sensor, idx) => (
                        <span
                          key={idx}
                          className={`text-sm ${
                            sensor.temperature > 80
                              ? "text-red-700 dark:text-red-300"
                              : "text-amber-700 dark:text-amber-300"
                          }`}
                        >
                          {sensor.label || sensor.sensor_key}:{" "}
                          {sensor.temperature.toFixed(1)}°C
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              </motion.div>
            );
          }
          return null;
        })()}
    </div>
  );
};
