import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { useState, useEffect, useMemo } from "react";
import { cn } from "@/lib/utils";
import { motion, AnimatePresence } from "framer-motion";
import { useInfiniteQuery } from "@tanstack/react-query";
import { ChevronDownIcon } from "lucide-react";
import { FaGripVertical } from "react-icons/fa6";
// Simple media query hook replacement
const useMediaQuery = (query: string) => {
  const [matches, setMatches] = useState(false);
  
  useEffect(() => {
    const mediaQuery = window.matchMedia(query);
    setMatches(mediaQuery.matches);
    
    const handler = (e: MediaQueryListEvent) => setMatches(e.matches);
    mediaQuery.addEventListener('change', handler);
    
    return () => mediaQuery.removeEventListener('change', handler);
  }, [query]);
  
  return matches;
};
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

import { Button } from "@/components/ui/Button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface SpeedtestResult {
  id: string;
  download: number;
  upload: number;
  latency: number;
  jitter: number;
  created_at: string;
  rawTimestamp?: number;
  server?: {
    server_id: string;
    server_name: string;
  };
}

interface ApiResponse {
  results: SpeedtestResult[];
  hasMore: boolean;
  nextCursor?: string;
}

type TimeRange = "1d" | "3d" | "1w" | "1m" | "3m" | "6m" | "1y" | "all";

interface VisibleMetrics {
  download: boolean;
  upload: boolean;
  latency: boolean;
  jitter: boolean;
}

interface SpeedHistoryChartProps {
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
  isPublic?: boolean;
  showDragHandle?: boolean;
  dragHandleRef?: any;
  dragHandleListeners?: any;
  dragHandleClassName?: string;
}

type ServerFilterMode = "all" | "single" | "multiple";

interface ServerInfo {
  id: string;
  name: string;
}

