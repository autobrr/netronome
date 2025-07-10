/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, Fragment, useMemo } from "react";
import { motion } from "motion/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  Transition,
  TransitionChild,
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
} from "@headlessui/react";
import { XMarkIcon, ChevronUpDownIcon } from "@heroicons/react/24/solid";
import {
  ChartBarIcon,
  TrashIcon,
  PlayIcon,
  StopIcon,
  PencilIcon,
  SignalIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  GlobeAltIcon,
  WifiIcon,
  ArrowTrendingUpIcon,
  ArrowTrendingDownIcon,
} from "@heroicons/react/24/outline";
import { MetricCard } from "@/components/common/MetricCard";
import { PacketLossMonitor } from "@/types/types";
import { CrossTabNavigationHook } from "@/hooks/useCrossTabNavigation";
import { Button } from "@/components/ui/Button";
import {
  getPacketLossMonitors,
  createPacketLossMonitor,
  updatePacketLossMonitor,
  deletePacketLossMonitor,
  startPacketLossMonitor,
  stopPacketLossMonitor,
  getPacketLossMonitorStatus,
  getPacketLossHistory,
} from "@/api/packetloss";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface MonitorFormData {
  host: string;
  name: string;
  interval: number;
  packetCount: number;
  threshold: number;
  enabled: boolean;
}

interface IntervalOption {
  value: number;
  label: string;
}

const intervalOptions: IntervalOption[] = [
  { value: 10, label: "Every 10 seconds" },
  { value: 30, label: "Every 30 seconds" },
  { value: 60, label: "Every 1 minute" },
  { value: 300, label: "Every 5 minutes" },
  { value: 900, label: "Every 15 minutes" },
  { value: 1800, label: "Every 30 minutes" },
  { value: 3600, label: "Every 1 hour" },
  { value: 21600, label: "Every 6 hours" },
  { value: 43200, label: "Every 12 hours" },
  { value: 86400, label: "Every 24 hours" },
];

