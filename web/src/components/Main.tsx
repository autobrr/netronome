/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect, useMemo } from "react";
import { Container } from "@mui/material";
import { FaGithub } from "react-icons/fa";
import { XMarkIcon } from "@heroicons/react/20/solid";
import { ShareModal } from "./speedtest/ShareModal";
import { TestProgress } from "./speedtest/TestProgress";
import { TabNavigation } from "./common/TabNavigation";
import { DashboardTab } from "./speedtest/DashboardTab";
import { SpeedTestTab } from "./speedtest/SpeedTestTab";
import { TracerouteTab } from "./speedtest/TracerouteTab";
import { MonitorTab } from "./monitor/MonitorTab";
import { showToast } from "@/components/common/Toast";
import {
  ChartBarIcon,
  PlayIcon,
  GlobeAltIcon,
  ServerIcon,
} from "@heroicons/react/24/outline";
import {
  Server,
  SpeedTestResult,
  TestProgress as TestProgressType,
  TimeRange,
  TestOptions,
  PaginatedResponse,
  Schedule,
} from "@/types/types";
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
  getPublicHistory,
} from "@/api/speedtest";
import { motion, AnimatePresence } from "motion/react";

interface MainProps {
  isPublic?: boolean;
}

export default function Main({ isPublic = false }: MainProps) {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [options, setOptions] = useState<TestOptions>({
    enableDownload: true,
    enableUpload: true,
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
  const [isLoading, setIsLoading] = useState(false);
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(() => {
    const saved = localStorage.getItem("netronome-active-tab");
    // Migrate old 'packetloss' tab to 'dashboard' since it no longer exists
    if (saved === "packetloss") {
      localStorage.setItem("netronome-active-tab", "dashboard");
      return "dashboard";
    }
    return saved || "dashboard";
  });

  // Tab configuration
  const tabs = [
    {
      id: "dashboard",
      label: "Dashboard",
      icon: <ChartBarIcon className="w-5 h-5" />,
    },
    {
      id: "speedtest",
      label: "Speed Test",
      icon: <PlayIcon className="w-5 h-5" />,
    },
    {
      id: "traceroute",
      label: "Traceroute",
      icon: <GlobeAltIcon className="w-5 h-5" />,
    },
    {
      id: "monitor",
      label: "Agents",
      icon: <ServerIcon className="w-5 h-5" />,
    },
  ];

  const handleTabChange = (tabId: string) => {
    setActiveTab(tabId);
    localStorage.setItem("netronome-active-tab", tabId);
  };

  // Queries
  const { data: speedtestServers = [] } = useQuery({
    queryKey: ["servers", "speedtest"],
    queryFn: () => getServers("speedtest"),
    enabled: !isPublic,
  }) as { data: Server[] };

  const { data: librespeedServers = [] } = useQuery({
    queryKey: ["servers", "librespeed"],
    queryFn: () => getServers("librespeed"),
    enabled: !isPublic,
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

  const { data: historyData } = useInfiniteQuery({
    queryKey: ["history", timeRange, isPublic],
    queryFn: async ({ pageParam = 1 }) => {
      const historyFn = isPublic ? getPublicHistory : getHistory;
      const response = await historyFn(timeRange, pageParam, 20);
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

  // Query to get all-time history for latest run display and existence check
  const { data: allTimeHistoryData } = useInfiniteQuery({
    queryKey: ["history", "all", isPublic],
    queryFn: async ({ pageParam = 1 }) => {
      const historyFn = isPublic ? getPublicHistory : getHistory;
      const response = await historyFn("all", pageParam, 20); // Get more results for latest run display
      return response as PaginatedResponse<SpeedTestResult>;
    },
    getNextPageParam: () => undefined, // Only fetch first page
    initialPageParam: 1,
    staleTime: 60000, // Cache for 1 minute
  });

  const allTimeHistory = useMemo(() => {
    if (!allTimeHistoryData?.pages) return [];
    return allTimeHistoryData.pages.flatMap(
      (page) => page?.data ?? []
    ) as SpeedTestResult[];
  }, [allTimeHistoryData]);

  const hasAnyTests = useMemo(() => {
    return allTimeHistory.length > 0;
  }, [allTimeHistory]);

  // Use current time range history if available, otherwise fall back to all-time history for latest run
  const latestTest = useMemo(() => {
    return history && history.length > 0
      ? history[0]
      : allTimeHistory.length > 0
      ? allTimeHistory[0]
      : null;
  }, [history, allTimeHistory]);

  const { data: schedules = [] } = useQuery({
    queryKey: ["schedules"],
    queryFn: () => {
      console.log("[Main] Fetching schedules...");
      return getSchedules();
    },
    refetchInterval: 30000, // Refetch every 30 seconds (same as ScheduleManager)
    staleTime: 10000, // Consider data stale after 10 seconds
    enabled: !isPublic,
  }) as { data: Schedule[] };

  // Log when schedules data changes
  useEffect(() => {
    console.log("[Main] Schedules data updated:", schedules);
  }, [schedules]);

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
      showToast("Please select a server", "warning");
      return;
    }

    setError(null);
    setTestStatus("running");
    setIsLoading(true);

    // Initialize progress with correct type flags
    if (testType === "iperf") {
      setProgress({
        currentServer: selectedServers[0].name,
        currentTest: "download",
        currentSpeed: 0,
        isComplete: false,
        type: "download",
        speed: 0,
        latency: 0,
        isScheduled: false,
        progress: 0,
        isIperf: true,
        isLibrespeed: false,
      });
    } else if (testType === "librespeed") {
      setProgress({
        currentServer: selectedServers[0].name,
        currentTest: "running",
        currentSpeed: 0,
        isComplete: false,
        type: "running",
        speed: 0,
        latency: 0,
        isScheduled: false,
        progress: 0,
        isIperf: false,
        isLibrespeed: true,
      });
    }

    try {
      // Show toast when test starts
      const testTypeName =
        testType === "iperf"
          ? "iperf3"
          : testType === "librespeed"
          ? "Librespeed"
          : "Speedtest";
      showToast(`${testTypeName} test started`, "success", {
        description: `Testing ${selectedServers[0].host}`,
      });

      await speedTestMutation.mutateAsync({
        ...options,
        useIperf: testType === "iperf",
        useLibrespeed: testType === "librespeed",
        serverIds:
          testType === "speedtest" || testType === "librespeed"
            ? selectedServers.map((s) => s.id)
            : [],
        serverHost: testType === "iperf" ? selectedServers[0].host : undefined,
        serverName: testType === "iperf" ? selectedServers[0].name : undefined,
      });
    } catch (error) {
      console.error("Error running test:", error);
      const errorMessage =
        error instanceof Error ? error.message : "An unknown error occurred";
      setError(errorMessage);
      setTestStatus("idle");
      showToast("Failed to start speed test", "error", {
        description: errorMessage,
      });
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
            // Debug log for test updates
            if (
              update.testType === "iperf3" ||
              update.testType === "speedtest"
            ) {
              console.log(`${update.testType} update:`, {
                type: update.type,
                speed: update.speed,
                progress: update.progress,
                isComplete: update.isComplete,
                testType: update.testType,
                serverName: update.serverName,
                timestamp: new Date().toISOString(),
              });
            }

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
                // Preserve isIperf/isLibrespeed state if testType is not provided
                isIperf: update.testType
                  ? update.testType === "iperf3"
                  : prev?.isIperf || false,
                isLibrespeed: update.testType
                  ? update.testType === "librespeed"
                  : prev?.isLibrespeed || false,
              };

              if (!prev) {
                return baseProgress;
              }

              // For iperf3 and speedtest.net, always update to show live progress
              // The backend already throttles updates to once per second

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
              // Show success toast with test results
              if (update.download && update.upload) {
                showToast("Speed test completed", "success", {
                  description: `Download: ${update.download.toFixed(
                    2
                  )} Mbps, Upload: ${update.upload.toFixed(2)} Mbps`,
                });
              }
            }
          }
        } catch (error) {
          console.error("Test status check error:", error);
        }
      }, 2000); // Poll every 2 seconds to match both speedtest.net and iperf3

      return () => clearInterval(pollInterval);
    }
  }, [testStatus, queryClient, testType]);

  return (
    <div className="min-h-screen flex flex-col">
      <Container maxWidth="xl" className="pb-20 sm:pb-8 pt-16 sm:pt-20 md:pt-14 flex-1">
        {/* Test Progress - Always rendered with fixed height to prevent layout shift */}
        <div className="flex justify-center mb-2 sm:mb-4 mt-2 sm:mt-4 md:mt-0 h-5">
          <TestProgress progress={progress} />
        </div>

        {/* Header - Now just an empty spacer */}
        <div className="mb-4 sm:mb-8" />

        {/* Share Modal */}
        <ShareModal
          isOpen={shareModalOpen}
          onClose={() => setShareModalOpen(false)}
        />

        {/* Error Messages */}
        <AnimatePresence>
          {error && (
            <motion.div
              key="error-message"
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
              className="relative bg-red-100/50 dark:bg-red-950/10 backdrop-blur-sm border border-red-400/50 dark:border-red-800/50 rounded-xl mb-4 shadow-lg overflow-hidden"
            >
              <div className="absolute inset-0 bg-gradient-to-r from-red-200/20 to-red-300/20 dark:from-red-900/20 dark:to-red-800/20 pointer-events-none" />
              <div className="relative flex items-start justify-between p-4">
                <div className="flex items-start gap-3 flex-1">
                  <div className="flex-shrink-0 mt-0.5">
                    <svg
                      className="w-5 h-5 text-red-600 dark:text-red-400"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                  </div>
                  <div className="flex-1">
                    <h3 className="text-sm font-bold text-red-800 dark:text-red-300 mb-1">
                      Error
                    </h3>
                    <div className="text-sm text-red-700/90 dark:text-red-300/90 break-words whitespace-pre-wrap">
                      {(() => {
                        try {
                          // Try to extract and format JSON from error messages
                          const jsonMatch = error.match(/(\{[\s\S]*\})/);
                          if (jsonMatch) {
                            const jsonStr = jsonMatch[1];
                            const parsed = JSON.parse(jsonStr);
                            const formatted = JSON.stringify(parsed, null, 2);
                            return error.replace(jsonMatch[1], formatted);
                          }
                          return error;
                        } catch {
                          return error;
                        }
                      })()}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => setError(null)}
                  className="flex-shrink-0 ml-4 text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 transition-colors duration-200"
                  aria-label="Dismiss error"
                >
                  <XMarkIcon className="w-5 h-5" />
                </button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Tab Navigation */}
        {!isPublic && (
          <div className="tab-navigation-container mb-4 sm:mb-6">
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5 }}
            >
              <TabNavigation
                tabs={tabs}
                activeTab={activeTab}
                onTabChange={handleTabChange}
              />
            </motion.div>
          </div>
        )}

        {/* Tab Content */}
        <AnimatePresence mode="wait">
          {(activeTab === "dashboard" || isPublic) && (
            <motion.div
              key="dashboard"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
              <DashboardTab
                latestTest={latestTest}
                tests={history}
                timeRange={timeRange}
                onTimeRangeChange={setTimeRange}
                isPublic={isPublic}
                hasAnyTests={hasAnyTests}
                onShareClick={() => setShareModalOpen(true)}
                onNavigateToSpeedTest={() => handleTabChange("speedtest")}
                onNavigateToVnstat={(agentId?: number) => {
                  handleTabChange("monitor");
                  if (agentId) {
                    sessionStorage.setItem(
                      "netronome-preselect-agent",
                      agentId.toString()
                    );
                  }
                }}
              />
            </motion.div>
          )}

          {!isPublic && activeTab === "speedtest" && (
            <motion.div
              key="speedtest"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
              <SpeedTestTab
                servers={servers}
                selectedServers={selectedServers}
                onServerSelect={handleServerSelect}
                options={options}
                onOptionsChange={setOptions}
                testType={testType}
                onTestTypeChange={setTestType}
                isLoading={isLoading}
                onRunTest={runTest}
                progress={progress}
                allServers={allServers}
              />
            </motion.div>
          )}

          {!isPublic && activeTab === "traceroute" && (
            <motion.div
              key="traceroute"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
              <TracerouteTab />
            </motion.div>
          )}

          {!isPublic && activeTab === "monitor" && (
            <motion.div
              key="monitor"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
              <MonitorTab />
            </motion.div>
          )}
        </AnimatePresence>
      </Container>


      {/* Public Footer */}
      {isPublic && (
        <div className="mb-8">
          <Container maxWidth="xl">
            <div className="flex justify-center">
              <div className="text-gray-600 dark:text-gray-500 text-sm">
                Powered by{" "}
                <a
                  href="https://netrono.me"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors duration-200 underline decoration-gray-500 dark:decoration-gray-600 hover:decoration-gray-700 dark:hover:decoration-gray-400"
                >
                  Netronome
                </a>
                {" • "}
                <a
                  href="https://github.com/autobrr/netronome"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 transition-colors duration-200 inline-flex items-center gap-1"
                >
                  <span className="underline decoration-gray-600 hover:decoration-gray-400">
                    Source
                  </span>
                  <FaGithub className="ml-1 w-3.5 h-3.5" />
                </a>
              </div>
            </div>
          </Container>
        </div>
      )}
    </div>
  );
}
