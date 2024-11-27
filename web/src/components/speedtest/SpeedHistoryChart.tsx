/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useMemo } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { SpeedTestResult, TimeRange, PaginatedResponse } from "@/types/types";
import { useInfiniteQuery } from "@tanstack/react-query";
import { fetchHistory } from "@/api/speedtest";
import { motion, AnimatePresence } from "motion/react";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";
import { FaDownload, FaUpload, FaClock, FaWaveSquare } from "react-icons/fa";

interface SpeedHistoryChartProps {
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
}

interface VisibleMetrics {
  download: boolean;
  upload: boolean;
  latency: boolean;
  jitter: boolean;
}

const timeRangeOptions: { value: TimeRange; label: string }[] = [
  { value: "1d", label: "24 Hours" },
  { value: "3d", label: "3 Days" },
  { value: "1w", label: "1 Week" },
  { value: "1m", label: "1 Month" },
  { value: "all", label: "All Time" },
];

const ChartSkeleton: React.FC = () => (
  <motion.div
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    exit={{ opacity: 0 }}
    className="animate-pulse h-full w-full"
  >
    <div className="h-full w-full bg-gray-800/50 rounded-lg" />
  </motion.div>
);

export const SpeedHistoryChart: React.FC<SpeedHistoryChartProps> = ({
  timeRange = "1w",
  onTimeRangeChange,
}) => {
  const [visibleMetrics, setVisibleMetrics] = useState<VisibleMetrics>(() => {
    const saved = localStorage.getItem("speedtest-visible-metrics");
    return saved
      ? JSON.parse(saved)
      : {
          download: true,
          upload: true,
          latency: true,
          jitter: true,
        };
  });

  const handleMetricToggle = (key: keyof VisibleMetrics) => {
    setVisibleMetrics((prev: VisibleMetrics) => {
      const newState = {
        ...prev,
        [key]: !prev[key],
      };
      localStorage.setItem(
        "speedtest-visible-metrics",
        JSON.stringify(newState)
      );
      return newState;
    });
  };

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useInfiniteQuery({
      queryKey: ["history", timeRange],
      queryFn: async ({ pageParam = 1 }) => {
        const response = await fetchHistory(timeRange, pageParam, 500);
        return response;
      },
      getNextPageParam: (lastPage: PaginatedResponse<SpeedTestResult>) => {
        if (lastPage.data.length < lastPage.limit) return undefined;
        return lastPage.page + 1;
      },
      initialPageParam: 1,
    });

  const filteredData = useMemo(() => {
    if (!data) return [];

    const allData = data.pages.flatMap(
      (page: PaginatedResponse<SpeedTestResult>) => page.data
    );

    return allData
      .sort(
        (a, b) =>
          new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      )
      .map((item) => ({
        timestamp: new Date(item.createdAt).toLocaleString(undefined, {
          month: "short",
          day: "numeric",
          hour: "numeric",
          minute: "numeric",
        }),
        download: Number(item.downloadSpeed) || 0,
        upload: Number(item.uploadSpeed) || 0,
        latency: Number(parseFloat(item.latency?.replace("ms", "")) || 0),
        jitter: Number(item.jitter) || 0,
      }))
      .filter(
        (item) =>
          !isNaN(item.download) &&
          !isNaN(item.upload) &&
          !isNaN(item.latency) &&
          !isNaN(item.jitter)
      );
  }, [data]);

  const handleTimeRangeChange = (range: TimeRange) => {
    localStorage.setItem("speedtest-time-range", range);
    onTimeRangeChange(range);
  };

  const chart = useMemo(
    () => (
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          key={`${timeRange}-${filteredData.length}`}
          data={filteredData}
          margin={{ top: 5, right: 30, left: 20, bottom: 25 }}
        >
          <defs>
            <linearGradient id="downloadGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3B82F6" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#3B82F6" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="uploadGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10B981" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#10B981" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#F59E0B" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#F59E0B" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="jitterGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#9333EA" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#9333EA" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid
            strokeDasharray="3 3"
            horizontal={true}
            vertical={false}
          />
          <XAxis dataKey="timestamp" height={60} tickMargin={10} />
          <YAxis
            yAxisId="speed"
            label={{
              value: "Speed (Mbps)",
              position: "insideLeft",
              angle: -90,
              offset: 0,
              style: {
                textAnchor: "middle",
                fill: "rgb(156 163 175)",
              },
              dy: 0,
            }}
            domain={[0, "auto"]}
            allowDataOverflow={false}
          />
          <YAxis
            yAxisId="latency"
            orientation="right"
            label={{
              value: "ms",
              position: "insideRight",
              angle: -90,
              offset: 0,
              className: "fill-gray-400",
            }}
            domain={[0, "auto"]}
            allowDataOverflow={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "#1F2937",
              border: "1px solid #374151",
              borderRadius: "0.5rem",
            }}
            labelStyle={{ color: "#9CA3AF" }}
            itemStyle={{ color: "#E5E7EB" }}
            formatter={(value: number, name: string) => {
              if (name === "Download" || name === "Upload") {
                return [`${value.toFixed(2)} Mbps`, name];
              }
              return [`${value.toFixed(2)} ms`, name];
            }}
          />
          {visibleMetrics.download && (
            <Area
              key="download"
              yAxisId="speed"
              type="monotone"
              dataKey="download"
              name="Download"
              stroke="#3B82F6"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#downloadGradient)"
              className="!stroke-blue-500"
              animationDuration={1750}
              animationBegin={0}
              isAnimationActive={true}
              strokeDasharray="0"
            />
          )}
          {visibleMetrics.upload && (
            <Area
              key="upload"
              yAxisId="speed"
              type="monotone"
              dataKey="upload"
              name="Upload"
              stroke="#10B981"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#uploadGradient)"
              className="!stroke-emerald-500"
              animationDuration={1750}
              animationBegin={0}
              isAnimationActive={true}
              strokeDasharray="0"
            />
          )}
          {visibleMetrics.latency && (
            <Area
              key="latency"
              yAxisId="latency"
              type="monotone"
              dataKey="latency"
              name="Latency"
              stroke="#F59E0B"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#latencyGradient)"
              className="!stroke-amber-500"
              strokeDasharray="3 3"
              animationDuration={1750}
              animationBegin={0}
              isAnimationActive={true}
            />
          )}
          {visibleMetrics.jitter && (
            <Area
              key="jitter"
              yAxisId="latency"
              type="monotone"
              dataKey="jitter"
              name="Jitter"
              stroke="#9333EA"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#jitterGradient)"
              className="!stroke-purple-500"
              strokeDasharray="5 5"
              animationDuration={1750}
              animationBegin={0}
              isAnimationActive={true}
            />
          )}
        </AreaChart>
      </ResponsiveContainer>
    ),
    [filteredData, timeRange, visibleMetrics]
  );

  return (
    <Disclosure defaultOpen={true}>
      {({ open }) => (
        <div className="flex flex-col h-full mb-6">
          <DisclosureButton
            className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
              open ? "rounded-t-xl border-b-0" : "rounded-xl"
            } shadow-lg border-b-0 border-gray-900 text-left`}
          >
            <h2 className="text-white text-xl font-semibold p-1 select-none">
              Speed History
            </h2>
            <ChevronDownIcon
              className={`${
                open ? "transform rotate-180" : ""
              } w-5 h-5 text-gray-400 transition-transform duration-200`}
            />
          </DisclosureButton>

          {open && (
            <div className="bg-gray-850/95 px-4 rounded-b-xl shadow-lg flex-1">
              <motion.div
                className="mt-1 speed-history-animate"
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{
                  duration: 0.5,
                  type: "spring",
                  stiffness: 300,
                  damping: 20,
                }}
              >
                {/* Controls */}
                <div className="flex flex-col md:flex-row justify-between items-center mb-6">
                  <div className="flex flex-wrap items-center gap-2 mb-4 md:mb-0">
                    {[
                      {
                        key: "download",
                        color: "#3B82F6",
                        label: "Download",
                        icon: <FaDownload />,
                      },
                      {
                        key: "upload",
                        color: "#10B981",
                        label: "Upload",
                        icon: <FaUpload />,
                      },
                      {
                        key: "latency",
                        color: "#F59E0B",
                        label: "Latency",
                        icon: <FaClock />,
                      },
                      {
                        key: "jitter",
                        color: "#9333EA",
                        label: "Jitter",
                        icon: <FaWaveSquare />,
                      },
                    ].map(({ key, label, icon, color }) => {
                      const isActive =
                        visibleMetrics[key as keyof typeof visibleMetrics];
                      return (
                        <motion.button
                          key={key}
                          whileHover={{ scale: 1.02 }}
                          whileTap={{ scale: 0.98 }}
                          onClick={() =>
                            handleMetricToggle(
                              key as keyof typeof visibleMetrics
                            )
                          }
                          className={`
                            px-3 py-1.5
                            rounded-md 
                            text-sm 
                            font-medium
                            flex items-center gap-2 
                            transition-all duration-150 ease-in-out
                            focus:outline-none
                            focus:ring-0
                            ${
                              isActive
                                ? `bg-opacity-20 border hover:bg-opacity-30`
                                : "bg-gray-800 text-gray-400 border border-gray-700 hover:bg-gray-750 hover:border-gray-600"
                            }
                          `}
                          style={{
                            backgroundColor: isActive
                              ? `${color}20`
                              : undefined,
                            borderColor: isActive ? color : undefined,
                            color: isActive ? color : undefined,
                          }}
                        >
                          <motion.div
                            layout
                            className="w-2 h-2 rounded-full"
                            style={{
                              backgroundColor: isActive ? color : "#6B7280",
                            }}
                          />
                          <span className="hidden md:inline">{label}</span>
                          <span className="md:hidden">{icon}</span>
                        </motion.button>
                      );
                    })}
                  </div>
                  <div className="flex gap-2">
                    {timeRangeOptions.map((option) => (
                      <motion.button
                        key={option.value}
                        whileHover={{ scale: 1.02 }}
                        whileTap={{ scale: 0.98 }}
                        onClick={() => handleTimeRangeChange(option.value)}
                        className={`px-3 py-2 rounded-lg text-sm transition-colors ${
                          timeRange === option.value
                            ? "bg-blue-500 text-white border border-blue-600 hover:border-blue-700"
                            : "bg-gray-800 text-gray-400 border border-gray-900/80 hover:bg-gray-700"
                        }`}
                      >
                        <span className="hidden md:inline">{option.label}</span>
                        <span className="md:hidden">
                          {option.label
                            .replace("Hours", "H")
                            .replace("Days", "D")
                            .replace("Week", "W")
                            .replace("Month", "M")
                            .replace("All Time", "All")}
                        </span>
                      </motion.button>
                    ))}
                  </div>
                </div>

                {/* Chart Area */}
                <div className="h-[300px] md:h-[400px]">
                  <AnimatePresence mode="wait">
                    {isLoading ? (
                      <ChartSkeleton />
                    ) : (
                      <motion.div
                        key="chart"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.3 }}
                        className="h-full"
                      >
                        {chart}
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                {hasNextPage && (
                  <motion.button
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    onClick={() => fetchNextPage()}
                    disabled={isFetchingNextPage}
                    className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400"
                  >
                    {isFetchingNextPage ? "Loading more..." : "Load more"}
                  </motion.button>
                )}
              </motion.div>
            </div>
          )}
        </div>
      )}
    </Disclosure>
  );
};