const formatInterval = (seconds: number): string => {
  if (seconds < 60) {
    return `${seconds} second${seconds !== 1 ? "s" : ""}`;
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes} minute${minutes !== 1 ? "s" : ""}`;
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    return `${hours} hour${hours !== 1 ? "s" : ""}`;
  } else {
    const days = Math.floor(seconds / 86400);
    return `${days} day${days !== 1 ? "s" : ""}`;
  }
};

const formatRTT = (rtt: number) => {
  if (rtt === 0) return "0";
  return rtt.toFixed(1);
};

interface PacketLossTabProps {
  crossTabNavigation?: CrossTabNavigationHook;
}

export const PacketLossTab: React.FC<PacketLossTabProps> = ({
  crossTabNavigation,
}) => {
  const queryClient = useQueryClient();
  const [selectedMonitor, setSelectedMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingMonitor, setEditingMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [formData, setFormData] = useState<MonitorFormData>({
    host: "",
    name: "",
    interval: 60,
    packetCount: 10,
    threshold: 5.0,
    enabled: true,
  });
  const [monitorStatuses, setMonitorStatuses] = useState<Map<number, any>>(
    new Map()
  );

  // Fetch monitors
  const { data: monitors, isLoading } = useQuery({
    queryKey: ["packetloss", "monitors"],
    queryFn: getPacketLossMonitors,
    staleTime: 30000, // Consider data stale after 30 seconds
  });

  // Ensure monitors is always an array
  const monitorList = monitors || [];

  // Fetch history for selected monitor - similar to TracerouteTab's results query
  const { data: monitorHistory } = useQuery({
    queryKey: ["packetloss", "history", selectedMonitor?.id],
    queryFn: () => getPacketLossHistory(selectedMonitor!.id, 100),
    enabled: !!selectedMonitor,
    staleTime: 5000, // Consider data stale after 5 seconds
    refetchInterval: false, // Don't refetch automatically
  });

  // Ensure monitorHistory is always an array
  const historyList = monitorHistory || [];

  // Fetch fresh history when monitor is selected
  useEffect(() => {
    if (selectedMonitor) {
      // Fetch fresh data and update cache directly
      getPacketLossHistory(selectedMonitor.id, 100)
        .then((freshHistory) => {
          queryClient.setQueryData(
            ["packetloss", "history", selectedMonitor.id],
            freshHistory
          );
        })
        .catch((error) => {
          console.error("Failed to fetch monitor history:", error);
        });
    }
  }, [selectedMonitor, queryClient]);

  // Poll for status updates for enabled monitors
  useEffect(() => {
    const enabledMonitors = monitorList.filter((m) => m.enabled);

    if (enabledMonitors.length === 0) {
      return;
    }

    const pollInterval = setInterval(async () => {
      const statusPromises = enabledMonitors.map(async (monitor) => {
        try {
          const status = await getPacketLossMonitorStatus(monitor.id);
          return { monitorId: monitor.id, status };
        } catch (error) {
          console.error(
            `Failed to get status for monitor ${monitor.id}:`,
            error
          );
          return null;
        }
      });

      const results = await Promise.all(statusPromises);

      // Check for completed tests - backend now properly sets isComplete
      const completedTests = results.filter(
        (result) => result && result.status.isComplete
      );

      // Update state first
      setMonitorStatuses((prev) => {
        const newStatuses = new Map(prev);
        results.forEach((result) => {
          if (result) {
            newStatuses.set(result.monitorId, result.status);
          }
        });
        return newStatuses;
      });

      // Then handle refetches for completed tests
      for (const completedTest of completedTests) {
        if (!completedTest) continue;

        // Refetch monitors list
        await queryClient.refetchQueries({
          queryKey: ["packetloss", "monitors"],
        });

        // If this is the selected monitor, fetch fresh data and update cache directly
        if (completedTest.monitorId === selectedMonitor?.id) {
          try {
            // Fetch fresh history from the backend
            const freshHistory = await getPacketLossHistory(
              completedTest.monitorId,
              100
            );

            // Directly update the cache with fresh data (like TracerouteTab does)
            queryClient.setQueryData(
              ["packetloss", "history", completedTest.monitorId],
              freshHistory
            );

            // Force React Query to notify all subscribers of the change
            queryClient.invalidateQueries({
              queryKey: ["packetloss", "history", completedTest.monitorId],
              exact: true,
            });
          } catch (error) {
            console.error("Failed to fetch updated history:", error);
          }
        } else {
          // For other monitors, just invalidate to mark as stale
          await queryClient.invalidateQueries({
            queryKey: ["packetloss", "history", completedTest.monitorId],
          });
        }

        // Clear completion status after delay
        setTimeout(() => {
          setMonitorStatuses((prev) => {
            const newMap = new Map(prev);
            const currentStatus = newMap.get(completedTest.monitorId);
            if (currentStatus && currentStatus.isComplete) {
              newMap.delete(completedTest.monitorId);
            }
            return newMap;
          });
        }, 3000);
      }
    }, 2000); // Poll every 2 seconds instead of 1 second

    return () => clearInterval(pollInterval);
  }, [monitorList, queryClient, selectedMonitor]);

  // Handle cross-tab data consumption
  useEffect(() => {
    if (crossTabNavigation) {
      const crossTabData = crossTabNavigation.consumePacketLossData();
      if (crossTabData?.host) {
        setFormData((prev) => ({
          ...prev,
          host: crossTabData.host || "",
          name: crossTabData.name || `Monitor for ${crossTabData.host}`,
          enabled: true,
        }));
        if (crossTabData.fromTraceroute) {
          setShowForm(true);
        }
      }
    }
  }, [crossTabNavigation]);

  const createMutation = useMutation({
    mutationFn: createPacketLossMonitor,
    onSuccess: async () => {
      // Invalidate first to ensure data is marked stale
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });

      // Then refetch
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      handleCancelForm(); // This will close the modal and reset form
    },
  });

  const updateMutation = useMutation({
    mutationFn: updatePacketLossMonitor,
    onSuccess: async () => {
      // Invalidate first to ensure data is marked stale
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });

      // Then refetch
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      handleCancelForm(); // This will close the modal and reset form
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deletePacketLossMonitor,
    onSuccess: async () => {
      // Invalidate first to ensure data is marked stale
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });

      // Then refetch
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });

      if (selectedMonitor?.id === deleteMutation.variables) {
        setSelectedMonitor(null);
      }
    },
  });

  const startMutation = useMutation({
    mutationFn: startPacketLossMonitor,
    onSuccess: async (_, monitorId) => {
      // Invalidate first to ensure data is marked stale
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      queryClient.invalidateQueries({
        queryKey: ["packetloss", "history", monitorId],
      });

      // Then refresh monitors and history to get updated state
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
    },
  });

  const stopMutation = useMutation({
    mutationFn: stopPacketLossMonitor,
    onSuccess: async (_, monitorId) => {
      setMonitorStatuses((prev) => {
        const newMap = new Map(prev);
        newMap.delete(monitorId);
        return newMap;
      });

      // Invalidate first to ensure data is marked stale
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      queryClient.invalidateQueries({
        queryKey: ["packetloss", "history", monitorId],
      });

      // Then refresh monitors and history to get updated state
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editingMonitor) {
      updateMutation.mutate({
        ...editingMonitor,
        ...formData,
      });
    } else {
      createMutation.mutate(formData);
    }
  };

  const handleEdit = (monitor: PacketLossMonitor) => {
    setEditingMonitor(monitor);
    setFormData({
      host: monitor.host,
      name: monitor.name || "",
      interval: monitor.interval,
      packetCount: monitor.packetCount,
      threshold: monitor.threshold,
      enabled: monitor.enabled,
    });
    setShowForm(true);
  };

  const handleCancelForm = () => {
    setShowForm(false);
    setEditingMonitor(null);
    setFormData({
      host: "",
      name: "",
      interval: 60,
      packetCount: 10,
      threshold: 5.0,
      enabled: true,
    });
  };

  const handleTraceRoute = () => {
    if (!selectedMonitor || !crossTabNavigation) return;

    crossTabNavigation.navigateToTraceroute({
      host: selectedMonitor.host,
      fromPacketLoss: true,
    });
  };

  // Prepare chart data - use useMemo to ensure it updates when historyList changes
  const chartData = useMemo(() => {
    // historyList is in descending order (newest first), so take first 30 and reverse
    const data = historyList
      .slice(0, 30) // First 30 results (most recent)
      .reverse() // Reverse to show oldest to newest for chart
      .map((result, index, array) => {
        const date = new Date(result.createdAt);
        const now = new Date();
        const isToday = date.toDateString() === now.toDateString();
        const hours = date.getHours().toString().padStart(2, "0");
        const minutes = date.getMinutes().toString().padStart(2, "0");

        // Show date for first item, last item, and when date changes
        let timeLabel = `${hours}:${minutes}`;
        if (!isToday || index === 0 || index === array.length - 1) {
          const month = (date.getMonth() + 1).toString().padStart(2, "0");
          const day = date.getDate().toString().padStart(2, "0");
          timeLabel = `${month}/${day} ${hours}:${minutes}`;
        }

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

  return (
    <div className="flex flex-col md:flex-row gap-6 md:items-start">
      {/* Left Column - Monitor List */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="flex-1"
      >
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Packet Loss Monitors
              </h2>
              <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                Direct ping monitoring to measure packet loss and latency over
                time
              </p>
              <p className="text-gray-500 dark:text-gray-500 text-xs mt-1 max-w-md">
                Note: This uses direct ICMP ping to measure packet loss - it
                does NOT trace network paths or show intermediate hops (use
                Traceroute tab for that)
              </p>
            </div>
            <Button
              onClick={() => setShowForm(true)}
              className="bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
            >
              Add Monitor
            </Button>
          </div>

          {/* Monitor List */}
          {isLoading ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400 mx-auto"></div>
              <p className="text-gray-600 dark:text-gray-400 mt-4">
                Loading monitors...
              </p>
            </div>
          ) : monitorList.length === 0 ? (
            <div className="text-center py-8">
              <SignalIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600 dark:text-gray-400">
                No monitors configured yet
              </p>
              <p className="text-gray-500 dark:text-gray-500 text-sm mt-2">
                Add a monitor to start tracking packet loss
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {monitorList.map((monitor) => (
                <motion.div
                  key={monitor.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.3 }}
                  className={`p-4 rounded-lg border transition-all cursor-pointer ${
                    selectedMonitor?.id === monitor.id
                      ? "bg-blue-500/10 border-blue-400/50 shadow-lg"
                      : "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 hover:shadow-md"
                  }`}
                  onClick={() =>
                    setSelectedMonitor(
                      selectedMonitor?.id === monitor.id ? null : monitor
                    )
                  }
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-gray-900 dark:text-white font-semibold truncate">
                          {monitor.name || monitor.host}
                        </h3>
                        {/* Status Badge */}
                        <div className="flex items-center gap-2">
                          <span
                            className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                              monitor.enabled
                                ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
                                : "bg-gray-500/10 text-gray-600 dark:text-gray-400 border border-gray-500/20"
                            }`}
                          >
                            <div
                              className={`w-1.5 h-1.5 rounded-full mr-1.5 ${
                                monitor.enabled
                                  ? "bg-emerald-500"
                                  : "bg-gray-400"
                              }`}
                            />
                            {monitor.enabled ? "Active" : "Stopped"}
                          </span>
                        </div>
                      </div>

                      {monitor.name && (
                        <p className="text-gray-600 dark:text-gray-400 text-sm mb-2 truncate">
                          {monitor.host}
                        </p>
                      )}

                      <div className="flex flex-wrap mt-4 gap-x-3 gap-y-1 text-xs text-gray-600 dark:text-gray-400">
                        <span className="flex items-center gap-1">
                          <ClockIcon className="w-3.5 h-3.5 text-blue-500" />
                          Every {formatInterval(monitor.interval)}
                        </span>
                        <span className="flex items-center gap-1">
                          <ChartBarIcon className="w-3.5 h-3.5 text-emerald-500" />
                          {monitor.packetCount} packets
                        </span>
                        <span className="flex items-center gap-1">
                          <ExclamationTriangleIcon className="w-3.5 h-3.5 text-amber-500" />
                          {monitor.threshold}% threshold
                        </span>
                      </div>
                    </div>

                    {/* Action Buttons */}
                    <div className="flex items-center gap-1 ml-4">
                      <Button
                        onClick={(e) => {
                          e.stopPropagation();
                          if (monitor.enabled) {
                            stopMutation.mutate(monitor.id);
                          } else {
                            startMutation.mutate(monitor.id);
                          }
                        }}
                        disabled={
                          startMutation.isPending || stopMutation.isPending
                        }
                        className={`px-1 py-1.5 min-w-8 ${
                          monitor.enabled
                            ? "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/30 hover:bg-red-500/20"
                            : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/20"
                        }`}
                        title={
                          monitor.enabled ? "Stop Monitor" : "Start Monitor"
                        }
                      >
                        {monitor.enabled ? (
                          <StopIcon className="w-3.5 h-3.5" />
                        ) : (
                          <PlayIcon className="w-3.5 h-3.5" />
                        )}
                      </Button>
                      <Button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(monitor);
                        }}
                        className="px-1 py-1.5 min-w-8 bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/30 hover:bg-blue-500/20"
                        title="Edit Monitor"
                      >
                        <PencilIcon className="w-3.5 h-3.5" />
                      </Button>
                      <Button
                        onClick={(e) => {
                          e.stopPropagation();
                          if (
                            confirm(
                              "Are you sure you want to delete this monitor?"
                            )
                          ) {
                            deleteMutation.mutate(monitor.id);
                          }
                        }}
                        disabled={deleteMutation.isPending}
                        className="px-1 py-1.5 min-w-8 bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/30 hover:bg-red-500/20"
                        title="Delete Monitor"
                      >
                        <TrashIcon className="w-3.5 h-3.5" />
                      </Button>
                    </div>
                  </div>
                  {monitor.enabled && (
                    <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700">
                      {(() => {
                        const status = monitorStatuses.get(monitor.id);

                        // State 1: Actively testing (IsRunning=true + Progress>0)
                        if (
                          status &&
                          status.isRunning &&
                          status.progress !== undefined &&
                          status.progress > 0
                        ) {
                          const progressText = `${Math.round(
                            status.progress || 0
                          )}%`;
                          const packetsText = status.packetsSent
                            ? ` (${status.packetsRecv || 0}/${
                                status.packetsSent
                              })`
                            : "";
                          return (
                            <div className="flex items-center gap-2">
                              <div className="flex items-center gap-2">
                                <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
                                <span className="text-blue-600 dark:text-blue-400 text-sm font-medium">
                                  Testing... {progressText}
                                  {packetsText}
                                </span>
                              </div>
                            </div>
                          );
                        }

                        // State 2: Scheduled monitoring (IsRunning=false, IsComplete=false)
                        else if (
                          status &&
                          !status.isRunning &&
                          !status.isComplete
                        ) {
                          // Show last result if available, otherwise show monitoring status
                          if (status.packetLoss !== undefined) {
                            const isHealthy =
                              status.packetLoss <= monitor.threshold;
                            return (
                              <div className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                  <div
                                    className={`w-2 h-2 rounded-full ${
                                      isHealthy
                                        ? "bg-emerald-500"
                                        : "bg-red-500"
                                    }`}
                                  />
                                  <span
                                    className={`text-sm font-medium ${
                                      isHealthy
                                        ? "text-emerald-600 dark:text-emerald-400"
                                        : "text-red-600 dark:text-red-400"
                                    }`}
                                  >
                                    {status.packetLoss.toFixed(1)}% packet loss
                                  </span>
                                </div>
                                {status.avgRtt && (
                                  <span className="text-xs text-gray-500 dark:text-gray-500">
                                    {formatRTT(status.avgRtt)} avg
                                  </span>
                                )}
                              </div>
                            );
                          } else {
                            return (
                              <div className="flex items-center gap-2">
                                <div className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse" />
                                <span className="text-emerald-600 dark:text-emerald-400 text-sm">
                                  Monitoring (every{" "}
                                  {formatInterval(monitor.interval)})
                                </span>
                              </div>
                            );
                          }
                        }

                        // State 3: Disabled (IsComplete=true) or no status yet
                        else {
                          return (
                            <div className="flex items-center gap-2">
                              <div className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse" />
                              <span className="text-emerald-600 dark:text-emerald-400 text-sm">
                                Monitoring (every{" "}
                                {formatInterval(monitor.interval)})
                              </span>
                            </div>
                          );
                        }
                      })()}
                    </div>
                  )}
                </motion.div>
              ))}
            </div>
          )}
        </div>
      </motion.div>

      {/* Right Column - Monitor Details */}
      {selectedMonitor && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="flex-1"
        >
          <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                {selectedMonitor.name || selectedMonitor.host} Details
              </h2>
              {crossTabNavigation && (
                <motion.button
                  onClick={handleTraceRoute}
                  className="flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-xs bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30 hover:bg-amber-500/20"
                  title="Run traceroute to this host"
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                >
                  <GlobeAltIcon className="w-3 h-3" />
                  <span>Trace Route</span>
                </motion.button>
              )}
            </div>

            {/* Current Status - Only show when actively testing or has recent results */}
            {selectedMonitor &&
              selectedMonitor.enabled &&
              (() => {
                const status = monitorStatuses.get(selectedMonitor.id);

                // Only show if actively testing or has results to display
                if (
                  status &&
                  (status.isRunning || status.packetLoss !== undefined)
                ) {
                  return (
                    <div className="mb-6 p-4 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg border border-gray-300 dark:border-gray-800">
                      <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-3">
                        Current Status
                      </h3>
                      {(() => {
                        // Only show testing progress when actively running a test
                        if (
                          status.isRunning &&
                          status.progress !== undefined &&
                          status.progress > 0
                        ) {
                          return (
                            <div className="text-center py-4">
                              <div className="animate-pulse">
                                <p className="text-blue-600 dark:text-blue-400 text-lg font-medium">
                                  Testing in progress...
                                </p>
                                <p className="text-gray-600 dark:text-gray-400 mt-2">
                                  {Math.round(status.progress || 0)}% complete
                                </p>
                                {status.packetsSent && (
                                  <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                                    Packets: {status.packetsRecv || 0} /{" "}
                                    {status.packetsSent} received
                                  </p>
                                )}
                              </div>
                            </div>
                          );
                        }

                        // Show last test results
                        if (
                          !status.isRunning &&
                          !status.isComplete &&
                          status.packetLoss !== undefined
                        ) {
                          return (
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                              <MetricCard
                                icon={
                                  <WifiIcon className="w-5 h-5 text-blue-500" />
                                }
                                title="Packet Loss"
                                value={status.packetLoss?.toFixed(1) || "0"}
                                unit="%"
                                status={
                                  (status.packetLoss || 0) >
                                  selectedMonitor.threshold
                                    ? "error"
                                    : "success"
                                }
                              />
                              <MetricCard
                                icon={
                                  <ClockIcon className="w-5 h-5 text-emerald-500" />
                                }
                                title="Avg RTT"
                                value={formatRTT(status.avgRtt || 0).replace(
                                  "ms",
                                  ""
                                )}
                                unit="ms"
                                status="normal"
                              />
                              <MetricCard
                                icon={
                                  <ArrowTrendingDownIcon className="w-5 h-5 text-green-500" />
                                }
                                title="Min RTT"
                                value={formatRTT(status.minRtt || 0).replace(
                                  "ms",
                                  ""
                                )}
                                unit="ms"
                                status="normal"
                              />
                              <MetricCard
                                icon={
                                  <ArrowTrendingUpIcon className="w-5 h-5 text-yellow-500" />
                                }
                                title="Max RTT"
                                value={formatRTT(status.maxRtt || 0).replace(
                                  "ms",
                                  ""
                                )}
                                unit="ms"
                                status="normal"
                              />
                            </div>
                          );
                        }
                      })()}
                    </div>
                  );
                }
                return null;
              })()}

            {/* Enhanced Performance Chart */}
            {chartData.length > 0 && (
              <div className="mb-6">
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
                  <div className="flex items-center justify-between mb-4">
                    <div>
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                        Performance Trends
                      </h3>
                      <p className="text-gray-600 dark:text-gray-400 text-sm">
                        Last 30 tests • {chartData.length} data points
                      </p>
                    </div>
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

                  <div className="h-80 bg-white/50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200/50 dark:border-gray-700/50">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart
                        data={chartData}
                        key={`chart-${selectedMonitor.id}-${
                          historyList[0]?.id || 0
                        }`}
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
                          formatter={(value: any) => {
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
              </div>
            )}

            {/* Recent Results */}
            <div>
              <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-3">
                Recent Results
              </h3>

              {/* Desktop Table View */}
              <div className="hidden md:block overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-300 dark:border-gray-800">
                      <th className="text-left py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                        Time
                      </th>
                      <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                        Loss
                      </th>
                      <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                        Avg RTT
                      </th>
                      <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                        Min RTT
                      </th>
                      <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                        Max RTT
                      </th>
                      <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                        Sent/Recv
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {historyList.slice(0, 10).map((result) => (
                      <motion.tr
                        key={result.id}
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.3 }}
                        className="border-b border-gray-300/50 dark:border-gray-800/50 last:border-0 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors"
                      >
                        <td className="py-2 px-3 text-gray-700 dark:text-gray-300">
                          {(() => {
                            const date = new Date(result.createdAt);
                            const month = date.toLocaleDateString("en-US", {
                              month: "short",
                            });
                            const day = date.getDate();
                            const hours = date
                              .getHours()
                              .toString()
                              .padStart(2, "0");
                            const minutes = date
                              .getMinutes()
                              .toString()
                              .padStart(2, "0");
                            return `${month} ${day}, ${hours}:${minutes}`;
                          })()}
                        </td>
                        <td
                          className={`py-2 px-3 text-center font-mono ${
                            result.packetLoss > selectedMonitor.threshold
                              ? "text-red-600 dark:text-red-400 font-medium"
                              : "text-emerald-600 dark:text-emerald-400"
                          }`}
                        >
                          {result.packetLoss.toFixed(1)}%
                        </td>
                        <td className="py-2 px-3 text-center font-mono text-blue-600 dark:text-blue-400">
                          {formatRTT(result.avgRtt)}ms
                        </td>
                        <td className="py-2 px-3 text-center font-mono text-emerald-600 dark:text-emerald-400">
                          {formatRTT(result.minRtt)}ms
                        </td>
                        <td className="py-2 px-3 text-center font-mono text-yellow-600 dark:text-yellow-400">
                          {formatRTT(result.maxRtt)}ms
                        </td>
                        <td className="py-2 px-3 text-center text-gray-600 dark:text-gray-400">
                          {result.packetsSent}/{result.packetsRecv}
                        </td>
                      </motion.tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Mobile Card View */}
              <div className="md:hidden space-y-3">
                {historyList.slice(0, 10).map((result) => (
                  <motion.div
                    key={result.id}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3 }}
                    className="bg-gray-200/50 dark:bg-gray-800/50 rounded-lg p-4 border border-gray-300 dark:border-gray-800"
                  >
                    <div className="flex items-center justify-between mb-3">
                      <div className="text-gray-700 dark:text-gray-300 text-sm font-medium">
                        {(() => {
                          const date = new Date(result.createdAt);
                          const month = date.toLocaleDateString("en-US", {
                            month: "short",
                          });
                          const day = date.getDate();
                          const hours = date
                            .getHours()
                            .toString()
                            .padStart(2, "0");
                          const minutes = date
                            .getMinutes()
                            .toString()
                            .padStart(2, "0");
                          return `${month} ${day}, ${hours}:${minutes}`;
                        })()}
                      </div>
                      <span
                        className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                          result.packetLoss > selectedMonitor.threshold
                            ? "bg-red-500/10 text-red-600 dark:text-red-400"
                            : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                        }`}
                      >
                        {result.packetLoss.toFixed(1)}% loss
                      </span>
                    </div>
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-600 dark:text-gray-400">
                          Avg RTT:
                        </span>
                        <span className="text-blue-600 dark:text-blue-400 font-mono">
                          {formatRTT(result.avgRtt)}ms
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-600 dark:text-gray-400">
                          Min RTT:
                        </span>
                        <span className="text-emerald-600 dark:text-emerald-400 font-mono">
                          {formatRTT(result.minRtt)}ms
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-600 dark:text-gray-400">
                          Max RTT:
                        </span>
                        <span className="text-yellow-600 dark:text-yellow-400 font-mono">
                          {formatRTT(result.maxRtt)}ms
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-600 dark:text-gray-400">
                          Packets:
                        </span>
                        <span className="text-gray-600 dark:text-gray-400">
                          {result.packetsRecv}/{result.packetsSent}
                        </span>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            </div>
          </div>
        </motion.div>
      )}

      {/* Enhanced placeholder when no monitor selected */}
      {!selectedMonitor && monitorList.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="flex-1"
        >
          <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
            <div className="text-center mb-6">
              <div className="w-16 h-16 bg-blue-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
                <ChartBarIcon className="w-8 h-8 text-blue-600 dark:text-blue-400" />
              </div>
              <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                Direct Ping Monitoring Dashboard
              </h3>
              <p className="text-gray-600 dark:text-gray-400 text-sm">
                Select a monitor to view packet loss trends and latency
                statistics from continuous ping tests
              </p>
            </div>

            <div className="space-y-4 mb-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
                  <div className="w-8 h-8 bg-emerald-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                    <WifiIcon className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  </div>
                  <div>
                    <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                      Direct ICMP Ping Tests
                    </h4>
                    <p className="text-gray-600 dark:text-gray-400 text-xs">
                      Measures end-to-end connectivity without route tracing
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
                  <div className="w-8 h-8 bg-red-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                    <ExclamationTriangleIcon className="w-4 h-4 text-red-600 dark:text-red-400" />
                  </div>
                  <div>
                    <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                      Smart Alerts
                    </h4>
                    <p className="text-gray-600 dark:text-gray-400 text-xs">
                      Threshold-based notifications for network issues
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
                  <div className="w-8 h-8 bg-blue-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                    <ChartBarIcon className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div>
                    <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                      Historical Analysis
                    </h4>
                    <p className="text-gray-600 dark:text-gray-400 text-xs">
                      Track performance trends over time
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
                  <div className="w-8 h-8 bg-purple-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                    <ClockIcon className="w-4 h-4 text-purple-600 dark:text-purple-400" />
                  </div>
                  <div>
                    <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                      Flexible Scheduling
                    </h4>
                    <p className="text-gray-600 dark:text-gray-400 text-xs">
                      Customizable test intervals and packet counts
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-lg">
              <div className="flex items-start gap-3">
                <SignalIcon className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                <div>
                  <h4 className="text-blue-700 dark:text-blue-300 text-sm font-medium mb-1">
                    Get Started
                  </h4>
                  <p className="text-blue-700/80 dark:text-blue-300/80 text-xs">
                    Select a monitor from the left panel to view detailed
                    metrics, performance charts, and historical data.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      )}

      {/* Add/Edit Monitor Modal */}
      <Transition appear show={showForm} as={Fragment}>
        <Dialog as="div" className="relative z-50" onClose={handleCancelForm}>
          <TransitionChild
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-black bg-opacity-25" />
          </TransitionChild>

          <div className="fixed inset-0 overflow-y-auto backdrop-blur-sm">
            <div className="flex min-h-full items-center justify-center p-4 text-center">
              <TransitionChild
                as={Fragment}
                enter="ease-out duration-300"
                enterFrom="opacity-0 scale-95"
                enterTo="opacity-100 scale-100"
                leave="ease-in duration-200"
                leaveFrom="opacity-100 scale-100"
                leaveTo="opacity-0 scale-95"
              >
                <DialogPanel className="w-full max-w-md transform overflow-visible rounded-2xl backdrop-blur-md bg-white dark:bg-gray-850/95 border dark:border-gray-900 p-4 md:p-6 text-left align-middle shadow-xl transition-all">
                  <div className="flex items-center justify-between mb-4">
                    <DialogTitle
                      as="h3"
                      className="text-lg font-medium leading-6 text-gray-900 dark:text-white"
                    >
                      {editingMonitor ? "Edit Monitor" : "New Monitor"}
                    </DialogTitle>
                    <button
                      onClick={handleCancelForm}
                      className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                    >
                      <XMarkIcon className="h-5 w-5" />
                    </button>
                  </div>

                  <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          Host
                        </label>
                        <input
                          type="text"
                          value={formData.host}
                          onChange={(e) =>
                            setFormData({ ...formData, host: e.target.value })
                          }
                          placeholder="e.g., google.com or 8.8.8.8"
                          className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                          required
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                          Some hosts may block ICMP ping requests
                        </p>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          Name (Optional)
                        </label>
                        <input
                          type="text"
                          value={formData.name}
                          onChange={(e) =>
                            setFormData({ ...formData, name: e.target.value })
                          }
                          placeholder="e.g., Google DNS"
                          className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                        />
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Check Interval
                          </label>
                          <Listbox
                            value={formData.interval}
                            onChange={(value) =>
                              setFormData({ ...formData, interval: value })
                            }
                          >
                            <div className="relative">
                              <ListboxButton className="relative w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-lg text-left text-gray-700 dark:text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                                <span className="block truncate">
                                  {intervalOptions.find(
                                    (opt) => opt.value === formData.interval
                                  )?.label ||
                                    `Every ${formatInterval(
                                      formData.interval
                                    )}`}
                                </span>
                                <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                                  <ChevronUpDownIcon
                                    className="h-5 w-5 text-gray-600 dark:text-gray-400"
                                    aria-hidden="true"
                                  />
                                </span>
                              </ListboxButton>
                              <Transition
                                enter="transition duration-100 ease-out"
                                enterFrom="transform scale-95 opacity-0"
                                enterTo="transform scale-100 opacity-100"
                                leave="transition duration-75 ease-out"
                                leaveFrom="transform scale-100 opacity-100"
                                leaveTo="transform scale-95 opacity-0"
                              >
                                <ListboxOptions className="absolute z-10 mt-1 max-h-80 w-full overflow-auto rounded-lg bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-900 py-1 shadow-lg focus:outline-none">
                                  {intervalOptions.map((option) => (
                                    <ListboxOption
                                      key={option.value}
                                      value={option.value}
                                      className={({ focus }) =>
                                        `relative cursor-pointer select-none py-2 px-4 ${
                                          focus
                                            ? "bg-blue-500/10 text-blue-600 dark:text-blue-200"
                                            : "text-gray-700 dark:text-gray-300"
                                        }`
                                      }
                                    >
                                      {option.label}
                                    </ListboxOption>
                                  ))}
                                </ListboxOptions>
                              </Transition>
                            </div>
                          </Listbox>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Packets per Test
                          </label>
                          <input
                            type="number"
                            value={formData.packetCount}
                            onChange={(e) =>
                              setFormData({
                                ...formData,
                                packetCount: parseInt(e.target.value) || 10,
                              })
                            }
                            min="1"
                            max="100"
                            className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                          />
                        </div>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          Alert Threshold (% packet loss)
                        </label>
                        <input
                          type="number"
                          value={formData.threshold}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              threshold: parseFloat(e.target.value) || 5.0,
                            })
                          }
                          min="0"
                          max="100"
                          step="0.1"
                          className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                        />
                      </div>
                      <div className="flex items-center">
                        <input
                          type="checkbox"
                          id="enabled"
                          checked={formData.enabled}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              enabled: e.target.checked,
                            })
                          }
                          className="mr-2"
                        />
                        <label
                          htmlFor="enabled"
                          className="text-sm font-medium text-gray-700 dark:text-gray-300"
                        >
                          Start monitoring immediately
                        </label>
                      </div>
                    </div>

                    <div className="flex gap-3 pt-4">
                      <Button
                        type="submit"
                        disabled={
                          !formData.host ||
                          createMutation.isPending ||
                          updateMutation.isPending
                        }
                        isLoading={
                          createMutation.isPending || updateMutation.isPending
                        }
                        className="flex-1 bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                      >
                        {editingMonitor ? "Update Monitor" : "Create Monitor"}
                      </Button>
                      <Button
                        type="button"
                        onClick={handleCancelForm}
                        className="bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300"
                      >
                        Cancel
                      </Button>
                    </div>
                  </form>
                </DialogPanel>
              </TransitionChild>
            </div>
          </div>
        </Dialog>
      </Transition>
    </div>
  );
};
