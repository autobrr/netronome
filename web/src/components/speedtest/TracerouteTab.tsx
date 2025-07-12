/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { motion } from "motion/react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ChartBarIcon, GlobeAltIcon } from "@heroicons/react/24/outline";
import {
  TracerouteResult,
  TracerouteUpdate,
  Server,
  PacketLossMonitor,
} from "@/types/types";
import { Button } from "@/components/ui/Button";

// Import packet loss components
import { PacketLossMonitorList } from "./packetloss/PacketLossMonitorList";
import { PacketLossMonitorForm } from "./packetloss/PacketLossMonitorForm";
import { PacketLossMonitorDetails } from "./packetloss/PacketLossMonitorDetails";
import { EmptyStatePlaceholder } from "./packetloss/components/EmptyStatePlaceholder";
import { usePacketLossMonitorStatus } from "./packetloss/hooks/usePacketLossMonitorStatus";
import {
  MonitorFormData,
  defaultFormData,
} from "./packetloss/constants/packetLossConstants";

// Import packet loss API functions
import {
  getPacketLossMonitors,
  createPacketLossMonitor,
  updatePacketLossMonitor,
  deletePacketLossMonitor,
  startPacketLossMonitor,
  stopPacketLossMonitor,
} from "@/api/packetloss";

// Import new traceroute components
import { TracerouteServerSelector } from "./traceroute/TracerouteServerSelector";
import { TracerouteProgress } from "./traceroute/TracerouteProgress";
import { TracerouteLiveResults } from "./traceroute/TracerouteLiveResults";
import { TracerouteResults } from "./traceroute/TracerouteResults";

// Import new traceroute hooks
import { useTracerouteExecution } from "./traceroute/hooks/useTracerouteExecution";
import { useTracerouteStatus } from "./traceroute/hooks/useTracerouteStatus";

// Import traceroute utilities
import { extractHostname } from "./traceroute/utils/tracerouteUtils";

// Import constants
import {
  TabMode,
  DEFAULT_TAB_MODE,
  TAB_MODE_STORAGE_KEY,
} from "./traceroute/constants/tracerouteConstants";

