/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useMemo, useEffect } from "react";
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
import { getHistory, getPublicHistory } from "@/api/speedtest";
import { motion, AnimatePresence } from "motion/react";
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "@/components/ui/collapsible";
import { ChevronDownIcon } from "lucide-react";
import { FaDownload, FaUpload, FaClock, FaWaveSquare, FaGripVertical } from "react-icons/fa";
import { Button } from "@/components/ui/Button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

interface SpeedHistoryChartProps {
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
  isPublic?: boolean;
  hasAnyTests?: boolean;
  hasCurrentRangeTests?: boolean;
  showDragHandle?: boolean;
  dragHandleRef?: (node: HTMLElement | null) => void;
  dragHandleListeners?: any;
  dragHandleClassName?: string;
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
    className="animate-ping h-full w-full"
  >
    <div className="h-full w-full bg-gray-200/50 dark:bg-gray-800/50 rounded-lg" />
  </motion.div>
);

const useIsMobile = () => {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 640);
    };

    checkMobile();
    window.addEventListener("resize", checkMobile);

    return () => window.removeEventListener("resize", checkMobile);
  }, []);

  return isMobile;
};

export const SpeedHistoryChart: React.FC<SpeedHistoryChartProps> = ({
  timeRange = "1w",
  onTimeRangeChange,
  isPublic = false,
  hasAnyTests = false,
  hasCurrentRangeTests = false,
  showDragHandle = false,
  dragHandleRef,
  dragHandleListeners,
  dragHandleClassName,
}) => {
  const isMobile = useIsMobile();

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
      queryKey: ["history-chart", timeRange, 500, isPublic],
      queryFn: async ({ pageParam = 1 }) => {
        const historyFn = isPublic ? getPublicHistory : getHistory;
        const response = await historyFn(timeRange, pageParam, 500);
        return response;
      },
      getNextPageParam: (
        lastPage: PaginatedResponse<SpeedTestResult>,
        allPages
      ) => {
        // If the last page has no data, there are no more pages.
        if (!lastPage.data || lastPage.data.length === 0) {
          return undefined;
        }

        const totalFetched = allPages.reduce(
          (acc, page) => acc + (page.data?.length || 0),
          0
        );

        // If the total fetched is greater than or equal to the total available, no more pages.
        if (totalFetched >= (lastPage.total ?? 0)) {
          return undefined;
        }

        // Otherwise, return the next page number.
        return lastPage.page + 1;
      },
      initialPageParam: 1,
    });

  const filteredData = useMemo(() => {
    if (!data) return [];

    const allData = data.pages.flatMap(
      (page: PaginatedResponse<SpeedTestResult>) => page.data || []
    );

    return allData
      .filter((item): item is SpeedTestResult => item != null)
      .sort(
        (a, b) =>
          new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      )
      .map((item) => ({
        rawTimestamp: item.createdAt,
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
        serverName: item.serverName || "Unknown Server",
        serverHost: item.serverHost || item.serverName || "Unknown Server",
        testType: item.testType || "speedtest",
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
          margin={
            isMobile
              ? { top: 5, right: 5, left: 0, bottom: 5 }
              : { top: 5, right: 30, left: 20, bottom: 25 }
          }
        >
          <defs>
            <linearGradient id="downloadGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#60a5fa" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#60a5fa" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="uploadGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#34d399" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#34d399" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#fbbf24" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#fbbf24" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="jitterGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#c084fc" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#c084fc" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid
            strokeDasharray="3 3"
            horizontal={true}
            vertical={false}
            stroke="var(--chart-text)"
            opacity={0.3}
          />
          <XAxis
            dataKey="rawTimestamp"
            height={isMobile ? 50 : 60}
            tickMargin={isMobile ? 5 : 10}
            tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
            tickFormatter={(value) => {
              const date = new Date(value);

              // Dynamic formatting based on time range
              switch (timeRange) {
                case "1d":
                  // 24 hours: show time, with day name on desktop
                  if (isMobile) {
                    return date.toLocaleTimeString(undefined, {
                      hour: "numeric",
                      minute: "2-digit",
                    });
                  }
                  return date.toLocaleString(undefined, {
                    weekday: "short",
                    hour: "numeric",
                    minute: "2-digit",
                  });

                case "3d":
                  // 3 days: show day and time
                  if (isMobile) {
                    return date.toLocaleString(undefined, {
                      weekday: "short",
                      hour: "numeric",
                    });
                  }
                  return date.toLocaleString(undefined, {
                    weekday: "short",
                    hour: "numeric",
                    minute: "2-digit",
                  });

                case "1w":
                  // 1 week: show date with optional time on desktop
                  if (isMobile) {
                    return date.toLocaleString(undefined, {
                      month: "numeric",
                      day: "numeric",
                    });
                  }
                  return date.toLocaleString(undefined, {
                    weekday: "short",
                    month: "short",
                    day: "numeric",
                  });

                case "1m":
                  // 1 month: show date
                  if (isMobile) {
                    return date.toLocaleString(undefined, {
                      month: "short",
                      day: "numeric",
                    });
                  }
                  return date.toLocaleString(undefined, {
                    month: "short",
                    day: "numeric",
                  });

                case "all": {
                  // All time: show date with year if needed
                  const now = new Date();
                  const showYear = date.getFullYear() !== now.getFullYear();

                  if (isMobile) {
                    return date.toLocaleString(undefined, {
                      month: "short",
                      day: "numeric",
                      year: showYear ? "2-digit" : undefined,
                    });
                  }
                  return date.toLocaleString(undefined, {
                    month: "short",
                    day: "numeric",
                    year: showYear ? "numeric" : undefined,
                  });
                }

                default:
                  // Fallback to time-based format
                  return date.toLocaleTimeString(undefined, {
                    hour: "numeric",
                    minute: "2-digit",
                  });
              }
            }}
          />
          <YAxis
            yAxisId="speed"
            label={
              isMobile
                ? {
                    value: "Mbps",
                    position: "insideLeft",
                    angle: -90,
                    offset: 0,
                    style: {
                      textAnchor: "middle",
                      fill: "var(--chart-text)",
                      fontSize: 11,
                    },
                    dy: 40,
                  }
                : {
                    value: "Speed (Mbps)",
                    position: "insideLeft",
                    angle: -90,
                    offset: 0,
                    style: {
                      textAnchor: "middle",
                      fill: "var(--chart-text)",
                    },
                    dy: 0,
                  }
            }
            tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
            tickFormatter={(value) => (isMobile ? Math.round(value) : value)}
            width={isMobile ? 35 : 45}
            domain={[0, "auto"]}
            allowDataOverflow={false}
          />
          <YAxis
            yAxisId="latency"
            orientation="right"
            label={
              isMobile
                ? {
                    value: "ms",
                    position: "insideRight",
                    angle: -90,
                    offset: 0,
                    style: {
                      textAnchor: "middle",
                      fill: "var(--chart-text)",
                      fontSize: 11,
                    },
                    dy: 20,
                  }
                : {
                    value: "ms",
                    position: "insideRight",
                    angle: -90,
                    offset: 0,
                    style: { fill: "var(--chart-text)" },
                  }
            }
            tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
            tickFormatter={(value) => (isMobile ? Math.round(value) : value)}
            width={isMobile ? 30 : 45}
            domain={[0, "auto"]}
            allowDataOverflow={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--tooltip-bg)",
              border: "1px solid var(--tooltip-border)",
              borderRadius: "0.5rem",
              fontSize: isMobile ? "12px" : "14px",
              padding: isMobile ? "8px" : "12px",
              maxWidth: isMobile ? "250px" : "none",
            }}
            labelStyle={{
              color: "var(--tooltip-label)",
              fontSize: isMobile ? "11px" : "12px",
            }}
            itemStyle={{
              color: "var(--tooltip-text)",
              padding: isMobile ? "2px 0" : "4px 0",
            }}
            // Allow tooltip to work on touch devices
            trigger={isMobile ? "click" : "hover"}
            formatter={(value: number, name: string) => {
              if (name === "Download" || name === "Upload") {
                return [`${value.toFixed(isMobile ? 1 : 2)} Mbps`, name];
              }
              return [`${value.toFixed(isMobile ? 1 : 2)} ms`, name];
            }}
            labelFormatter={(label, payload) => {
              if (payload && payload.length > 0) {
                const data = payload[0].payload;
                const shouldRedact =
                  isPublic &&
                  (data.testType === "iperf3" ||
                    data.testType === "librespeed");

                const formattedDate =
                  data.timestamp ||
                  new Date(label).toLocaleString(undefined, {
                    month: "short",
                    day: "numeric",
                    hour: "numeric",
                    minute: "numeric",
                  });

                return (
                  <>
                    <span>{formattedDate}</span>
                    <br />
                    <span
                      style={{
                        color: "var(--tooltip-accent)",
                        fontSize: isMobile ? "11px" : "12px",
                        fontWeight: "500",
                      }}
                    >
                      {shouldRedact
                        ? "redacted host"
                        : data.serverHost || data.serverName}
                      {data.testType && (
                        <span
                          style={{
                            color: "var(--tooltip-muted)",
                            fontSize: isMobile ? "10px" : "11px",
                            marginLeft: "8px",
                          }}
                        >
                          {" "}
                          (
                          {data.testType === "speedtest"
                            ? "speedtest.net"
                            : data.testType}
                          )
                        </span>
                      )}
                    </span>
                  </>
                );
              }
              return label;
            }}
          />
          {visibleMetrics.download && (
            <Area
              key="download"
              yAxisId="speed"
              type="monotone"
              dataKey="download"
              name="Download"
              stroke="#60a5fa"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#downloadGradient)"
              className="!stroke-blue-400"
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
              stroke="#34d399"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#uploadGradient)"
              className="!stroke-emerald-400"
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
              stroke="#fbbf24"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#latencyGradient)"
              className="!stroke-amber-400"
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
              stroke="#c084fc"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#jitterGradient)"
              className="!stroke-purple-400"
              strokeDasharray="5 5"
              animationDuration={1750}
              animationBegin={0}
              isAnimationActive={true}
            />
          )}
        </AreaChart>
      </ResponsiveContainer>
    ),
    [filteredData, timeRange, visibleMetrics, isMobile, isPublic]
  );

  const [isOpen, setIsOpen] = useState(() => {
    const saved = localStorage.getItem("speedtest-chart-open");
    return saved === null ? true : saved === "true";
  });

  // Persist chart open state to localStorage
  useEffect(() => {
    localStorage.setItem("speedtest-chart-open", isOpen.toString());
  }, [isOpen]);

  // Persist time range to localStorage
  useEffect(() => {
    localStorage.setItem("speedtest-time-range", timeRange);
  }, [timeRange]);

  return (
    <div className="shadow-lg rounded-xl overflow-hidden mb-6">
      <Collapsible open={isOpen} onOpenChange={setIsOpen}>
        <div className="flex flex-col h-full">
        <CollapsibleTrigger asChild>
          <button
            className={cn(
              "flex justify-between items-center w-full px-4 py-3 sm:py-2 min-h-[44px] sm:min-h-0 bg-gray-50/95 dark:bg-gray-850/95",
              isOpen ? "rounded-t-xl" : "rounded-xl",
              "border border-gray-200 dark:border-gray-800",
              isOpen && "border-b-0",
              "text-left hover:bg-gray-100/95 dark:hover:bg-gray-800/95 transition-colors touch-manipulation"
            )}
          >
            <div className="flex items-center gap-2">
              {showDragHandle && (
                <div
                  ref={dragHandleRef}
                  {...dragHandleListeners}
                  className={cn("cursor-grab active:cursor-grabbing touch-none p-1 -m-1", dragHandleClassName)}
                >
                  <FaGripVertical className="w-4 h-4 text-gray-400 dark:text-gray-600" />
                </div>
              )}
              <h2 className="text-gray-900 dark:text-white text-lg sm:text-xl font-semibold p-1 select-none">
                Speedtest History
              </h2>
            </div>
            <div className="p-1 -m-1">
              <ChevronDownIcon
                className={cn(
                  "w-5 h-5 text-gray-600 dark:text-gray-400 transition-transform duration-200",
                  isOpen && "transform rotate-180"
                )}
              />
            </div>
          </button>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <motion.div
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{
              duration: 0.5,
              type: "spring",
              stiffness: 300,
              damping: 20,
            }}
            className="bg-gray-50/95 dark:bg-gray-850/95 px-2 sm:px-4 rounded-b-xl flex-1 border border-t-0 border-gray-200 dark:border-gray-800"
          >
            <div className="pt-2 sm:pt-3 pb-3 sm:pb-4">
                  {/* Controls */}
                  <div className="flex flex-col gap-3 mb-2">
                    {/* Metric Toggle Controls */}
                    <div className="grid grid-cols-2 sm:flex sm:flex-wrap items-center gap-2 sm:gap-2 w-full sm:w-auto">
                      {[
                        {
                          key: "download",
                          color: "#60a5fa",
                          label: "Download",
                          icon: <FaDownload size={14} />,
                        },
                        {
                          key: "upload",
                          color: "#34d399",
                          label: "Upload",
                          icon: <FaUpload size={14} />,
                        },
                        {
                          key: "latency",
                          color: "#fbbf24",
                          label: "Latency",
                          icon: <FaClock size={14} />,
                        },
                        {
                          key: "jitter",
                          color: "#c084fc",
                          label: "Jitter",
                          icon: <FaWaveSquare size={14} />,
                        },
                      ].map(({ key, label, icon, color }) => {
                        const isActive =
                          visibleMetrics[key as keyof typeof visibleMetrics];
                        return (
                          <button
                            key={key}
                            onClick={() =>
                              handleMetricToggle(
                                key as keyof typeof visibleMetrics
                              )
                            }
                            aria-label={`${isActive ? "Hide" : "Show"} ${label} metric`}
                            aria-pressed={isActive}
                            className={cn(
                              "relative px-2.5 sm:px-2 py-2 sm:py-1.5",
                              "text-[11px] sm:text-xs font-medium",
                              "flex items-center justify-center sm:justify-start gap-1.5 sm:gap-1.5",
                              "rounded-md sm:rounded-md transition-all duration-200",
                              "border border-transparent",
                              // Reduced touch target size for mobile
                              "min-h-[36px] sm:min-h-0",
                              "flex-1 sm:flex-initial",
                              isActive
                                ? "bg-opacity-20 shadow-sm"
                                : "bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700",
                              // Add active states for touch feedback
                              "active:scale-95 sm:active:scale-100"
                            )}
                            style={{
                              backgroundColor: isActive
                                ? `${color}15`
                                : undefined,
                              borderColor: isActive ? `${color}40` : undefined,
                              color: isActive ? color : undefined,
                            }}
                          >
                            <div
                              className={cn(
                                "w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full transition-all duration-200",
                                isActive ? "scale-110" : "scale-90 opacity-60"
                              )}
                              style={{
                                backgroundColor: isActive
                                  ? color
                                  : "currentColor",
                              }}
                            />
                            <span className="hidden sm:inline">{label}</span>
                            <div className="sm:hidden flex items-center gap-1">
                              {icon}
                              <span>{label}</span>
                            </div>
                          </button>
                        );
                      })}
                    </div>

                    {/* Desktop layout - Time Range Controls alongside metrics */}
                    <div className="hidden sm:flex sm:justify-end sm:-mt-12">
                      <Select value={timeRange} onValueChange={handleTimeRangeChange}>
                        <SelectTrigger className="w-[140px] px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-lg text-gray-700 dark:text-gray-300 shadow-md">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-lg">
                          {timeRangeOptions.map((option) => (
                            <SelectItem 
                              key={option.value} 
                              value={option.value}
                              className="hover:bg-gray-100 dark:hover:bg-gray-800 focus:bg-gray-100 dark:focus:bg-gray-800 text-gray-900 dark:text-gray-100"
                            >
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    {/* Mobile layout - Full width time selector */}
                    <div className="sm:hidden w-full">
                      <Select value={timeRange} onValueChange={handleTimeRangeChange}>
                        <SelectTrigger className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-lg text-gray-700 dark:text-gray-300 shadow-md">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-lg">
                          {timeRangeOptions.map((option) => (
                            <SelectItem 
                              key={option.value} 
                              value={option.value}
                              className="hover:bg-gray-100 dark:hover:bg-gray-800 focus:bg-gray-100 dark:focus:bg-gray-800 text-gray-900 dark:text-gray-100"
                            >
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  {/* Chart Area */}
                  <div className="h-[250px] sm:h-[300px] md:h-[400px] -mx-2 sm:mx-0">
                    <AnimatePresence mode="wait">
                      {isLoading ? (
                        <ChartSkeleton />
                      ) : hasAnyTests &&
                        !hasCurrentRangeTests &&
                        filteredData.length === 0 ? (
                        <motion.div
                          key="no-data-message"
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          exit={{ opacity: 0 }}
                          transition={{ duration: 0.3 }}
                          className="h-full flex items-center justify-center"
                        >
                          <div className="text-center">
                            <h3 className="text-gray-900 dark:text-white text-lg font-medium mb-2">
                              No tests in the last{" "}
                              {timeRange === "1d"
                                ? "24 hours"
                                : timeRange === "3d"
                                ? "3 days"
                                : timeRange === "1w"
                                ? "week"
                                : timeRange === "1m"
                                ? "month"
                                : "selected period"}
                            </h3>
                            <p className="text-gray-600 dark:text-gray-400">
                              Try selecting a different time range to view your
                              test history.
                            </p>
                          </div>
                        </motion.div>
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

                  {hasNextPage && !isLoading && filteredData.length > 0 && (
                    <div className="flex justify-end">
                      <motion.div
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                      >
                        <Button
                          onClick={() => fetchNextPage()}
                          disabled={isFetchingNextPage}
                          isLoading={isFetchingNextPage}
                          className="mb-4"
                        >
                          {isFetchingNextPage ? "Loading more..." : "Load more"}
                        </Button>
                      </motion.div>
                    </div>
                  )}
                </div>
              </motion.div>
            </CollapsibleContent>
        </div>
      </Collapsible>
    </div>
      );
    };
