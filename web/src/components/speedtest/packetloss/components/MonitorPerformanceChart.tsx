/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { PacketLossResult } from "@/types/types";

interface MonitorPerformanceChartProps {
  historyList: PacketLossResult[];
  selectedMonitorId: number;
}

export const MonitorPerformanceChart: React.FC<
  MonitorPerformanceChartProps
> = ({ historyList, selectedMonitorId }) => {
  // Prepare chart data - use useMemo to ensure it updates when historyList changes
  const chartData = useMemo(() => {
    // historyList is in descending order (newest first), so take first 30 and reverse
    const data = historyList
      .slice(0, 30) // First 30 results (most recent)
      .reverse() // Reverse to show oldest to newest for chart
      .map((result) => {
        const date = new Date(result.createdAt);

        // Use consistent formatting for all data points
        const timeLabel = date.toLocaleString(undefined, {
          month: "numeric",
          day: "numeric",
          hour: "numeric",
          minute: "2-digit",
        });

        return {
          time: timeLabel,
          packetLoss: result.packetLoss,
          avgRtt: result.avgRtt,
          minRtt: result.minRtt,
          maxRtt: result.maxRtt,
        };
      });

    return data;
  }, [historyList]);

  if (chartData.length === 0) {
    return null;
  }

  return (
    <div className="mb-6">
      <div className="mb-4">
        <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-2">
          Performance Trends
        </h3>
        <div className="flex items-center justify-between">
          <p className="text-gray-600 dark:text-gray-400 text-xs">
            Last 30 tests • {chartData.length} data points
          </p>
          <div className="flex items-center gap-4 text-xs text-gray-600 dark:text-gray-400">
            <div className="flex items-center gap-1">
              <div className="w-3 h-0.5 bg-red-500"></div>
              <span>Packet Loss</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-3 h-0.5 bg-blue-500"></div>
              <span>Avg RTT</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-3 h-0.5 bg-emerald-500 border-dashed"></div>
              <span>Min RTT</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-3 h-0.5 bg-yellow-500 border-dashed"></div>
              <span>Max RTT</span>
            </div>
          </div>
        </div>
      </div>

      <div className="h-80 bg-white/50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200/50 dark:border-gray-700/50">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            data={chartData}
            key={`chart-${selectedMonitorId}-${historyList[0]?.id || 0}`}
          >
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="rgba(128, 128, 128, 0.15)"
              strokeWidth={0.5}
            />
            <XAxis
              dataKey="time"
              stroke="rgb(156, 163, 175)"
              fontSize={11}
              axisLine={false}
              tickLine={false}
              dy={10}
            />
            <YAxis
              yAxisId="left"
              orientation="left"
              stroke="rgb(156, 163, 175)"
              fontSize={11}
              axisLine={false}
              tickLine={false}
              label={{
                value: "RTT (ms)",
                angle: -90,
                position: "insideLeft",
                style: {
                  fill: "rgb(156, 163, 175)",
                  textAnchor: "middle",
                },
              }}
            />
            <YAxis
              yAxisId="right"
              orientation="right"
              stroke="rgb(239, 68, 68)"
              fontSize={11}
              axisLine={false}
              tickLine={false}
              label={{
                value: "Packet Loss (%)",
                angle: 90,
                position: "insideRight",
                style: {
                  fill: "rgb(239, 68, 68)",
                  textAnchor: "middle",
                },
              }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "rgba(17, 24, 39, 0.95)",
                border: "1px solid rgba(75, 85, 99, 0.3)",
                borderRadius: "0.5rem",
                boxShadow: "0 10px 15px -3px rgba(0, 0, 0, 0.1)",
              }}
              labelStyle={{
                color: "rgb(229, 231, 235)",
                fontSize: "12px",
                fontWeight: "medium",
              }}
              formatter={(value: number | string) => {
                if (typeof value === "number") {
                  return value.toFixed(1);
                }
                return value;
              }}
            />
            <Line
              yAxisId="right"
              type="monotone"
              dataKey="packetLoss"
              stroke="rgb(239, 68, 68)"
              strokeWidth={2.5}
              name="Packet Loss %"
              dot={{
                fill: "rgb(239, 68, 68)",
                strokeWidth: 0,
                r: 3,
              }}
              activeDot={{
                r: 5,
                stroke: "rgb(239, 68, 68)",
                strokeWidth: 2,
              }}
            />
            <Line
              yAxisId="left"
              type="monotone"
              dataKey="avgRtt"
              stroke="rgb(59, 130, 246)"
              strokeWidth={2.5}
              name="Avg RTT"
              dot={{
                fill: "rgb(59, 130, 246)",
                strokeWidth: 0,
                r: 3,
              }}
              activeDot={{
                r: 5,
                stroke: "rgb(59, 130, 246)",
                strokeWidth: 2,
              }}
            />
            <Line
              yAxisId="left"
              type="monotone"
              dataKey="minRtt"
              stroke="rgb(16, 185, 129)"
              strokeWidth={1.5}
              strokeDasharray="4 4"
              name="Min RTT"
              dot={false}
              activeDot={{
                r: 4,
                stroke: "rgb(16, 185, 129)",
                strokeWidth: 2,
              }}
            />
            <Line
              yAxisId="left"
              type="monotone"
              dataKey="maxRtt"
              stroke="rgb(251, 191, 36)"
              strokeWidth={1.5}
              strokeDasharray="4 4"
              name="Max RTT"
              dot={false}
              activeDot={{
                r: 4,
                stroke: "rgb(251, 191, 36)",
                strokeWidth: 2,
              }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
