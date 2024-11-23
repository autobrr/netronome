/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, useMemo } from "react";
import { Container } from "@mui/material";
import { IoIosPulse, IoMdGitCompare } from "react-icons/io";
import { FaArrowDown, FaArrowUp } from "react-icons/fa";
import { ServerList } from "./speedtest/ServerList";
import { TestProgress } from "./speedtest/TestProgress";
import { SpeedHistoryChart } from "./speedtest/SpeedHistoryChart";
import ScheduleManager from "./ScheduleManager";
import {
  Server,
  SpeedTestResult,
  TestProgress as TestProgressType,
  TimeRange,
  TestOptions,
  PaginatedResponse,
} from "../types/types";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";
import logo from "../assets/logo.png";
import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import {
  fetchServers,
  fetchHistory,
  fetchSchedules,
  runSpeedTest,
  fetchTestStatus,
} from "../api/speedtest";

export default function SpeedTest() {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [options, setOptions] = useState<TestOptions>({
    enableDownload: true,
    enableUpload: true,
    enablePacketLoss: true,
    multiServer: false,
  });
  const [selectedServers, setSelectedServers] = useState<Server[]>([]);
  const [progress, setProgress] = useState<TestProgressType | null>(null);
  const [testStatus, setTestStatus] = useState<"idle" | "running" | "complete">(
    "idle"
  );
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    const saved = localStorage.getItem("speedtest-time-range");
    return (saved as TimeRange) || "1w";
  });
  const [scheduledTestRunning, setScheduledTestRunning] = useState(false);

  // Queries
  const { data: servers = [] } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
  });

  const { data: historyData, isLoading: isHistoryLoading } = useInfiniteQuery({
    queryKey: ["history", timeRange],
    queryFn: async ({ pageParam = 1 }) => {
      const response = await fetchHistory(timeRange, pageParam);
      return response as PaginatedResponse<SpeedTestResult>;
    },
    getNextPageParam: (lastPage: PaginatedResponse<SpeedTestResult>) => {
      if (lastPage.data.length < lastPage.limit) return undefined;
      return lastPage.page + 1;
    },
    initialPageParam: 1,
    staleTime: 0,
    placeholderData: (previousData) => previousData,
  });

  const history = useMemo(() => {
    if (!historyData) return [];
    return historyData.pages.flatMap((page) => page.data);
  }, [historyData]);

  const { data: schedules = [] } = useQuery({
    queryKey: ["schedules"],
    queryFn: fetchSchedules,
  });

  // Mutations
  const speedTestMutation = useMutation({
    mutationFn: runSpeedTest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["history"] });
      setProgress(null);
      setTestStatus("complete");
    },
    onError: (error: Error) => {
      setError(error.message);
      setTestStatus("idle");
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

    speedTestMutation.mutate({
      ...options,
      serverIds: selectedServers.map((s) => s.id),
    });
  };

  useEffect(() => {
    if (testStatus !== "running" && !scheduledTestRunning) return;

    let pollCount = 0;
    const maxPolls = 180;
    let lastUpdate = Date.now();

    const pollInterval = setInterval(async () => {
      try {
        pollCount++;
        if (pollCount > maxPolls) {
          clearInterval(pollInterval);
          setError("Test timed out after 3 minutes");
          setTestStatus("idle");
          return;
        }

        const now = Date.now();
        if (now - lastUpdate < 2000) {
          return;
        }
        lastUpdate = now;

        const update = await fetchTestStatus();

        if (update.isComplete || update.type === "complete") {
          clearInterval(pollInterval);
          setTestStatus("complete");
          setProgress(null);
          queryClient.invalidateQueries({ queryKey: ["history"] });
        } else if (update.speed > 0) {
          setTestStatus("running");
          setProgress({
            currentServer: update.serverName,
            currentTest: update.type,
            currentSpeed: update.speed,
            progress: update.progress,
            isComplete: update.isComplete,
            type: update.type,
            speed: update.speed,
            latency: update.latency,
            isScheduled: update.isScheduled,
          });
        }
      } catch (error) {
        console.error("Status check error:", error);
      }
    }, 2000);

    return () => clearInterval(pollInterval);
  }, [testStatus, scheduledTestRunning, queryClient]);

  useEffect(() => {
    const pollScheduledTests = setInterval(async () => {
      try {
        const update = await fetchTestStatus();

        if (update.isScheduled) {
          setScheduledTestRunning(true);
          setTestStatus("running");
          setProgress({
            currentServer: update.serverName,
            currentTest: update.type,
            currentSpeed: update.speed,
            progress: update.progress,
            isComplete: update.isComplete,
            type: update.type,
            speed: update.speed,
            latency: update.latency,
            isScheduled: update.isScheduled,
          });

          if (update.isComplete) {
            setScheduledTestRunning(false);
            setTestStatus("complete");
            setProgress(null);
            await Promise.all([
              queryClient.invalidateQueries({ queryKey: ["history"] }),
              queryClient.invalidateQueries({ queryKey: ["schedules"] }),
            ]);
          }
        }
      } catch (error) {
        console.error("Scheduled test status check error:", error);
      }
    }, 1000);

    return () => clearInterval(pollScheduledTests);
  }, [queryClient]);

  return (
    <div className="min-h-screen">
      <Container maxWidth="lg" className="pb-8">
        {/* Header */}
        <div className="flex justify-between items-center -ml-2 mb-8 pt-8">
          <div className="flex items-center gap-3">
            <img
              src={logo}
              alt="netronome Logo"
              className="h-12 w-12 select-none pointer-events-none"
              draggable="false"
            />
            <h1 className="text-3xl font-bold text-white select-none">
              Netronome
            </h1>
          </div>
        </div>

        {/* Test Progress */}
        {(testStatus === "running" || scheduledTestRunning) && progress && (
          <TestProgress progress={progress} />
        )}

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
        {history && history.length > 0 && (
          <div className="mb-6">
            <h2 className="text-white text-xl font-semibold">Latest Run</h2>
            <div className="flex justify-between items-center text-gray-400 text-sm mb-4">
              <div>
                Last test run:{" "}
                {new Date(history[0].createdAt).toLocaleString(undefined, {
                  dateStyle: "short",
                  timeStyle: "short",
                })}
              </div>
              {schedules && schedules.length > 0 && (
                <div>
                  Next scheduled run:{" "}
                  <span className="text-blue-400">
                    {formatNextRun(schedules[0].nextRun)}
                  </span>
                </div>
              )}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 cursor-default">
              <MetricCard
                icon={<IoIosPulse className="w-5 h-5 text-blue-400" />}
                title="Latency"
                value={parseFloat(history[0].latency).toFixed(2)}
                unit="ms"
                average={calculateAverage(history, "latency", timeRange)}
              />
              <MetricCard
                icon={<FaArrowDown className="w-5 h-5 text-emerald-400" />}
                title="Download"
                value={history[0].downloadSpeed.toFixed(2)}
                unit="Mbps"
                average={calculateAverage(history, "downloadSpeed", timeRange)}
              />
              <MetricCard
                icon={<FaArrowUp className="w-5 h-5 text-purple-400" />}
                title="Upload"
                value={history[0].uploadSpeed.toFixed(2)}
                unit="Mbps"
                average={calculateAverage(history, "uploadSpeed", timeRange)}
              />
              <MetricCard
                icon={<IoMdGitCompare className="w-5 h-5 text-blue-400" />}
                title="Jitter"
                value={history[0].jitter?.toFixed(2) ?? "N/A"}
                unit="ms"
                average={calculateAverage(history, "jitter", timeRange)}
              />
            </div>
          </div>
        )}

        {/* Speed History Chart */}
        {historyData &&
          historyData.pages[0] &&
          historyData.pages[0].data.length > 0 && (
            <SpeedHistoryChart
              timeRange={timeRange}
              onTimeRangeChange={setTimeRange}
            />
          )}

        {/* Server Selection */}
        <Disclosure defaultOpen={false}>
          {({ open }) => (
            <div className="flex flex-col mb-6">
              <DisclosureButton
                className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
                  open ? "rounded-t-xl border-b-0" : "rounded-xl"
                } shadow-lg border-b-0 border-gray-900 text-left`}
              >
                <div className="flex flex-col">
                  <h2 className="text-white text-xl font-semibold p-1 select-none">
                    Server Selection
                  </h2>
                </div>
                <div className="flex items-center gap-2">
                  {selectedServers.length > 0 && (
                    <span className="text-gray-400">
                      {selectedServers.length} server
                      {selectedServers.length !== 1 ? "s" : ""} selected
                    </span>
                  )}
                  <ChevronDownIcon
                    className={`${
                      open ? "transform rotate-180" : ""
                    } w-5 h-5 text-gray-400 transition-transform duration-200`}
                  />
                </div>
              </DisclosureButton>

              {open && (
                <div className="bg-gray-850/95 px-4 rounded-b-xl shadow-lg">
                  <div className="flex flex-col">
                    <p className="text-gray-400 text-sm pl-1 select-none pointer-events-none">
                      Select one or more servers to test
                    </p>
                  </div>
                  <ServerList
                    servers={servers}
                    selectedServers={selectedServers}
                    onSelect={handleServerSelect}
                    multiSelect={options.multiServer}
                    onMultiSelectChange={(enabled) =>
                      setOptions((prev) => ({ ...prev, multiServer: enabled }))
                    }
                    onRunTest={runTest}
                    isLoading={speedTestMutation.isPending}
                  />
                </div>
              )}
            </div>
          )}
        </Disclosure>

        {/* Schedule Manager */}
        <ScheduleManager
          servers={servers}
          selectedServers={selectedServers}
          onServerSelect={handleServerSelect}
          loading={speedTestMutation.isPending}
        />
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
  <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900">
    <div className="flex items-center gap-3 mb-4">
      {icon}
      <h3 className="text-gray-400 font-medium">{title}</h3>
    </div>
    <div className="flex flex-col">
      <div className="text-white text-3xl font-bold">
        {value}
        <span className="text-xl font-normal text-gray-400 ml-1">{unit}</span>
      </div>
      {average && (
        <div className="text-sm text-gray-400 mt-1">
          avg: {average} {unit}
        </div>
      )}
    </div>
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
