/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, useMemo } from "react";
import { Container } from "@mui/material";
import { FaWaveSquare } from "react-icons/fa";
import { IoIosPulse } from "react-icons/io";
import { FaArrowDown, FaArrowUp } from "react-icons/fa";
import { ServerList } from "./ServerList";
import { TestProgress } from "./TestProgress";
import { SpeedHistoryChart } from "./SpeedHistoryChart";
import ScheduleManager from "./ScheduleManager";
import {
  Server,
  SpeedTestResult,
  TestProgress as TestProgressType,
  TimeRange,
  TestOptions,
  PaginatedResponse,
  Schedule,
} from "@/types/types";
import logo from "@/assets/logo_small.png";
import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import {
  getServers,
  getHistory,
  getSchedules,
  runSpeedTest,
  getSpeedTestStatus,
} from "@/api/speedtest";
import { motion } from "motion/react";

export default function SpeedTest() {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [options, setOptions] = useState<TestOptions>({
    enableDownload: true,
    enableUpload: true,
    enablePacketLoss: true,
    enableJitter: true,
    multiServer: false,
    useIperf: false,
    useLibrespeed: false,
    serverIds: [],
  });
  const [testType, setTestType] = useState<
    "speedtest" | "iperf" | "librespeed"
  >("speedtest");
  const [selectedServers, setSelectedServers] = useState<Server[]>([]);
  const [progress, setProgress] = useState<TestProgressType | null>(null);
  const [testStatus, setTestStatus] = useState<"idle" | "running" | "complete">(
    "idle"
  );
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    const saved = localStorage.getItem("speedtest-time-range");
    return (saved as TimeRange) || "1w";
  });
  const [scheduledTestRunning] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  // Queries
  const { data: speedtestServers = [] } = useQuery({
    queryKey: ["servers", "speedtest"],
    queryFn: () => getServers("speedtest"),
  }) as { data: Server[] };

  const { data: librespeedServers = [] } = useQuery({
    queryKey: ["servers", "librespeed"],
    queryFn: () => getServers("librespeed"),
  }) as { data: Server[] };

  const allServers = useMemo(
    () => [...speedtestServers, ...librespeedServers],
    [speedtestServers, librespeedServers]
  );

  const servers = useMemo(() => {
    if (testType === "librespeed") return librespeedServers;
    if (testType === "iperf") return []; // or fetch iperf servers if they are separate
    return speedtestServers;
  }, [testType, speedtestServers, librespeedServers]);

  const { data: historyData, isLoading: isHistoryLoading } = useInfiniteQuery({
    queryKey: ["history", timeRange],
    queryFn: async ({ pageParam = 1 }) => {
      const response = await getHistory(timeRange, pageParam, 20);
      return response as PaginatedResponse<SpeedTestResult>;
    },
    getNextPageParam: (
      lastPage: PaginatedResponse<SpeedTestResult> | undefined
    ) => {
      if (!lastPage?.data) return undefined;
      if (lastPage.data.length < lastPage.limit) return undefined;
      return lastPage.page + 1;
    },
    initialPageParam: 1,
    staleTime: 0,
    placeholderData: (previousData) => previousData,
  });

  const history = useMemo(() => {
    if (!historyData?.pages) return [];
    return historyData.pages.flatMap(
      (page) => page?.data ?? []
    ) as SpeedTestResult[];
  }, [historyData]);

  const { data: schedules = [] } = useQuery({
    queryKey: ["schedules"],
    queryFn: getSchedules,
  }) as { data: Schedule[] };

  // Mutations
  const speedTestMutation = useMutation({
    mutationFn: runSpeedTest,
    onMutate: () => {
      console.log("Starting speed test with options:", options);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["history"] });
      queryClient.invalidateQueries({ queryKey: ["history-chart"] });
      setProgress(null);
      setTestStatus("complete");
    },
    onError: (error: Error) => {
      setError(error.message);
      setTestStatus("idle");
      console.error("Speed test error:", error);
    },
  });

  const handleServerSelect = (server: Server) => {
    setSelectedServers((prev) => {
      const isSelected = prev.some((s) => s.id === server.id);
      if (!options.multiServer) {
        return isSelected ? [] : [server];
      }
      return isSelected
        ? prev.filter((s) => s.id !== server.id)
        : [...prev, server];
    });
  };

  const runTest = async () => {
    if (selectedServers.length === 0) {
      setError("Please select at least one server");
      return;
    }

    setError(null);
    setTestStatus("running");
    setIsLoading(true);

    try {
      await speedTestMutation.mutateAsync({
        ...options,
        useIperf: testType === "iperf",
        useLibrespeed: testType === "librespeed",
        serverIds:
          testType === "speedtest" || testType === "librespeed"
            ? selectedServers.map((s) => s.id)
            : [],
        serverHost: testType === "iperf" ? selectedServers[0].host : undefined,
      });
    } catch (error) {
      console.error("Error running test:", error);
      setError(
        error instanceof Error ? error.message : "An unknown error occurred"
      );
      setTestStatus("idle");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (testStatus === "running") {
      const pollInterval = setInterval(async () => {
        try {
          const update = await getSpeedTestStatus();

          if (update) {
            setProgress((prev) => {
              const baseProgress = {
                currentServer: update.serverName || "",
                currentTest: update.type,
                currentSpeed: update.speed || 0,
                isComplete: update.isComplete,
                type: update.type,
                speed: update.speed || 0,
                latency:
                  typeof update.latency === "string"
                    ? parseFloat(update.latency)
                    : update.latency || 0,
                isScheduled: update.isScheduled,
                progress: update.progress || 0,
                isIperf: testType === "iperf",
                isLibrespeed: testType === "librespeed",
              };

              if (!prev) {
                return baseProgress;
              }

              const speedDiff = Math.abs(
                (prev.currentSpeed || 0) - (update.speed || 0)
              );
              if (speedDiff < 2.0) return prev;

              return {
                ...prev,
                ...baseProgress,
              };
            });

            if (update.isComplete && update.type !== "download") {
              setTestStatus("complete");
              setProgress(null);
              await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["history"] }),
                queryClient.invalidateQueries({ queryKey: ["history-chart"] }),
                queryClient.invalidateQueries({ queryKey: ["schedules"] }),
              ]);
            }
          }
        } catch (error) {
          console.error("Test status check error:", error);
        }
      }, 1000);

      return () => clearInterval(pollInterval);
    }
  }, [testStatus, queryClient, testType]);

  return (
    <div className="min-h-screen">
      <Container maxWidth="xl" className="pb-8">
        {/* Header */}
        <div className="flex flex-col md:flex-row justify-center md:justify-between mb-8 pt-8 relative">
          <div className="flex items-center gap-3">
            <img
              src={logo}
              alt="netronome Logo"
              className="h-12 w-12 select-none pointer-events-none"
              draggable="false"
            />
            <div>
              <h1 className="text-3xl font-bold text-white select-none">
                Netronome
              </h1>
              <h2 className="text-sm font-medium text-gray-300 select-none">
                Network Speed Testing
              </h2>
            </div>
          </div>
          {(testStatus === "running" || scheduledTestRunning) && (
            <div
              className="mt-8 md:mt-0 flex items-center justify-center"
              style={{ minWidth: "120px", height: "40px" }}
            >
              {progress !== null && <TestProgress progress={progress} />}
            </div>
          )}
        </div>
        {/* Error Messages */}
        {error && (
          <div className="bg-red-900/50 border border-red-700 text-red-200 px-4 py-3 rounded-xl mb-4">
            {error}
          </div>
        )}

        {/* No History Message - Only show when explicitly empty and not loading */}
        {!isHistoryLoading && (!history || history.length === 0) && (
          <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900 mb-6">
            <div className="text-center space-y-4">
              <div>
                <h2 className="text-white text-xl font-semibold mb-2">
                  No History Available
                </h2>
                <p className="text-gray-400">
                  Get started with your first speed test in two ways:
                </p>
              </div>
              <div className="flex justify-center gap-12 text-left text-gray-400">
                <div className="flex items-start gap-3">
                  <div className="mt-1 flex-shrink-0">
                    <FaArrowDown className="w-4 h-4 text-emerald-400" />
                  </div>
                  <div>
                    <p className="font-medium text-white">Run a test now</p>
                    <p className="text-sm">
                      Select a server below and start testing
                    </p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="mt-1 flex-shrink-0">
                    <IoIosPulse className="w-4 h-4 text-blue-400" />
                  </div>
                  <div>
                    <p className="font-medium text-white">
                      Schedule regular tests
                    </p>
                    <p className="text-sm">
                      Set up automated testing intervals
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Latest Results */}
        {history && history.length > 0 && history[0] && (
          <motion.div
            initial={{ opacity: 0, y: 20 }} // Initial state for animation
            animate={{ opacity: 1, y: 0 }} // Animate to this state
            exit={{ opacity: 0, y: 20 }} // Exit animation state
            transition={{ duration: 0.5 }} // Duration of the animation
            className="mb-6"
          >
            <h2 className="text-white text-xl ml-1 font-semibold">
              Latest Run
            </h2>
            <div className="flex justify-between ml-1 items-center text-gray-400 text-sm mb-4">
              <div>
                Last test run:{" "}
                {history[0]?.createdAt
                  ? new Date(history[0].createdAt).toLocaleString(undefined, {
                      dateStyle: "short",
                      timeStyle: "short",
                    })
                  : "N/A"}
              </div>
              {schedules && schedules.length > 0 && (
                <div>
                  Next scheduled run:{" "}
                  <span className="text-blue-400 mr-1">
                    {formatNextRun(schedules[0].nextRun)}
                  </span>
                </div>
              )}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6 cursor-default">
              <MetricCard
                icon={<IoIosPulse className="w-5 h-5 text-amber-500" />}
                title="Latency"
                value={parseFloat(history[0].latency).toFixed(2)}
                unit="ms"
                average={calculateAverage(history, "latency", timeRange)}
              />
              <MetricCard
                icon={<FaArrowDown className="w-5 h-5 text-blue-500" />}
                title="Download"
                value={history[0].downloadSpeed.toFixed(2)}
                unit="Mbps"
                average={calculateAverage(history, "downloadSpeed", timeRange)}
              />
              <MetricCard
                icon={<FaArrowUp className="w-5 h-5 text-emerald-500" />}
                title="Upload"
                value={history[0].uploadSpeed.toFixed(2)}
                unit="Mbps"
                average={calculateAverage(history, "uploadSpeed", timeRange)}
              />
              <MetricCard
                icon={<FaWaveSquare className="w-5 h-5 text-purple-400" />}
                title="Jitter"
                value={history[0].jitter?.toFixed(2) ?? "N/A"}
                unit="ms"
                average={calculateAverage(history, "jitter", timeRange)}
              />
            </div>
          </motion.div>
        )}

        {/* Speed History Chart */}
        {Boolean(historyData?.pages?.[0]?.data?.length) && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 20 }}
            transition={{ duration: 0.5 }}
          >
            <SpeedHistoryChart
              timeRange={timeRange}
              onTimeRangeChange={setTimeRange}
            />
          </motion.div>
        )}

        {/* Server Selection and Schedule Manager Container */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6 items-start">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 20 }}
            transition={{ duration: 0.5 }}
          >
            <ServerList
              servers={servers}
              selectedServers={selectedServers}
              onSelect={handleServerSelect}
              multiSelect={options.multiServer}
              onMultiSelectChange={(value: boolean) =>
                setOptions((prev) => ({ ...prev, multiServer: value }))
              }
              onRunTest={runTest}
              isLoading={isLoading}
              testType={testType}
              onTestTypeChange={setTestType}
            />
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 20 }}
            transition={{ duration: 0.5 }}
          >
            <ScheduleManager
              servers={allServers}
              selectedServers={selectedServers}
              onServerSelect={handleServerSelect}
            />
          </motion.div>
        </div>
      </Container>
    </div>
  );
}

