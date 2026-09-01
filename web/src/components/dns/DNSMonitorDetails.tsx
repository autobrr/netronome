/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo } from "react";
import { motion } from "motion/react";
import { useInfiniteQuery } from "@tanstack/react-query";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import {
  ClockIcon,
  CheckCircleIcon,
  SignalIcon,
} from "@heroicons/react/24/outline";
import { DNSMonitor, DNSUpdate } from "@/types/types";
import { getDNSHistory } from "@/api/dns";
import { MetricCard } from "@/components/common/MetricCard";
import { formatters, formatDateTimeWithSettings } from "@/utils/timeSettings";
import { stateBadge } from "./constants";

interface DNSMonitorDetailsProps {
  monitor: DNSMonitor;
  status?: DNSUpdate;
}

export const DNSMonitorDetails: React.FC<DNSMonitorDetailsProps> = ({
  monitor,
  status,
}) => {
  const {
    data: history,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ["dns", "history", monitor.id],
    queryFn: ({ pageParam }) => getDNSHistory(monitor.id, pageParam),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((total, page) => total + page.data.length, 0);
      return loaded < (lastPage.total || 0) ? lastPage.page + 1 : undefined;
    },
    refetchInterval: 10000,
  });

  // A new result at the top can push a row into the next page, so drop the
  // repeats a refetch brings in.
  const results = useMemo(() => {
    const byID = new Map(
      (history?.pages.flatMap((page) => page.data) ?? []).map((result) => [
        result.id,
        result,
      ]),
    );
    return [...byID.values()];
  }, [history]);

  const totalCount = history?.pages[0]?.total ?? results.length;

  const chartData = useMemo(
    () =>
      results
        .slice(0, 30)
        .reverse()
        .map((result) => ({
          time: formatters.chartTick(new Date(result.createdAt), "1d"),
          responseTime: result.success ? result.responseTimeMs : null,
        })),
    [results],
  );

  // A disabled monitor polls no status, so the newest stored result stands in.
  const latest = results[0];
  // a live status replaces the stored result whole: its optional fields are
  // absent on purpose, so a field-by-field fallback would keep stale values
  const responseTimeMs = status ? status.responseTimeMs : latest?.responseTimeMs;
  const responseCode = status ? status.responseCode : latest?.responseCode;
  const success = status ? status.success : latest?.success;
  const error = status ? status.error : latest?.error;
  const badge = stateBadge(status?.state || monitor.lastState);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.1 }}
      className="flex-1"
    >
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
          {monitor.name || monitor.host} Details
        </h2>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
          <MetricCard
            icon={<ClockIcon className="w-5 h-5 text-blue-500" />}
            title="Latency"
            value={
              success && responseTimeMs !== undefined
                ? responseTimeMs.toFixed(1)
                : "—"
            }
            unit="ms"
            status="normal"
          />
          <MetricCard
            icon={<CheckCircleIcon className="w-5 h-5 text-emerald-500" />}
            title="Code"
            value={responseCode || "—"}
            unit=""
            status={responseCode && !success ? "error" : "normal"}
          />
          <MetricCard
            icon={<SignalIcon className="w-5 h-5 text-amber-500" />}
            title="State"
            value={badge?.label ?? "—"}
            unit=""
            status={
              badge?.label === "Down"
                ? "error"
                : badge
                  ? "success"
                  : "normal"
            }
          />
        </div>

        {error && (
          <div className="mb-6 p-3 rounded-lg border border-red-500/30 bg-red-500/10 text-sm text-red-600 dark:text-red-400 break-words">
            {error}
          </div>
        )}

        {chartData.length > 0 && (
          <div className="mb-6">
            <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-2">
              Latency History
            </h3>
            <p className="text-gray-600 dark:text-gray-400 text-xs mb-4">
              Last {chartData.length} checks • failed checks leave a gap
            </p>
            <div className="h-64 bg-white/50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200/50 dark:border-gray-700/50">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData}>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    stroke="var(--color-gray-500, #6b7280)"
                    strokeOpacity={0.15}
                    strokeWidth={0.5}
                  />
                  <XAxis
                    dataKey="time"
                    stroke="var(--color-gray-400, #9ca3af)"
                    fontSize={11}
                    axisLine={false}
                    tickLine={false}
                    dy={10}
                  />
                  <YAxis
                    stroke="var(--color-gray-400, #9ca3af)"
                    fontSize={11}
                    axisLine={false}
                    tickLine={false}
                    label={{
                      value: "Response (ms)",
                      angle: -90,
                      position: "insideLeft",
                      style: {
                        fill: "var(--color-gray-400, #9ca3af)",
                        textAnchor: "middle",
                      },
                    }}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "var(--tooltip-bg)",
                      border: "1px solid var(--tooltip-border)",
                      borderRadius: "0.5rem",
                      boxShadow: "0 10px 15px -3px rgba(0, 0, 0, 0.1)",
                    }}
                    labelStyle={{
                      color: "var(--tooltip-label)",
                      fontSize: "12px",
                      fontWeight: "medium",
                    }}
                    itemStyle={{ color: "var(--tooltip-text)" }}
                    formatter={(value) =>
                      typeof value === "number"
                        ? `${value.toFixed(1)} ms`
                        : String(value ?? "")
                    }
                  />
                  <Line
                    type="monotone"
                    dataKey="responseTime"
                    stroke="var(--color-blue-500, #3b82f6)"
                    strokeWidth={2.5}
                    name="Response Time"
                    connectNulls={false}
                    // the panel polls, and a redraw animation on every poll
                    // reads as a flicker
                    isAnimationActive={false}
                    dot={{
                      fill: "var(--color-blue-500, #3b82f6)",
                      strokeWidth: 0,
                      r: 3,
                    }}
                    activeDot={{
                      r: 5,
                      stroke: "var(--color-blue-500, #3b82f6)",
                      strokeWidth: 2,
                    }}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>
        )}

        <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-3">
          Recent Results {totalCount > 0 && `(${totalCount} total)`}
        </h3>

        {isLoading ? (
          <p className="text-sm text-gray-600 dark:text-gray-400">Loading...</p>
        ) : results.length === 0 ? (
          <p className="text-sm text-gray-600 dark:text-gray-400">
            No checks recorded yet.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-300 dark:border-gray-800">
                  <th className="text-left py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                    Time
                  </th>
                  <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                    Response
                  </th>
                  <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                    Code
                  </th>
                  <th className="text-left py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                    Error
                  </th>
                </tr>
              </thead>
              <tbody>
                {results.map((result) => (
                  <tr
                    key={result.id}
                    className="border-b border-gray-300/50 dark:border-gray-800/50 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors"
                  >
                    <td className="py-2 px-3 text-gray-700 dark:text-gray-300 whitespace-nowrap">
                      {formatDateTimeWithSettings(result.createdAt)}
                    </td>
                    <td
                      className={`py-2 px-3 text-center font-mono ${
                        result.success
                          ? "text-blue-600 dark:text-blue-400"
                          : "text-red-600 dark:text-red-400"
                      }`}
                    >
                      {result.success
                        ? `${result.responseTimeMs.toFixed(1)}ms`
                        : "failed"}
                    </td>
                    <td className="py-2 px-3 text-center text-gray-600 dark:text-gray-400">
                      {result.responseCode || "—"}
                    </td>
                    <td className="py-2 px-3 text-red-600 dark:text-red-400 break-words">
                      {result.error || ""}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {hasNextPage && (
              <div className="flex justify-center mt-4">
                <button
                  onClick={() => void fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="px-4 py-2 bg-gray-200/30 dark:bg-gray-800/30 border border-gray-300/50 dark:border-gray-900/50 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800/50 transition-colors"
                >
                  {isFetchingNextPage ? "Loading..." : "Load More"}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </motion.div>
  );
};
