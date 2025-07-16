/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { formatBytes } from "@/utils/formatBytes";
import { motion } from "motion/react";

interface BandwidthData {
  time: string;
  rx: number;
  tx: number;
  rxPeak?: number;
  txPeak?: number;
}

interface VnstatBandwidthChartProps {
  data: BandwidthData[];
  title: string;
  timeFormat?: "hour" | "day" | "month";
  showPeaks?: boolean;
}

export const VnstatBandwidthChart: React.FC<VnstatBandwidthChartProps> = ({
  data,
  title,
  timeFormat = "hour",
  showPeaks = false,
}) => {
  const [selectedTimeRange, setSelectedTimeRange] = useState<"24h" | "7d" | "30d" | "1y">("24h");

  const formatTooltipValue = (value: number) => {
    return formatBytes(value);
  };

  const formatAxisTick = (value: number) => {
    if (value === 0) return "0";
    const sizes = ["B", "KiB", "MiB", "GiB", "TiB"];
    const k = 1024;
    const i = Math.floor(Math.log(value) / Math.log(k));
    return `${Math.round(value / Math.pow(k, i))}${sizes[i]}`;
  };

  const formatXAxisTick = (tickItem: string) => {
    const date = new Date(tickItem);
    switch (timeFormat) {
      case "hour":
        return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
      case "day":
        return date.toLocaleDateString([], { month: "short", day: "numeric" });
      case "month":
        return date.toLocaleDateString([], { month: "short", year: "numeric" });
      default:
        return tickItem;
    }
  };

  const stats = React.useMemo(() => {
    if (!data || data.length === 0) return null;

    const rxValues = data.map(d => d.rx);
    const txValues = data.map(d => d.tx);
    
    // Calculate time period in seconds based on timeFormat
    const secondsPerPeriod = timeFormat === "hour" ? 3600 : 
                            timeFormat === "day" ? 86400 : 
                            timeFormat === "month" ? 2592000 : 3600; // default to hour
    
    // Calculate average rate (bytes per second) from total bytes per period
    const avgRxBytes = rxValues.reduce((a, b) => a + b, 0) / rxValues.length;
    const avgTxBytes = txValues.reduce((a, b) => a + b, 0) / txValues.length;
    
    return {
      avgRx: avgRxBytes / secondsPerPeriod, // Convert to bytes per second
      avgTx: avgTxBytes / secondsPerPeriod, // Convert to bytes per second
      maxRx: Math.max(...rxValues) / secondsPerPeriod, // Peak rate in bytes per second
      maxTx: Math.max(...txValues) / secondsPerPeriod, // Peak rate in bytes per second
      totalRx: rxValues.reduce((a, b) => a + b, 0),
      totalTx: txValues.reduce((a, b) => a + b, 0),
    };
  }, [data, timeFormat]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
    >
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white">
          {title}
        </h3>
        <div className="flex items-center space-x-2">
          {["24h", "7d", "30d", "1y"].map((range) => (
            <button
              key={range}
              onClick={() => setSelectedTimeRange(range as any)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                selectedTimeRange === range
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600"
              }`}
            >
              {range}
            </button>
          ))}
        </div>
      </div>

      {/* Statistics Summary */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-600 dark:text-gray-400">Avg Download</p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {formatBytes(stats.avgRx)}/s
            </p>
          </div>
          <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-600 dark:text-gray-400">Avg Upload</p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {formatBytes(stats.avgTx)}/s
            </p>
          </div>
          <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-600 dark:text-gray-400">Peak Download</p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {formatBytes(stats.maxRx)}/s
            </p>
          </div>
          <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-600 dark:text-gray-400">Peak Upload</p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {formatBytes(stats.maxTx)}/s
            </p>
          </div>
        </div>
      )}

      <div className="h-64 sm:h-80">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart
            data={data}
            margin={{ top: 10, right: 10, left: 10, bottom: 0 }}
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
              <linearGradient id="colorRxPeak" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#6366F1" stopOpacity={0.4} />
                <stop offset="95%" stopColor="#6366F1" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="colorTxPeak" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#34D399" stopOpacity={0.4} />
                <stop offset="95%" stopColor="#34D399" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" className="stroke-gray-300 dark:stroke-gray-700" />
            <XAxis
              dataKey="time"
              tickFormatter={formatXAxisTick}
              className="text-xs text-gray-600 dark:text-gray-400"
            />
            <YAxis
              tickFormatter={formatAxisTick}
              className="text-xs text-gray-600 dark:text-gray-400"
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "rgba(31, 41, 55, 0.95)",
                border: "1px solid rgba(75, 85, 99, 0.3)",
                borderRadius: "0.5rem",
              }}
              itemStyle={{ color: "#E5E7EB" }}
              formatter={formatTooltipValue}
              labelFormatter={(label) => new Date(label).toLocaleString()}
            />
            <Legend
              verticalAlign="top"
              height={36}
              iconType="rect"
              wrapperStyle={{ paddingBottom: "10px" }}
            />
            {showPeaks && data.some(d => d.rxPeak) && (
              <Area
                type="monotone"
                dataKey="rxPeak"
                stroke="#6366F1"
                fillOpacity={1}
                fill="url(#colorRxPeak)"
                name="Download Peak"
                strokeWidth={1}
              />
            )}
            {showPeaks && data.some(d => d.txPeak) && (
              <Area
                type="monotone"
                dataKey="txPeak"
                stroke="#34D399"
                fillOpacity={1}
                fill="url(#colorTxPeak)"
                name="Upload Peak"
                strokeWidth={1}
              />
            )}
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

      {/* Total Usage Summary */}
      {stats && (
        <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-800">
          <div className="flex justify-between items-center">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Total Usage ({selectedTimeRange})
            </p>
            <div className="flex items-center space-x-4">
              <span className="text-sm">
                <span className="text-blue-600 dark:text-blue-400 font-medium">
                  ↓ {formatBytes(stats.totalRx)}
                </span>
              </span>
              <span className="text-sm">
                <span className="text-green-600 dark:text-green-400 font-medium">
                  ↑ {formatBytes(stats.totalTx)}
                </span>
              </span>
              <span className="text-sm font-medium text-gray-900 dark:text-white">
                = {formatBytes(stats.totalRx + stats.totalTx)}
              </span>
            </div>
          </div>
        </div>
      )}
    </motion.div>
  );
};