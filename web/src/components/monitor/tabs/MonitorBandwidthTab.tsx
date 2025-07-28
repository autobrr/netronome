/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo, useState } from "react";
import { motion } from "motion/react";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CalendarIcon,
} from "@heroicons/react/24/outline";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { MonitorBandwidthChart } from "../MonitorBandwidthChart";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";
import { formatBytes } from "@/utils/formatBytes";

interface MonitorBandwidthTabProps {
  agent: MonitorAgent;
}

export const MonitorBandwidthTab: React.FC<MonitorBandwidthTabProps> = ({
  agent,
}) => {
  const [selectedTimeRange, setSelectedTimeRange] = useState<
    "6h" | "12h" | "24h" | "48h" | "7d" | "30d"
  >("24h");
  const { nativeData, status, peakStats } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includePeakStats: true,
  });

  // Process bandwidth data based on selected time range
  const chartData = useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic) {
      return { data: [], title: "No Data", timeFormat: "hour" as const };
    }

    const traffic = nativeData.interfaces[0].traffic;

    switch (selectedTimeRange) {
      case "6h": {
        if (!traffic.hour)
          return {
            data: [],
            title: "Hourly Bandwidth (6 hours)",
            timeFormat: "hour" as const,
          };
        return {
          data: traffic.hour.slice(-6).map((hour) => ({
            time: new Date(
              hour.date.year,
              hour.date.month - 1,
              hour.date.day || 1,
              hour.time?.hour || 0
            ).toISOString(),
            rx: hour.rx,
            tx: hour.tx,
          })),
          title: "Hourly Bandwidth (6 hours)",
          timeFormat: "hour" as const,
        };
      }
      case "12h": {
        if (!traffic.hour)
          return {
            data: [],
            title: "Hourly Bandwidth (12 hours)",
            timeFormat: "hour" as const,
          };
        return {
          data: traffic.hour.slice(-12).map((hour) => ({
            time: new Date(
              hour.date.year,
              hour.date.month - 1,
              hour.date.day || 1,
              hour.time?.hour || 0
            ).toISOString(),
            rx: hour.rx,
            tx: hour.tx,
          })),
          title: "Hourly Bandwidth (12 hours)",
          timeFormat: "hour" as const,
        };
      }
      case "24h": {
        if (!traffic.hour)
          return {
            data: [],
            title: "Hourly Bandwidth (24 hours)",
            timeFormat: "hour" as const,
          };
        return {
          data: traffic.hour.slice(-24).map((hour) => ({
            time: new Date(
              hour.date.year,
              hour.date.month - 1,
              hour.date.day || 1,
              hour.time?.hour || 0
            ).toISOString(),
            rx: hour.rx,
            tx: hour.tx,
          })),
          title: "Hourly Bandwidth (24 hours)",
          timeFormat: "hour" as const,
        };
      }
      case "48h": {
        if (!traffic.hour)
          return {
            data: [],
            title: "Hourly Bandwidth (48 hours)",
            timeFormat: "hour" as const,
          };
        return {
          data: traffic.hour.slice(-48).map((hour) => ({
            time: new Date(
              hour.date.year,
              hour.date.month - 1,
              hour.date.day || 1,
              hour.time?.hour || 0
            ).toISOString(),
            rx: hour.rx,
            tx: hour.tx,
          })),
          title: "Hourly Bandwidth (48 hours)",
          timeFormat: "hour" as const,
        };
      }
      case "7d": {
        if (!traffic.day)
          return {
            data: [],
            title: "Daily Bandwidth (7 days)",
            timeFormat: "day" as const,
          };
        return {
          data: traffic.day.slice(0, 7).map((day) => ({
            time: new Date(
              day.date.year,
              day.date.month - 1,
              day.date.day
            ).toISOString(),
            rx: day.rx,
            tx: day.tx,
          })),
          title: "Daily Bandwidth (7 days)",
          timeFormat: "day" as const,
        };
      }
      case "30d": {
        if (!traffic.day)
          return {
            data: [],
            title: "Daily Bandwidth (30 days)",
            timeFormat: "day" as const,
          };
        return {
          data: traffic.day.slice(0, 30).map((day) => ({
            time: new Date(
              day.date.year,
              day.date.month - 1,
              day.date.day
            ).toISOString(),
            rx: day.rx,
            tx: day.tx,
          })),
          title: "Daily Bandwidth (30 days)",
          timeFormat: "day" as const,
        };
      }
    }
  }, [nativeData, selectedTimeRange]);

  if (!nativeData) {
    return (
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3 }}
        className="text-center py-12"
      >
        <p className="text-lg text-gray-500 dark:text-gray-400">
          Loading bandwidth data...
        </p>
      </motion.div>
    );
  }

  const isOffline = !status?.connected;
  const isFromCache = nativeData?.from_cache || false;

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Offline Banner */}
      {isOffline && isFromCache && (
        <MonitorOfflineBanner message="Showing cached bandwidth data. Real-time monitoring unavailable." />
      )}

      {chartData.data.length > 0 ? (
        <MonitorBandwidthChart
          data={chartData.data}
          title={chartData.title}
          timeFormat={chartData.timeFormat}
          selectedTimeRange={selectedTimeRange}
          onTimeRangeChange={setSelectedTimeRange}
        />
      ) : (
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3 }}
          className="text-center py-8 sm:py-12 bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <p className="text-base sm:text-lg text-gray-500 dark:text-gray-400">
            No bandwidth data available
          </p>
          <p className="text-xs sm:text-sm text-gray-400 dark:text-gray-500 mt-1.5 sm:mt-2">
            The agent may have just been added or monitor hasn't collected
            enough data yet.
          </p>
        </motion.div>
      )}

      {/* Total Statistics */}
      {nativeData?.interfaces?.[0]?.traffic && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.2 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-4 sm:p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center space-x-2 sm:space-x-3 mb-4 sm:mb-6">
            <CalendarIcon className="h-5 w-5 sm:h-6 sm:w-6 text-purple-600 dark:text-purple-400" />
            <h3 className="text-base sm:text-lg font-medium text-gray-900 dark:text-white">
              Total Statistics
            </h3>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
            {/* All-time totals */}
            {nativeData.interfaces[0].traffic.total && (
              <>
                <motion.div
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ duration: 0.3, delay: 0.25 }}
                  className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4"
                >
                  <div className="flex items-center space-x-2 mb-2">
                    <ArrowDownIcon className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    <span className="text-sm font-medium text-blue-600 dark:text-blue-400">
                      All-time Download
                    </span>
                  </div>
                  <p className="text-lg sm:text-xl font-bold text-gray-900 dark:text-white">
                    {formatBytes(nativeData.interfaces[0].traffic.total.rx)}
                  </p>
                </motion.div>

                <motion.div
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ duration: 0.3, delay: 0.3 }}
                  className="bg-emerald-500/10 border border-emerald-500/30 rounded-lg p-4"
                >
                  <div className="flex items-center space-x-2 mb-2">
                    <ArrowUpIcon className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                    <span className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
                      All-time Upload
                    </span>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-white">
                    {formatBytes(nativeData.interfaces[0].traffic.total.tx)}
                  </p>
                </motion.div>

                <motion.div
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ duration: 0.3, delay: 0.35 }}
                  className="bg-purple-500/10 border border-purple-500/30 rounded-lg p-4"
                >
                  <div className="flex items-center space-x-2 mb-2">
                    <CalendarIcon className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                    <span className="text-sm font-medium text-purple-600 dark:text-purple-400">
                      Combined Total
                    </span>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-white">
                    {formatBytes(
                      nativeData.interfaces[0].traffic.total.rx +
                        nativeData.interfaces[0].traffic.total.tx
                    )}
                  </p>
                </motion.div>
              </>
            )}
          </div>

          {/* Current periods */}
          <div className="grid gap-6 sm:grid-cols-2 mt-6">
            {/* Today */}
            {nativeData.interfaces[0].traffic.day?.[0] && (
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.3, delay: 0.4 }}
                className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3 sm:p-4"
              >
                <h4 className="text-sm sm:text-base font-medium text-gray-900 dark:text-white mb-2 sm:mb-3">
                  Today
                </h4>
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-xs sm:text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-1.5 sm:space-x-2">
                      <ArrowDownIcon className="h-3.5 w-3.5 sm:h-4 sm:w-4 text-blue-600 dark:text-blue-400" />
                      <span>Downloaded</span>
                    </span>
                    <span className="text-xs sm:text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(nativeData.interfaces[0].traffic.day[0].rx)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-xs sm:text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-1.5 sm:space-x-2">
                      <ArrowUpIcon className="h-3.5 w-3.5 sm:h-4 sm:w-4 text-emerald-600 dark:text-emerald-400" />
                      <span>Uploaded</span>
                    </span>
                    <span className="text-xs sm:text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(nativeData.interfaces[0].traffic.day[0].tx)}
                    </span>
                  </div>
                  <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex justify-between items-center">
                      <span className="text-xs sm:text-sm font-medium text-gray-900 dark:text-white">
                        Total
                      </span>
                      <span className="text-xs sm:text-sm font-bold text-gray-900 dark:text-white">
                        {formatBytes(
                          nativeData.interfaces[0].traffic.day[0].rx +
                            nativeData.interfaces[0].traffic.day[0].tx
                        )}
                      </span>
                    </div>
                  </div>
                </div>
              </motion.div>
            )}

            {/* This Month */}
            {nativeData.interfaces[0].traffic.month?.[0] && (
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.3, delay: 0.45 }}
                className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3 sm:p-4"
              >
                <h4 className="text-sm sm:text-base font-medium text-gray-900 dark:text-white mb-2 sm:mb-3">
                  This Month
                </h4>
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-xs sm:text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-1.5 sm:space-x-2">
                      <ArrowDownIcon className="h-3.5 w-3.5 sm:h-4 sm:w-4 text-blue-600 dark:text-blue-400" />
                      <span>Downloaded</span>
                    </span>
                    <span className="text-xs sm:text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(
                        nativeData.interfaces[0].traffic.month[0].rx
                      )}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-xs sm:text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-1.5 sm:space-x-2">
                      <ArrowUpIcon className="h-3.5 w-3.5 sm:h-4 sm:w-4 text-emerald-600 dark:text-emerald-400" />
                      <span>Uploaded</span>
                    </span>
                    <span className="text-xs sm:text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(
                        nativeData.interfaces[0].traffic.month[0].tx
                      )}
                    </span>
                  </div>
                  <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex justify-between items-center">
                      <span className="text-xs sm:text-sm font-medium text-gray-900 dark:text-white">
                        Total
                      </span>
                      <span className="text-xs sm:text-sm font-bold text-gray-900 dark:text-white">
                        {formatBytes(
                          nativeData.interfaces[0].traffic.month[0].rx +
                            nativeData.interfaces[0].traffic.month[0].tx
                        )}
                      </span>
                    </div>
                  </div>
                </div>
              </motion.div>
            )}
          </div>
        </motion.div>
      )}

      {/* Peak Times and Averages */}
      {nativeData?.interfaces?.[0]?.traffic && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.3 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-4 sm:p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <h3 className="text-base sm:text-lg font-medium text-gray-900 dark:text-white mb-4 sm:mb-6">
            Peak Times & Averages
          </h3>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6">
            {/* Peak Times */}
            <div className="space-y-4">
              <h4 className="text-xs sm:text-sm font-medium text-gray-700 dark:text-gray-300">
                Peak Bandwidth
              </h4>

              {peakStats && (
                <div className="space-y-3">
                  <motion.div
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ duration: 0.3, delay: 0.35 }}
                    className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-3"
                  >
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-blue-600 dark:text-blue-400 flex items-center space-x-2">
                        <ArrowDownIcon className="h-4 w-4" />
                        <span>Peak Download</span>
                      </span>
                      <span className="text-sm font-bold text-gray-900 dark:text-white">
                        {peakStats.peak_rx_string}
                      </span>
                    </div>
                  </motion.div>

                  <motion.div
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ duration: 0.3, delay: 0.4 }}
                    className="bg-emerald-500/10 border border-emerald-500/30 rounded-lg p-3"
                  >
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-emerald-600 dark:text-emerald-400 flex items-center space-x-2">
                        <ArrowUpIcon className="h-4 w-4" />
                        <span>Peak Upload</span>
                      </span>
                      <span className="text-sm font-bold text-gray-900 dark:text-white">
                        {peakStats.peak_tx_string}
                      </span>
                    </div>
                  </motion.div>
                </div>
              )}
            </div>

            {/* Averages */}
            <div className="space-y-4">
              <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Daily Averages
              </h4>

              {nativeData.interfaces[0].traffic.day &&
                nativeData.interfaces[0].traffic.day.length > 0 &&
                (() => {
                  const last7Days = nativeData.interfaces[0].traffic.day.slice(
                    0,
                    7
                  );
                  const avgRx =
                    last7Days.reduce((sum, day) => sum + day.rx, 0) /
                    last7Days.length;
                  const avgTx =
                    last7Days.reduce((sum, day) => sum + day.tx, 0) /
                    last7Days.length;
                  const avgTotal = avgRx + avgTx;

                  return (
                    <div className="space-y-3">
                      <motion.div
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ duration: 0.3, delay: 0.45 }}
                        className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3"
                      >
                        <div className="flex justify-between items-center">
                          <span className="text-sm text-gray-600 dark:text-gray-400">
                            Avg Daily Download
                          </span>
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {formatBytes(avgRx)}
                          </span>
                        </div>
                      </motion.div>

                      <motion.div
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ duration: 0.3, delay: 0.5 }}
                        className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3"
                      >
                        <div className="flex justify-between items-center">
                          <span className="text-sm text-gray-600 dark:text-gray-400">
                            Avg Daily Upload
                          </span>
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {formatBytes(avgTx)}
                          </span>
                        </div>
                      </motion.div>

                      <motion.div
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ duration: 0.3, delay: 0.55 }}
                        className="bg-purple-500/10 border border-purple-500/30 rounded-lg p-3"
                      >
                        <div className="flex justify-between items-center">
                          <span className="text-sm text-purple-600 dark:text-purple-400 font-medium">
                            Avg Daily Total
                          </span>
                          <span className="text-sm font-bold text-gray-900 dark:text-white">
                            {formatBytes(avgTotal)}
                          </span>
                        </div>
                        <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                          Based on last 7 days
                        </p>
                      </motion.div>
                    </div>
                  );
                })()}
            </div>
          </div>
        </motion.div>
      )}
    </div>
  );
};
