/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo, useState } from "react";
import { motion } from "motion/react";
import { ArrowDownIcon, ArrowUpIcon, CalendarIcon, ChartBarIcon } from "@heroicons/react/24/outline";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { MonitorBandwidthChart } from "../MonitorBandwidthChart";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";
import { MiniSparkline } from "../MiniSparkline";
import { formatBytes } from "@/utils/formatBytes";

interface MonitorBandwidthTabProps {
  agent: MonitorAgent;
}

export const MonitorBandwidthTab: React.FC<MonitorBandwidthTabProps> = ({ agent }) => {
  const [selectedTimeRange, setSelectedTimeRange] = useState<"6h" | "12h" | "24h" | "48h" | "7d" | "30d">("24h");
  const { nativeData, status, peakStats } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includePeakStats: true,
  });

  // Extract sparkline data for trend preview
  const sparklineData = useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.hour) {
      return { rx: [], tx: [] };
    }

    const hourData = nativeData.interfaces[0].traffic.hour.slice(0, 24);

    return {
      rx: hourData.map((h) => h.rx).reverse(),
      tx: hourData.map((h) => h.tx).reverse(),
    };
  }, [nativeData]);

  // Process bandwidth data based on selected time range
  const chartData = useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic) {
      return { data: [], title: "No Data", timeFormat: "hour" as const };
    }

    const traffic = nativeData.interfaces[0].traffic;

    switch (selectedTimeRange) {
      case "6h": {
        if (!traffic.hour) return { data: [], title: "Hourly Bandwidth (6 hours)", timeFormat: "hour" as const };
        return {
          data: traffic.hour
            .slice(0, 6)
            .reverse()
            .map((hour) => ({
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
        if (!traffic.hour) return { data: [], title: "Hourly Bandwidth (12 hours)", timeFormat: "hour" as const };
        return {
          data: traffic.hour
            .slice(0, 12)
            .reverse()
            .map((hour) => ({
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
        if (!traffic.hour) return { data: [], title: "Hourly Bandwidth (24 hours)", timeFormat: "hour" as const };
        return {
          data: traffic.hour
            .slice(0, 24)
            .reverse()
            .map((hour) => ({
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
        if (!traffic.hour) return { data: [], title: "Hourly Bandwidth (48 hours)", timeFormat: "hour" as const };
        return {
          data: traffic.hour
            .slice(0, 48)
            .reverse()
            .map((hour) => ({
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
        if (!traffic.day) return { data: [], title: "Daily Bandwidth (7 days)", timeFormat: "day" as const };
        return {
          data: traffic.day
            .slice(0, 7)
            .reverse()
            .map((day) => ({
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
        if (!traffic.day) return { data: [], title: "Daily Bandwidth (30 days)", timeFormat: "day" as const };
        return {
          data: traffic.day
            .slice(0, 30)
            .reverse()
            .map((day) => ({
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
      <div className="text-center py-12">
        <p className="text-lg text-gray-500 dark:text-gray-400">Loading bandwidth data...</p>
      </div>
    );
  }

  const isOffline = !status?.connected;
  const isFromCache = nativeData?.from_cache || false;

  return (
    <div className="space-y-6">
      {/* Description or Offline Banner */}
      {isOffline && isFromCache ? (
        <MonitorOfflineBanner message="Showing cached bandwidth data. Real-time monitoring unavailable." />
      ) : (
        <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
          <h4 className="text-sm font-medium text-blue-600 dark:text-blue-400 mb-2">
            Real-time Bandwidth Monitoring
          </h4>
          <p className="text-sm text-blue-600 dark:text-blue-400">
            Network bandwidth data collected from <code className="bg-blue-500/10 px-1 rounded text-xs">vnstat</code> running on the remote agent. 
            Data shows actual bytes transferred during each time period. Use the time range buttons to view different periods of activity.
          </p>
        </div>
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
        <div className="text-center py-12 bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800">
          <p className="text-lg text-gray-500 dark:text-gray-400">No bandwidth data available</p>
          <p className="text-sm text-gray-400 dark:text-gray-500 mt-2">
            The agent may have just been added or monitor hasn't collected enough data yet.
          </p>
        </div>
      )}

      {/* Trend Preview with Sparklines */}
      {sparklineData.rx.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center space-x-3 mb-4">
            <ChartBarIcon className="h-6 w-6 text-gray-400" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              24 Hour Trend Preview
            </h3>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
            <div>
              <div className="flex items-center justify-between mb-2">
                <p className="text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-2">
                  <ArrowDownIcon className="h-4 w-4 text-blue-500" />
                  <span>Download Trend</span>
                </p>
                {nativeData?.interfaces?.[0]?.traffic?.hour?.[0] && (
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    Current: {formatBytes(nativeData.interfaces[0].traffic.hour[0].rx)}/h
                  </span>
                )}
              </div>
              <MiniSparkline data={sparklineData.rx} color="blue" height={48} />
            </div>
            <div>
              <div className="flex items-center justify-between mb-2">
                <p className="text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-2">
                  <ArrowUpIcon className="h-4 w-4 text-green-500" />
                  <span>Upload Trend</span>
                </p>
                {nativeData?.interfaces?.[0]?.traffic?.hour?.[0] && (
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    Current: {formatBytes(nativeData.interfaces[0].traffic.hour[0].tx)}/h
                  </span>
                )}
              </div>
              <MiniSparkline data={sparklineData.tx} color="green" height={48} />
            </div>
          </div>
        </motion.div>
      )}

      {/* Total Statistics */}
      {nativeData?.interfaces?.[0]?.traffic && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.2 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center space-x-3 mb-6">
            <CalendarIcon className="h-6 w-6 text-purple-600 dark:text-purple-400" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              Total Statistics
            </h3>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {/* All-time totals */}
            {nativeData.interfaces[0].traffic.total && (
              <>
                <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
                  <div className="flex items-center space-x-2 mb-2">
                    <ArrowDownIcon className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    <span className="text-sm font-medium text-blue-600 dark:text-blue-400">All-time Download</span>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-white">
                    {formatBytes(nativeData.interfaces[0].traffic.total.rx)}
                  </p>
                </div>

                <div className="bg-emerald-500/10 border border-emerald-500/30 rounded-lg p-4">
                  <div className="flex items-center space-x-2 mb-2">
                    <ArrowUpIcon className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                    <span className="text-sm font-medium text-emerald-600 dark:text-emerald-400">All-time Upload</span>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-white">
                    {formatBytes(nativeData.interfaces[0].traffic.total.tx)}
                  </p>
                </div>

                <div className="bg-purple-500/10 border border-purple-500/30 rounded-lg p-4">
                  <div className="flex items-center space-x-2 mb-2">
                    <CalendarIcon className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                    <span className="text-sm font-medium text-purple-600 dark:text-purple-400">Combined Total</span>
                  </div>
                  <p className="text-xl font-bold text-gray-900 dark:text-white">
                    {formatBytes(nativeData.interfaces[0].traffic.total.rx + nativeData.interfaces[0].traffic.total.tx)}
                  </p>
                </div>
              </>
            )}
          </div>

          {/* Current periods */}
          <div className="grid gap-6 sm:grid-cols-2 mt-6">
            {/* Today */}
            {nativeData.interfaces[0].traffic.day?.[0] && (
              <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4">
                <h4 className="font-medium text-gray-900 dark:text-white mb-3">Today</h4>
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-2">
                      <ArrowDownIcon className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                      <span>Downloaded</span>
                    </span>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(nativeData.interfaces[0].traffic.day[0].rx)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-2">
                      <ArrowUpIcon className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
                      <span>Uploaded</span>
                    </span>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(nativeData.interfaces[0].traffic.day[0].tx)}
                    </span>
                  </div>
                  <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex justify-between items-center">
                      <span className="text-sm font-medium text-gray-900 dark:text-white">Total</span>
                      <span className="text-sm font-bold text-gray-900 dark:text-white">
                        {formatBytes(nativeData.interfaces[0].traffic.day[0].rx + nativeData.interfaces[0].traffic.day[0].tx)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* This Month */}
            {nativeData.interfaces[0].traffic.month?.[0] && (
              <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4">
                <h4 className="font-medium text-gray-900 dark:text-white mb-3">This Month</h4>
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-2">
                      <ArrowDownIcon className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                      <span>Downloaded</span>
                    </span>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(nativeData.interfaces[0].traffic.month[0].rx)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400 flex items-center space-x-2">
                      <ArrowUpIcon className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
                      <span>Uploaded</span>
                    </span>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {formatBytes(nativeData.interfaces[0].traffic.month[0].tx)}
                    </span>
                  </div>
                  <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex justify-between items-center">
                      <span className="text-sm font-medium text-gray-900 dark:text-white">Total</span>
                      <span className="text-sm font-bold text-gray-900 dark:text-white">
                        {formatBytes(nativeData.interfaces[0].traffic.month[0].rx + nativeData.interfaces[0].traffic.month[0].tx)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
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
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-6">
            Peak Times & Averages
          </h3>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
            {/* Peak Times */}
            <div className="space-y-4">
              <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">Peak Bandwidth</h4>
              
              {peakStats && (
                <div className="space-y-3">
                  <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-3">
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-blue-600 dark:text-blue-400 flex items-center space-x-2">
                        <ArrowDownIcon className="h-4 w-4" />
                        <span>Peak Download</span>
                      </span>
                      <span className="text-sm font-bold text-gray-900 dark:text-white">
                        {peakStats.peak_rx_string}
                      </span>
                    </div>
                  </div>

                  <div className="bg-emerald-500/10 border border-emerald-500/30 rounded-lg p-3">
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-emerald-600 dark:text-emerald-400 flex items-center space-x-2">
                        <ArrowUpIcon className="h-4 w-4" />
                        <span>Peak Upload</span>
                      </span>
                      <span className="text-sm font-bold text-gray-900 dark:text-white">
                        {peakStats.peak_tx_string}
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Averages */}
            <div className="space-y-4">
              <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">Daily Averages</h4>
              
              {nativeData.interfaces[0].traffic.day && nativeData.interfaces[0].traffic.day.length > 0 && (() => {
                const last7Days = nativeData.interfaces[0].traffic.day.slice(0, 7);
                const avgRx = last7Days.reduce((sum, day) => sum + day.rx, 0) / last7Days.length;
                const avgTx = last7Days.reduce((sum, day) => sum + day.tx, 0) / last7Days.length;
                const avgTotal = avgRx + avgTx;

                return (
                  <div className="space-y-3">
                    <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Avg Daily Download
                        </span>
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          {formatBytes(avgRx)}
                        </span>
                      </div>
                    </div>

                    <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Avg Daily Upload
                        </span>
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          {formatBytes(avgTx)}
                        </span>
                      </div>
                    </div>

                    <div className="bg-purple-500/10 border border-purple-500/30 rounded-lg p-3">
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
                    </div>
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