export const TracerouteTab: React.FC = () => {
  const queryClient = useQueryClient();

  // Tab mode state with localStorage persistence
  const [mode, setMode] = useState<TabMode>(() => {
    const saved = localStorage.getItem(TAB_MODE_STORAGE_KEY);
    return saved === "traceroute" || saved === "monitors"
      ? saved
      : DEFAULT_TAB_MODE;
  });

  // Traceroute state
  const [host, setHost] = useState("");
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [tracerouteStatus, setTracerouteStatus] =
    useState<TracerouteUpdate | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Packet Loss Monitor state
  const [selectedMonitor, setSelectedMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingMonitor, setEditingMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [formData, setFormData] = useState<MonitorFormData>(defaultFormData);

  // Save tab mode to localStorage when it changes
  useEffect(() => {
    localStorage.setItem(TAB_MODE_STORAGE_KEY, mode);
  }, [mode]);

  // Get cached results from TanStack Query
  const { data: results } = useQuery<TracerouteResult | null>({
    queryKey: ["traceroute", "results"],
    queryFn: () => null,
    enabled: false,
    staleTime: Infinity,
    initialData: null,
  });

  // Traceroute execution hook
  const { runTraceroute, isRunning } = useTracerouteExecution({
    onStatusUpdate: setTracerouteStatus,
    onError: setError,
  });

  // Traceroute status polling hook
  useTracerouteStatus({
    isRunning,
    tracerouteStatus,
    results,
    onStatusUpdate: setTracerouteStatus,
  });

  // Packet Loss Monitor queries and mutations
  const { data: monitors, isLoading: monitorsLoading } = useQuery({
    queryKey: ["packetloss", "monitors"],
    queryFn: getPacketLossMonitors,
    staleTime: 30000,
  });

  const monitorList = monitors || [];
  const monitorStatuses = usePacketLossMonitorStatus(
    monitorList,
    selectedMonitor?.id,
  );

  const createMutation = useMutation({
    mutationFn: createPacketLossMonitor,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      handleCancelForm();
    },
  });

  const updateMutation = useMutation({
    mutationFn: updatePacketLossMonitor,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      handleCancelForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deletePacketLossMonitor,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
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
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      queryClient.invalidateQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
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
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      queryClient.invalidateQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
    },
  });

  // Event handlers
  const handleRunTraceroute = () => {
    const targetHost = selectedServer ? selectedServer.host : host.trim();
    if (!targetHost) return;
    runTraceroute(host, selectedServer?.host);
  };

  const handleCreateMonitorFromTraceroute = () => {
    if (!results) return;

    const hostname = extractHostname(results.destination);
    setFormData({
      ...defaultFormData,
      host: hostname,
      name: `Monitor for ${hostname}`,
      enabled: true,
    });
    setMode("monitors");
    setShowForm(true);
  };

  const handleSubmit = (data: MonitorFormData) => {
    // Convert scheduleType and exactTimes to interval format
    let interval = data.interval;
    if (
      data.scheduleType === "exact" &&
      data.exactTimes &&
      data.exactTimes.length > 0
    ) {
      interval = `exact:${data.exactTimes.join(",")}`;
    }

    const monitorData = {
      ...data,
      interval,
      scheduleType: undefined,
      exactTimes: undefined,
    };

    if (editingMonitor) {
      updateMutation.mutate({ ...editingMonitor, ...monitorData });
    } else {
      createMutation.mutate(monitorData);
    }
  };

  const handleEdit = (monitor: PacketLossMonitor) => {
    setEditingMonitor(monitor);

    // Parse interval format back to scheduleType and exactTimes
    let scheduleType: "interval" | "exact" = "interval";
    let exactTimes: string[] = [];
    let interval = monitor.interval;

    if (monitor.interval.startsWith("exact:")) {
      scheduleType = "exact";
      const timePart = monitor.interval.substring(6);
      exactTimes = timePart.split(",").map((t) => t.trim());
      interval = "1h";
    }

    setFormData({
      host: monitor.host,
      name: monitor.name || "",
      interval,
      scheduleType,
      exactTimes,
      packetCount: monitor.packetCount,
      threshold: monitor.threshold,
      enabled: monitor.enabled,
    });
    setShowForm(true);
  };

  const handleCancelForm = () => {
    setShowForm(false);
    setEditingMonitor(null);
    setFormData(defaultFormData);
  };

  const handleToggle = (monitorId: number, enabled: boolean) => {
    if (enabled) {
      startMutation.mutate(monitorId);
    } else {
      stopMutation.mutate(monitorId);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Mode Selector Tabs */}
      <div className="flex gap-2 p-1 bg-gray-50/95 dark:bg-gray-850 rounded-lg border border-gray-200 dark:border-gray-800 max-w-md mx-auto md:mx-0 md:max-w-none">
        <button
          onClick={() => setMode("traceroute")}
          className={`relative flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg transition-all duration-200 ${
            mode === "traceroute"
              ? "text-blue-600 dark:text-blue-400"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
          }`}
        >
          {mode === "traceroute" && (
            <motion.div
              layoutId="activeTabMode"
              className="absolute inset-0 bg-white/80 dark:bg-gray-800/80 rounded-lg shadow-sm border border-gray-200/40 dark:border-gray-700/80"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ type: "tween", duration: 0.2 }}
            />
          )}
          <span className="relative flex items-center gap-2">
            <GlobeAltIcon className="w-4 h-4" />
            <span className="font-medium">Single Trace</span>
          </span>
        </button>
        <button
          onClick={() => setMode("monitors")}
          className={`relative flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg transition-all duration-200 ${
            mode === "monitors"
              ? "text-purple-600 dark:text-purple-400"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
          }`}
        >
          {mode === "monitors" && (
            <motion.div
              layoutId="activeTabMode"
              className="absolute inset-0 bg-white/80 dark:bg-gray-800/80 rounded-lg shadow-sm border border-gray-200/40 dark:border-gray-700/80"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ type: "tween", duration: 0.2 }}
            />
          )}
          <span className="relative flex items-center gap-2">
            <ChartBarIcon className="w-4 h-4" />
            <span className="font-medium">Monitors</span>
          </span>
        </button>
      </div>

      {mode === "traceroute" ? (
        <div className="flex flex-col md:flex-row gap-6 md:items-start">
          {/* Left Column - Server Selection */}
          <TracerouteServerSelector
            host={host}
            selectedServer={selectedServer}
            onHostChange={setHost}
            onServerSelect={setSelectedServer}
            onRunTraceroute={handleRunTraceroute}
            isRunning={isRunning}
          />

          {/* Right Column - Results */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="flex-1"
          >
            {/* Error Display */}
            {error && (
              <div className="mb-6 p-4 backdrop-blur-sm bg-red-500/10 border border-red-500/30 rounded-lg">
                <div className="text-red-600 dark:text-red-400 text-sm">
                  <span className="font-medium">Error: </span>
                  {error}
                </div>
              </div>
            )}

            {/* Progress */}
            {tracerouteStatus && (
              <TracerouteProgress tracerouteStatus={tracerouteStatus} />
            )}

            {/* Live Results */}
            {tracerouteStatus && (
              <TracerouteLiveResults tracerouteStatus={tracerouteStatus} />
            )}

            {/* Placeholder when no results */}
            {!results && !tracerouteStatus && !error && (
              <EmptyStatePlaceholder
                mode="traceroute"
                onHostSelect={(hostname) => {
                  setHost(hostname);
                  setSelectedServer(null);
                }}
                onSwitchToMonitors={() => setMode("monitors")}
              />
            )}

            {/* Final Results */}
            {results && (
              <TracerouteResults
                results={results}
                onCreateMonitor={handleCreateMonitorFromTraceroute}
              />
            )}
          </motion.div>
        </div>
      ) : (
        /* Monitors Mode */
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
                    Network Monitors
                  </h2>
                  <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                    Continuous monitoring with MTR or ICMP ping
                  </p>
                </div>

                <Button
                  onClick={() => setShowForm(true)}
                  className="bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                >
                  Add
                </Button>
              </div>

              <PacketLossMonitorList
                monitors={monitorList}
                selectedMonitor={selectedMonitor}
                monitorStatuses={monitorStatuses}
                onMonitorSelect={setSelectedMonitor}
                onEdit={handleEdit}
                onDelete={(id) => deleteMutation.mutate(id)}
                onToggle={handleToggle}
                isLoading={monitorsLoading}
                togglingMonitorId={
                  startMutation.isPending
                    ? startMutation.variables
                    : stopMutation.isPending
                      ? stopMutation.variables
                      : null
                }
              />
            </div>
          </motion.div>

          {/* Right Column - Monitor Details */}
          {selectedMonitor && (
            <PacketLossMonitorDetails
              selectedMonitor={selectedMonitor}
              monitorStatuses={monitorStatuses}
            />
          )}
          {!selectedMonitor && monitorList.length > 0 && (
            <EmptyStatePlaceholder mode="monitors" />
          )}
        </div>
      )}

      {/* Add/Edit Monitor Modal */}
      <PacketLossMonitorForm
        showForm={showForm}
        onClose={handleCancelForm}
        onSubmit={handleSubmit}
        editingMonitor={editingMonitor}
        formData={formData}
        onFormDataChange={setFormData}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />
    </div>
  );
};