// Helper Components
const MetricCard: React.FC<{
  icon: React.ReactNode;
  title: string;
  value: string;
  unit: string;
  average?: string;
}> = ({ icon, title, value, unit, average }) => (
  <div className="bg-gray-850/95 p-4 rounded-xl border border-gray-900 shadow-lg">
    <div className="flex items-center gap-3 mb-2">
      <div className="text-gray-400">{icon}</div>
      <h3 className="text-gray-300 font-medium">{title}</h3>
    </div>
    <div className="flex items-baseline gap-2">
      <span className="text-2xl font-bold text-white">{value}</span>
      <span className="text-gray-400">{unit}</span>
    </div>
    {average && (
      <div className="mt-1 text-sm text-gray-400">
        Average: {average} {unit}
      </div>
    )}
  </div>
);

const formatNextRun = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffMins = Math.round(diffMs / 60000);

  if (diffMins < 60) {
    return `in ${diffMins} minute${diffMins !== 1 ? "s" : ""}`;
  } else if (diffMins < 1440) {
    const hours = Math.floor(diffMins / 60);
    return `in ${hours} hour${hours !== 1 ? "s" : ""}`;
  } else {
    const days = Math.floor(diffMins / 1440);
    return `in ${days} day${days !== 1 ? "s" : ""}`;
  }
};

