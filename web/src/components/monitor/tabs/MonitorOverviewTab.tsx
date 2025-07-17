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
} from "@heroicons/react/24/outline";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { formatBytes } from "@/utils/formatBytes";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import { MiniSparkline } from "../MiniSparkline";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface MonitorOverviewTabProps {
  agent: MonitorAgent;
}

export const MonitorOverviewTab: React.FC<MonitorOverviewTabProps> = ({
  agent,
}) => {
  const { status, nativeData, hardwareStats } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeSystemInfo: true,
    includeHardwareStats: true,
  });

  const usage = nativeData ? parseMonitorUsagePeriods(nativeData) : null;

  // Extract hourly data for mini chart (last 24 hours)
  const chartData = React.useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.hour) {
      return [];
    }

    return nativeData.interfaces[0].traffic.hour
      .slice(0, 24)
      .reverse()
      .map((h) => ({
        time: new Date(
          h.date.year,
          h.date.month - 1,
          h.date.day || 1,
          h.time?.hour || 0
        ).toISOString(),
        rx: h.rx,
        tx: h.tx,
      }));
  }, [nativeData]);

  // Extract sparkline data
  const sparklineData = React.useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.hour) {
      return { rx: [], tx: [] };
    }

    const hourData = nativeData.interfaces[0].traffic.hour.slice(0, 24);

    return {
      rx: hourData.map((h) => h.rx).reverse(),
      tx: hourData.map((h) => h.tx).reverse(),
    };
  }, [nativeData]);

  const isOffline = !status?.connected;

  return (
    <div className="space-y-6">
      {/* Offline Banner */}
      {isOffline && <MonitorOfflineBanner />}

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
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
            <CpuChipIcon className="h-5 w-5 text-gray-400" />
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
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 24h Bandwidth Chart */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.4 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
            24 Hour Bandwidth
          </h3>
          {chartData.length > 0 ? (
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart
                  data={chartData}
                  margin={{ top: 10, right: 10, left: 0, bottom: 0 }}
                >
                  <defs>
                    <linearGradient id="colorRx" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3B82F6" stopOpacity={0.8} />
                      <stop offset="95%" stopColor="#3B82F6" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="colorTx" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#10B981" stopOpacity={0.8} />
                      <stop offset="95%" stopColor="#10B981" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    className="stroke-gray-300 dark:stroke-gray-700"
                  />
                  <XAxis
                    dataKey="time"
                    tickFormatter={(tick) =>
                      new Date(tick).toLocaleTimeString([], {
                        hour: "2-digit",
                        minute: "2-digit",
                      })
                    }
                    className="text-xs text-gray-600 dark:text-gray-400"
                  />
                  <YAxis
                    tickFormatter={(value) => {
                      const sizes = ["B", "KiB", "MiB", "GiB"];
                      const k = 1024;
                      const i = Math.floor(Math.log(value) / Math.log(k));
                      return `${Math.round(value / Math.pow(k, i))}${sizes[i]}`;
                    }}
                    className="text-xs text-gray-600 dark:text-gray-400"
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "rgba(31, 41, 55, 0.95)",
                      border: "1px solid rgba(75, 85, 99, 0.3)",
                      borderRadius: "0.5rem",
                    }}
                    labelStyle={{ color: "#E5E7EB" }}
                    formatter={(value: number) => formatBytes(value)}
                    labelFormatter={(label) => new Date(label).toLocaleString()}
                  />
                  <Area
                    type="monotone"
                    dataKey="rx"
                    stroke="#3B82F6"
                    fillOpacity={1}
                    fill="url(#colorRx)"
                    name="Download"
                    strokeWidth={2}
                  />
                  <Area
                    type="monotone"
                    dataKey="tx"
                    stroke="#10B981"
                    fillOpacity={1}
                    fill="url(#colorTx)"
                    name="Upload"
                    strokeWidth={2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="h-64 flex items-center justify-center">
              <p className="text-gray-500 dark:text-gray-400">
                No data available
              </p>
            </div>
          )}
        </motion.div>

        {/* Usage Summary */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.5 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
            Usage Summary
          </h3>
          {usage ? (
            <div className="space-y-3">
              {Object.entries(usage).map(([period, data]) => (
                <div
                  key={period}
                  className="flex items-center justify-between p-3 bg-gray-100 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-900"
                >
                  <div className="flex-1">
                    <p className="font-medium text-gray-900 dark:text-white">
                      {period}
                    </p>
                    <div className="flex items-center space-x-3 mt-1">
                      <div className="flex items-center space-x-1">
                        <ArrowDownIcon className="h-3 w-3 text-blue-500" />
                        <span className="text-xs text-gray-600 dark:text-gray-400">
                          {formatBytes((data as any).download)}
                        </span>
                      </div>
                      <div className="flex items-center space-x-1">
                        <ArrowUpIcon className="h-3 w-3 text-green-500" />
                        <span className="text-xs text-gray-600 dark:text-gray-400">
                          {formatBytes((data as any).upload)}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-gray-900 dark:text-white">
                      {formatBytes((data as any).total)}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">
              No usage data available
            </p>
          )}
        </motion.div>
      </div>

      {/* Sparklines */}
      {sparklineData.rx.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.6 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
            24 Hour Trend
          </h3>
          <div className="grid grid-cols-2 gap-6">
            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                Download
              </p>
              <MiniSparkline data={sparklineData.rx} color="blue" height={40} />
            </div>
            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                Upload
              </p>
              <MiniSparkline
                data={sparklineData.tx}
                color="green"
                height={40}
              />
            </div>
          </div>
        </motion.div>
      )}
    </div>
  );
};