export const SpeedHistoryChart = ({
  timeRange,
  onTimeRangeChange,
  isPublic = false,
  showDragHandle = false,
  dragHandleRef,
  dragHandleListeners,
  dragHandleClassName = "",
}: SpeedHistoryChartProps) => {
  const isMobile = useMediaQuery("(max-width: 768px)");
  const [visibleMetrics, setVisibleMetrics] = useState<VisibleMetrics>(() => {
    const saved = localStorage.getItem("speedtest-visible-metrics");
    return saved
      ? JSON.parse(saved)
      : { download: true, upload: true, latency: true, jitter: false };
  });

  // Server filtering state
  const [serverFilterMode, setServerFilterMode] = useState<ServerFilterMode>("all");
  const [selectedSingleServer, setSelectedSingleServer] = useState<string>("all");
  const [selectedMultipleServers, setSelectedMultipleServers] = useState<Set<string>>(new Set());

  const fetchSpeedtests = async ({ pageParam }: { pageParam?: string }): Promise<ApiResponse> => {
    const url = `/api/speedtest?timeRange=${timeRange}${pageParam ? `&cursor=${pageParam}` : ""}`;
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error("Failed to fetch speedtest results");
    }
    const data = await response.json();
    
    // Add rawTimestamp for better chart performance
    const results = data.results.map((result: SpeedtestResult) => ({
      ...result,
      rawTimestamp: new Date(result.created_at).getTime(),
    }));

    return { ...data, results };
  };

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
    error,
  } = useInfiniteQuery({
    queryKey: ["speedtest-results", timeRange],
    queryFn: fetchSpeedtests,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialPageParam: undefined,
  });

  const allResults = useMemo(() => {
    return data?.pages.flatMap((page) => page.results) || [];
  }, [data]);

  // Extract available servers from results
  const availableServers = useMemo(() => {
    const servers = new Map<string, ServerInfo>();
    allResults.forEach(result => {
      if (result.server) {
        servers.set(result.server.server_id, {
          id: result.server.server_id,
          name: result.server.server_name || `Server ${result.server.server_id}`
        });
      }
    });
    return Array.from(servers.values()).sort((a, b) => a.name.localeCompare(b.name));
  }, [allResults]);

  // Filter data based on server selection
  const filteredData = useMemo(() => {
    let filtered = allResults;

    if (serverFilterMode === "single" && selectedSingleServer !== "all") {
      filtered = allResults.filter(result => result.server?.server_id === selectedSingleServer);
    } else if (serverFilterMode === "multiple" && selectedMultipleServers.size > 0) {
      // For multiple mode, we'll handle filtering per chart
      filtered = allResults.filter(result => 
        result.server && selectedMultipleServers.has(result.server.server_id)
      );
    }

    return filtered;
  }, [allResults, serverFilterMode, selectedSingleServer, selectedMultipleServers]);

  const hasAnyTests = allResults.length > 0;
  const hasCurrentRangeTests = filteredData.length > 0;

  // Persist visible metrics to localStorage
  useEffect(() => {
    localStorage.setItem("speedtest-visible-metrics", JSON.stringify(visibleMetrics));
  }, [visibleMetrics]);

  const toggleMetric = (metric: keyof VisibleMetrics) => {
    setVisibleMetrics((prev) => ({ ...prev, [metric]: !prev[metric] }));
  };

  const handleTimeRangeChange = (range: TimeRange) => {
    localStorage.setItem("speedtest-time-range", range);
    onTimeRangeChange(range);
  };

  const chart = useMemo(() => {
    // For multiple server mode, render individual charts
    if (serverFilterMode === "multiple" && selectedMultipleServers.size > 0) {
      return (
        <div className="space-y-6">
          {Array.from(selectedMultipleServers).map((serverId) => {
            const server = availableServers.find(s => s.id === serverId);
            const serverData = allResults.filter(result => result.server?.server_id === serverId);
            
            if (serverData.length === 0) return null;
            
            return (
              <div key={serverId} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                <h3 className="text-sm font-medium text-gray-800 dark:text-gray-200 mb-4">
                  {server?.name || `Server ${serverId}`}
                </h3>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart
                      data={serverData}
                      margin={
                        isMobile
                          ? { top: 5, right: 5, left: 0, bottom: 5 }
                          : { top: 5, right: 30, left: 20, bottom: 25 }
                      }
                    >
                      <defs>
                        <linearGradient id={`downloadGradient-${serverId}`} x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#60a5fa" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#60a5fa" stopOpacity={0} />
                        </linearGradient>
                        <linearGradient id={`uploadGradient-${serverId}`} x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#34d399" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#34d399" stopOpacity={0} />
                        </linearGradient>
                        <linearGradient id={`latencyGradient-${serverId}`} x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#fbbf24" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#fbbf24" stopOpacity={0} />
                        </linearGradient>
                        <linearGradient id={`jitterGradient-${serverId}`} x1="0" y1="0" x2="0" y2="1">
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
                          switch (timeRange) {
                            case "1d":
                              return date.toLocaleTimeString(undefined, {
                                hour: "numeric",
                                minute: "2-digit",
                              });
                            case "3d":
                            case "1w":
                              return date.toLocaleDateString(undefined, {
                                month: "numeric",
                                day: "numeric",
                              });
                            default:
                              return date.toLocaleDateString();
                          }
                        }}
                      />
                      <YAxis
                        yAxisId="speed"
                        orientation="left"
                        tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
                        label={{ value: "Speed (Mbps)", angle: -90, position: "insideLeft" }}
                      />
                      <YAxis
                        yAxisId="latency"
                        orientation="right"
                        tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
                        label={{ value: "Latency (ms)", angle: 90, position: "insideRight" }}
                      />
                      <Tooltip
                        labelFormatter={(value) => new Date(value).toLocaleString()}
                        formatter={(value: number, name: string) => [
                          name === "latency" || name === "jitter"
                            ? `${value.toFixed(1)} ms`
                            : `${value.toFixed(2)} Mbps`,
                          name.charAt(0).toUpperCase() + name.slice(1),
                        ]}
                        contentStyle={{
                          backgroundColor: "var(--tooltip-bg)",
                          border: "1px solid var(--tooltip-border)",
                          borderRadius: "8px",
                          fontSize: "12px",
                        }}
                      />
                      {visibleMetrics.download && (
                        <Area
                          yAxisId="speed"
                          type="monotone"
                          dataKey="download"
                          name="Download"
                          stroke="#3b82f6"
                          strokeWidth={2}
                          dot={false}
                          activeDot={{ r: 6 }}
                          fill={`url(#downloadGradient-${serverId})`}
                          className="!stroke-blue-500"
                        />
                      )}
                      {visibleMetrics.upload && (
                        <Area
                          yAxisId="speed"
                          type="monotone"
                          dataKey="upload"
                          name="Upload"
                          stroke="#10b981"
                          strokeWidth={2}
                          dot={false}
                          activeDot={{ r: 6 }}
                          fill={`url(#uploadGradient-${serverId})`}
                          className="!stroke-emerald-500"
                        />
                      )}
                      {visibleMetrics.latency && (
                        <Area
                          yAxisId="latency"
                          type="monotone"
                          dataKey="latency"
                          name="Latency"
                          stroke="#f59e0b"
                          strokeWidth={2}
                          dot={false}
                          activeDot={{ r: 6 }}
                          fill={`url(#latencyGradient-${serverId})`}
                          className="!stroke-amber-500"
                        />
                      )}
                      {visibleMetrics.jitter && (
                        <Area
                          yAxisId="latency"
                          type="monotone"
                          dataKey="jitter"
                          name="Jitter"
                          stroke="#c084fc"
                          strokeWidth={2}
                          dot={false}
                          activeDot={{ r: 6 }}
                          fill={`url(#jitterGradient-${serverId})`}
                          className="!stroke-purple-400"
                          strokeDasharray="5 5"
                        />
                      )}
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </div>
            );
          })}
        </div>
      );
    }

    // Single chart for all or single server
    return (
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
                    month: "numeric",
                    day: "numeric",
                  });

                case "1m":
                case "3m":
                case "6m":
                case "1y":
                case "all":
                default:
                  // Longer periods: show date
                  if (isMobile) {
                    return date.toLocaleString(undefined, {
                      month: "short",
                      day: "numeric",
                    });
                  }
                  return date.toLocaleString(undefined, {
                    month: "short",
                    day: "numeric",
                    year: "2-digit",
                  });
              }
            }}
          />
          <YAxis
            yAxisId="speed"
            orientation="left"
            tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
            axisLine={{ stroke: "var(--chart-text)" }}
            tickLine={{ stroke: "var(--chart-text)" }}
            label={{
              value: "Speed (Mbps)",
              angle: -90,
              position: "insideLeft",
              style: { textAnchor: "middle", fill: "var(--chart-text)" },
            }}
          />
          <YAxis
            yAxisId="latency"
            orientation="right"
            tick={{ fontSize: isMobile ? 11 : 12, fill: "var(--chart-text)" }}
            axisLine={{ stroke: "var(--chart-text)" }}
            tickLine={{ stroke: "var(--chart-text)" }}
            label={{
              value: "Latency (ms)",
              angle: 90,
              position: "insideRight",
              style: { textAnchor: "middle", fill: "var(--chart-text)" },
            }}
          />
          <Tooltip
            labelFormatter={(value) => new Date(value).toLocaleString()}
            formatter={(value: number, name: string) => [
              name === "latency" || name === "jitter"
                ? `${value.toFixed(1)} ms`
                : `${value.toFixed(2)} Mbps`,
              name.charAt(0).toUpperCase() + name.slice(1),
            ]}
            contentStyle={{
              backgroundColor: "var(--tooltip-bg)",
              border: "1px solid var(--tooltip-border)",
              borderRadius: "8px",
              fontSize: "12px",
            }}
          />
          {visibleMetrics.download && (
            <Area
              key="download"
              yAxisId="speed"
              type="monotone"
              dataKey="download"
              name="Download"
              stroke="#3b82f6"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#downloadGradient)"
              className="!stroke-blue-500"
              animationDuration={1500}
              animationBegin={0}
              isAnimationActive={true}
            />
          )}
          {visibleMetrics.upload && (
            <Area
              key="upload"
              yAxisId="speed"
              type="monotone"
              dataKey="upload"
              name="Upload"
              stroke="#10b981"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              fill="url(#uploadGradient)"
              className="!stroke-emerald-500"
              animationDuration={1750}
              animationBegin={0}
              isAnimationActive={true}
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
    );
  }, [filteredData, timeRange, visibleMetrics, isMobile, isPublic, serverFilterMode, selectedMultipleServers, availableServers]);

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
                "text-left transition-colors touch-manipulation"
              )}
            >
              <div className="flex items-center gap-2">
                {showDragHandle && (
                  <div
                    ref={dragHandleRef}
                    {...dragHandleListeners}
                    className={cn(
                      "cursor-grab active:cursor-grabbing touch-none p-1 -m-1",
                      dragHandleClassName
                    )}
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
              className="bg-gray-50/95 dark:bg-gray-850/95 px-3 sm:px-4 rounded-b-xl flex-1 border border-t-0 border-gray-200 dark:border-gray-800"
            >
              <div className="pt-2 sm:pt-0 pb-3 sm:pb-4">
                {/* Controls */}
                <div className="flex flex-col gap-3 mb-4">
                  {/* Metric Toggle Controls */}
                  <div className="grid grid-cols-2 sm:flex sm:flex-wrap items-center gap-2 sm:gap-2 w-full sm:w-auto">
                    {[
                      {
                        key: "download",
                        color: "#60a5fa",
                        label: "Download",
                      },
                      {
                        key: "upload",
                        color: "#34d399",
                        label: "Upload",
                      },
                      {
                        key: "latency",
                        color: "#fbbf24",
                        label: "Latency",
                      },
                      {
                        key: "jitter",
                        color: "#c084fc",
                        label: "Jitter",
                      },
                    ].map(({ key, color, label }) => (
                      <button
                        key={key}
                        onClick={() => toggleMetric(key as keyof VisibleMetrics)}
                        className={cn(
                          "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all min-h-[36px] sm:min-h-0",
                          visibleMetrics[key as keyof VisibleMetrics]
                            ? "bg-white dark:bg-gray-800 shadow-sm border border-gray-200 dark:border-gray-600"
                            : "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800",
                          "text-gray-700 dark:text-gray-300"
                        )}
                        style={{
                          borderLeft: visibleMetrics[key as keyof VisibleMetrics]
                            ? `3px solid ${color}`
                            : "3px solid transparent",
                        }}
                      >
                        <div
                          className="w-3 h-3 rounded-full flex-shrink-0"
                          style={{
                            backgroundColor: visibleMetrics[
                              key as keyof VisibleMetrics
                            ]
                              ? color
                              : "#9ca3af",
                          }}
                        />
                        <span className="whitespace-nowrap">{label}</span>
                      </button>
                    ))}
                  </div>

                  {/* Time Range Controls */}
                  <div className="grid grid-cols-4 sm:flex sm:flex-wrap gap-1 sm:gap-2">
                    {[
                      { key: "1d", label: "24h" },
                      { key: "3d", label: "3d" },
                      { key: "1w", label: "1w" },
                      { key: "1m", label: "1m" },
                      { key: "3m", label: "3m" },
                      { key: "6m", label: "6m" },
                      { key: "1y", label: "1y" },
                      { key: "all", label: "All" },
                    ].map(({ key, label }) => (
                      <button
                        key={key}
                        onClick={() => handleTimeRangeChange(key as TimeRange)}
                        className={cn(
                          "px-2 py-1 text-xs rounded-md transition-colors min-h-[32px] sm:min-h-0",
                          timeRange === key
                            ? "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border border-blue-300 dark:border-blue-600"
                            : "bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700",
                          "font-medium"
                        )}
                      >
                        {label}
                      </button>
                    ))}
                  </div>

                  {/* Server Filtering Controls */}
                  <div className="flex flex-col sm:flex-row gap-2 sm:gap-3 items-start sm:items-center">
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-gray-600 dark:text-gray-400 font-medium">Server Filter:</span>
                      
                      {/* Server Selection Dropdown */}
                      <Select
                        value={serverFilterMode === "all" ? "all" : selectedSingleServer}
                        onValueChange={(value) => {
                          if (value === "all") {
                            setServerFilterMode("all");
                            setSelectedSingleServer("all");
                          } else if (value === "multiple") {
                            setServerFilterMode("multiple");
                          } else {
                            setServerFilterMode("single");
                            setSelectedSingleServer(value);
                          }
                        }}
                      >
                        <SelectTrigger className="w-[200px] px-3 py-1.5 text-xs bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-600 rounded-lg">
                          <SelectValue placeholder="Select server..." />
                        </SelectTrigger>
                        <SelectContent className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-lg max-h-60">
                          <SelectItem value="all">All Servers</SelectItem>
                          <SelectItem value="multiple">Multiple Servers...</SelectItem>
                          {availableServers.map((server) => (
                            <SelectItem key={server.id} value={server.id}>
                              {server.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    {/* Multiple Server Selection */}
                    {serverFilterMode === "multiple" && (
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button 
                            variant="outline" 
                            size="sm" 
                            className="w-[200px] justify-between text-xs"
                          >
                            {selectedMultipleServers.size === 0 
                              ? "Select servers..." 
                              : `${selectedMultipleServers.size} selected`}
                            <ChevronDownIcon className="h-3 w-3" />
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-80 p-0" align="start">
                          <div className="max-h-60 overflow-y-auto p-2">
                            <div className="space-y-2">
                              {availableServers.map((server) => (
                                <div key={server.id} className="flex items-center space-x-2 p-1 hover:bg-gray-100 dark:hover:bg-gray-800 rounded">
                                  <Checkbox
                                    id={server.id}
                                    checked={selectedMultipleServers.has(server.id)}
                                    onCheckedChange={(checked) => {
                                      const newSelected = new Set(selectedMultipleServers);
                                      if (checked) {
                                        newSelected.add(server.id);
                                      } else {
                                        newSelected.delete(server.id);
                                      }
                                      setSelectedMultipleServers(newSelected);
                                    }}
                                  />
                                  <label 
                                    htmlFor={server.id}
                                    className="text-xs font-medium cursor-pointer flex-1"
                                  >
                                    {server.name}
                                  </label>
                                </div>
                              ))}
                            </div>
                            {availableServers.length === 0 && (
                              <div className="text-xs text-gray-500 p-2 text-center">
                                No servers available
                              </div>
                            )}
                          </div>
                        </PopoverContent>
                      </Popover>
                    )}
                  </div>
                </div>

                {/* Chart Container */}
                <div className="h-64 sm:h-80">
                  <AnimatePresence mode="wait">
                    {isError && error ? (
                      <motion.div
                        key="error"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.3 }}
                        className="h-full flex items-center justify-center"
                      >
                        <div className="text-center">
                          <h3 className="text-gray-900 dark:text-white text-lg font-medium mb-2">
                            Error loading speedtest data
                          </h3>
                          <p className="text-gray-600 dark:text-gray-400">
                            {(error as Error).message}
                          </p>
                        </div>
                      </motion.div>
                    ) : isLoading && !hasAnyTests ? (
                      <motion.div
                        key="loading"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.3 }}
                        className="h-full flex items-center justify-center"
                      >
                        <div className="text-center">
                          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4"></div>
                          <p className="text-gray-600 dark:text-gray-400">
                            Loading speedtest results...
                          </p>
                        </div>
                      </motion.div>
                    ) : !hasAnyTests ? (
                      <motion.div
                        key="no-tests"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.3 }}
                        className="h-full flex items-center justify-center"
                      >
                        <div className="text-center">
                          <h3 className="text-gray-900 dark:text-white text-lg font-medium mb-2">
                            No speedtest results found
                          </h3>
                          <p className="text-gray-600 dark:text-gray-400">
                            Run your first speedtest to see results here.
                          </p>
                        </div>
                      </motion.div>
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