const calculateAverage = (
  history: SpeedTestResult[],
  field: keyof SpeedTestResult,
  timeRange: TimeRange
): string => {
  const now = new Date();
  const cutoff = new Date();

  // Set cutoff date based on timeRange
  switch (timeRange) {
    case "1d":
      cutoff.setDate(now.getDate() - 1);
      break;
    case "3d":
      cutoff.setDate(now.getDate() - 3);
      break;
    case "1w":
      cutoff.setDate(now.getDate() - 7);
      break;
    case "1m":
      cutoff.setMonth(now.getMonth() - 1);
      break;
    case "all":
      return calculateAllTimeAverage(history, field);
    default:
      cutoff.setDate(now.getDate() - 7); // Default to 1 week
  }

  const filteredHistory = history.filter(
    (item) => new Date(item.createdAt) >= cutoff
  );

  const validValues = filteredHistory
    .map((item) => {
      const value = item[field];
      if (typeof value === "string") {
        // Handle string values (like latency with "ms" suffix)
        return parseFloat(value.replace("ms", ""));
      }
      return Number(value);
    })
    .filter((value) => !isNaN(value));

  if (validValues.length === 0) return "N/A";

  const avg =
    validValues.reduce((sum, value) => sum + value, 0) / validValues.length;
  return avg.toFixed(2);
};

// Helper function for "all" time range
const calculateAllTimeAverage = (
  history: SpeedTestResult[],
  field: keyof SpeedTestResult
): string => {
  const validValues = history
    .map((item) => {
      const value = item[field];
      if (typeof value === "string") {
        // Handle string values (like latency with "ms" suffix)
        return parseFloat(value.replace("ms", ""));
      }
      return Number(value);
    })
    .filter((value) => !isNaN(value));

  if (validValues.length === 0) return "N/A";

  const avg =
    validValues.reduce((sum, value) => sum + value, 0) / validValues.length;
  return avg.toFixed(2);
};
