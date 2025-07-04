/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect, useMemo } from "react";
import { Container } from "@mui/material";
import {
  FaGithub,
  FaArrowDown,
} from "react-icons/fa";
import { IoIosPulse } from "react-icons/io";
import { XMarkIcon } from "@heroicons/react/20/solid";
import { ShareModal } from "./ShareModal";
import { TestProgress } from "./TestProgress";
import { TabNavigation } from "../common/TabNavigation";
import { DashboardTab } from "./DashboardTab";
import { SpeedTestTab } from "./SpeedTestTab";
import { TracerouteTab } from "./TracerouteTab";
import {
  ChartBarIcon,
  PlayIcon,
  GlobeAltIcon,
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
  getPublicHistory,
} from "@/api/speedtest";
import { motion, AnimatePresence } from "motion/react";

interface SpeedTestProps {
  isPublic?: boolean;
}

export default function SpeedTest({ isPublic = false }: SpeedTestProps) {
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
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(() => {
    const saved = localStorage.getItem("netronome-active-tab");
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
  ];

  const handleTabChange = (tabId: string) => {
    setActiveTab(tabId);
    localStorage.setItem("netronome-active-tab", tabId);
  };

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
      console.log("[SpeedTest] Fetching schedules...");
      return getSchedules();
    },
    refetchInterval: 30000, // Refetch every 30 seconds (same as ScheduleManager)
    staleTime: 10000, // Consider data stale after 10 seconds
  }) as { data: Schedule[] };

  // Log when schedules data changes
  useEffect(() => {
    console.log("[SpeedTest] Schedules data updated:", schedules);
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
              className="relative bg-red-950/80 backdrop-blur-sm border border-red-800/50 rounded-xl mb-4 shadow-lg overflow-hidden"
            >
              <div className="absolute inset-0 bg-gradient-to-r from-red-900/20 to-red-800/20 pointer-events-none" />
              <div className="relative flex items-start justify-between p-4">
                <div className="flex items-start gap-3 flex-1">
                  <div className="flex-shrink-0 mt-0.5">
                    <svg
                      className="w-5 h-5 text-red-400"
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
                    <h3 className="text-sm font-medium text-red-200 mb-1">
                      Error
                    </h3>
                    <div className="text-sm text-red-300/90 whitespace-pre-wrap break-words">
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
                  className="flex-shrink-0 ml-4 text-red-400 hover:text-red-300 transition-colors duration-200"
                  aria-label="Dismiss error"
                >
                  <XMarkIcon className="w-5 h-5" />
                </button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* No History Message - Only show when no tests exist at all */}
        {!isHistoryLoading && !hasAnyTests && (
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

        {/* Tab Navigation */}
        {!isPublic && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="mb-6"
          >
            <TabNavigation
              tabs={tabs}
              activeTab={activeTab}
              onTabChange={handleTabChange}
            />
          </motion.div>
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
        </AnimatePresence>


      </Container>

      {/* Public Footer */}
      {isPublic && (
        <div className="border-t border-gray-800/50 py-4 mt-8">
          <Container maxWidth="xl">
            <div className="flex justify-center">
              <div className="text-gray-500 text-sm">
                Powered by{" "}
                <a
                  href="https://netrono.me"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-300 hover:text-white transition-colors duration-200 underline decoration-gray-600 hover:decoration-gray-400"
                >
                  Netronome
                </a>
                {" • "}
                <a
                  href="https://github.com/autobrr/netronome"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-400 hover:text-gray-300 transition-colors duration-200 inline-flex items-center gap-1"
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

