/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, useMemo } from "react";
import { motion } from "motion/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ChevronUpDownIcon,
  ClipboardDocumentIcon,
  ChartBarIcon,
} from "@heroicons/react/24/solid";
import {
  GlobeAltIcon,
  SignalIcon,
  PlusIcon,
} from "@heroicons/react/24/outline";
import {
  TracerouteResult,
  TracerouteUpdate,
  Server,
  SavedIperfServer,
  PacketLossMonitor,
} from "@/types/types";
import { Button } from "@/components/ui/Button";
import {
  formatTracerouteForClipboard,
  copyToClipboard,
  filterTrailingTimeouts,
} from "@/utils/clipboard";
import {
  runTraceroute,
  getTracerouteStatus,
  getServers,
} from "@/api/speedtest";
import {
  getPacketLossMonitors,
  createPacketLossMonitor,
  updatePacketLossMonitor,
  deletePacketLossMonitor,
  startPacketLossMonitor,
  stopPacketLossMonitor,
} from "@/api/packetloss";
import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
  Transition,
} from "@headlessui/react";
import { getApiUrl } from "@/utils/baseUrl";
import { getCountryFlag } from "@/utils/countryFlags";

// Import components from packet loss monitoring
import { PacketLossMonitorList } from "./packetloss/PacketLossMonitorList";
import { PacketLossMonitorForm } from "./packetloss/PacketLossMonitorForm";
import { PacketLossMonitorDetails } from "./packetloss/PacketLossMonitorDetails";
import { EmptyStatePlaceholder } from "./packetloss/components/EmptyStatePlaceholder";
import { usePacketLossMonitorStatus } from "./packetloss/hooks/usePacketLossMonitorStatus";
import {
  MonitorFormData,
  defaultFormData,
} from "./packetloss/constants/packetLossConstants";

// Utility function to extract hostname from any server host value
const extractHostname = (hostValue: string): string => {
  let hostname = hostValue;

  // Extract hostname from URL if it's a full URL (LibreSpeed servers)
  if (hostname.startsWith("http://") || hostname.startsWith("https://")) {
    try {
      const url = new URL(hostname);
      hostname = url.hostname;
    } catch {
      console.warn("Failed to parse URL:", hostname);
    }
  }

  // Strip port from hostname if present (iperf3 and other servers)
  if (hostname.includes(":")) {
    hostname = hostname.split(":")[0];
  }

  return hostname;
};

// Simple flag component using country code from backend
const CountryFlag: React.FC<{ countryCode?: string; className?: string }> = ({
  countryCode,
  className = "w-4 h-3",
}) => {
  if (!countryCode) return null;

  return (
    <span
      className={`text-xs leading-none flex-shrink-0 ${className}`}
      title={countryCode}
      style={{ verticalAlign: "baseline" }}
    >
      {getCountryFlag(countryCode)}
    </span>
  );
};

type TabMode = "traceroute" | "monitors";

export const TracerouteTab: React.FC = () => {
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<TabMode>("traceroute");
  const [host, setHost] = useState("");
  const [tracerouteStatus, setTracerouteStatus] =
    useState<TracerouteUpdate | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copySuccess, setCopySuccess] = useState(false);

  // Packet Loss Monitor state
  const [selectedMonitor, setSelectedMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingMonitor, setEditingMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [formData, setFormData] = useState<MonitorFormData>(defaultFormData);

  // Get cached results from TanStack Query
  const { data: results } = useQuery<TracerouteResult | null>({
    queryKey: ["traceroute", "results"],
    queryFn: () => null,
    enabled: false,
    staleTime: Infinity,
    initialData: null,
  });
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState("");
  const [displayCount, setDisplayCount] = useState(4);
  const [iperfServers, setIperfServers] = useState<SavedIperfServer[]>([]);

  // Fetch monitors
  const { data: monitors, isLoading: monitorsLoading } = useQuery({
    queryKey: ["packetloss", "monitors"],
    queryFn: getPacketLossMonitors,
    staleTime: 30000,
  });

  const monitorList = monitors || [];

  // Use custom hook for monitor status polling
  const monitorStatuses = usePacketLossMonitorStatus(
    monitorList,
    selectedMonitor?.id,
  );

  // Fetch servers
  const { data: speedtestServers = [] } = useQuery({
    queryKey: ["servers", "speedtest"],
    queryFn: () => getServers("speedtest"),
  }) as { data: Server[] };

  const { data: librespeedServers = [] } = useQuery({
    queryKey: ["servers", "librespeed"],
    queryFn: () => getServers("librespeed"),
  }) as { data: Server[] };

  // Fetch iperf servers
  useEffect(() => {
    const fetchIperfServers = async () => {
      try {
        const response = await fetch(getApiUrl("/iperf/servers"));
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        setIperfServers(data || []);
      } catch (error) {
        console.error("Failed to fetch iperf servers:", error);
      }
    };
    fetchIperfServers();
  }, []);

  // Convert iperf servers to Server format and combine with other servers
  const allServers = useMemo(() => {
    const iperfServerList: Server[] = iperfServers.map((server) => ({
      id: `iperf3-${server.host}:${server.port}`,
      name: server.name,
      host: `${server.host}:${server.port}`,
      location: "Saved",
      distance: 0,
      country: "Saved",
      sponsor: "iperf3",
      latitude: 0,
      longitude: 0,
      isIperf: true,
      isLibrespeed: false,
    }));

    return [...speedtestServers, ...librespeedServers, ...iperfServerList];
  }, [speedtestServers, librespeedServers, iperfServers]);

  // Server type options for filtering
  const serverTypes = [
    { value: "", label: "All Types" },
    { value: "speedtest", label: "Speedtest.net" },
    { value: "iperf3", label: "iperf3" },
    { value: "librespeed", label: "LibreSpeed" },
  ];

  // Filter and sort servers
  const filteredServers = useMemo(() => {
    const filtered = allServers.filter((server) => {
      const matchesSearch =
        searchTerm === "" ||
        server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.sponsor.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.country.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.host.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesType =
        filterType === "" ||
        (filterType === "speedtest" &&
          !server.isIperf &&
          !server.isLibrespeed) ||
        (filterType === "iperf3" && server.isIperf) ||
        (filterType === "librespeed" && server.isLibrespeed);

      return matchesSearch && matchesType;
    });

    return filtered.sort((a, b) => {
      // Sort iperf servers by name, others by distance
      if (a.isIperf && b.isIperf) {
        return a.name.localeCompare(b.name);
      }
      return a.distance - b.distance;
    });
  }, [allServers, searchTerm, filterType]);

  const tracerouteMutation = useMutation({
    mutationFn: runTraceroute,
    onMutate: () => {
      // Clear previous results and error state
      queryClient.setQueryData(["traceroute", "results"], null);
      setError(null);
      setTracerouteStatus({
        type: "traceroute",
        host: host,
        progress: 0,
        isComplete: false,
        currentHop: 0,
        totalHops: 30,
        isScheduled: false,
        hops: [],
        destination: host,
        ip: "",
      });
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["traceroute", "results"], data);
      setTracerouteStatus(null);
      setError(null);
    },
    onError: (error) => {
      console.error("Traceroute failed:", error);
      setTracerouteStatus(null);
      setError(
        error.message ||
          "Traceroute failed. Please check the hostname and try again.",
      );
    },
  });

  // Poll for traceroute status
  const { data: statusData } = useQuery({
    queryKey: ["traceroute", "status"],
    queryFn: getTracerouteStatus,
    refetchInterval:
      tracerouteMutation.isPending ||
      (tracerouteStatus && !tracerouteStatus.isComplete)
        ? 1000
        : false,
    enabled:
      tracerouteMutation.isPending ||
      Boolean(tracerouteStatus && !tracerouteStatus.isComplete),
  });

  // Update status when we get new data
  useEffect(() => {
    if (statusData && !results) {
      setTracerouteStatus(statusData);
    } else if (results) {
      // Clear status when we have final results
      setTracerouteStatus(null);
    }
  }, [statusData, results]);

  // Mutations for monitors
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

  const handleRunTraceroute = () => {
    let targetHost = selectedServer ? selectedServer.host : host.trim();
    if (!targetHost) return;

    // Extract hostname for all server types
    targetHost = extractHostname(targetHost);

    // Clear previous state before starting
    queryClient.setQueryData(["traceroute", "results"], null);
    setTracerouteStatus(null);
    tracerouteMutation.mutate(targetHost);
  };

  const handleServerSelect = (server: Server) => {
    const isSelected = selectedServer?.id === server.id;
    if (isSelected) {
      // Unselect the server
      setSelectedServer(null);
      setHost("");
    } else {
      // Select the server
      setSelectedServer(server);
      // Extract hostname for all server types
      setHost(extractHostname(server.host));
    }
  };

  const formatRTT = (rtt: number) => {
    if (rtt === 0) return "*";
    return `${rtt.toFixed(1)}ms`;
  };

  const getAverageRTT = (hop: {
    rtt1: number;
    rtt2: number;
    rtt3: number;
    timeout: boolean;
  }) => {
    if (hop.timeout) return 0;
    const validRTTs = [hop.rtt1, hop.rtt2, hop.rtt3].filter((rtt) => rtt > 0);
    if (validRTTs.length === 0) return 0;
    return validRTTs.reduce((sum, rtt) => sum + rtt, 0) / validRTTs.length;
  };

  const copyTracerouteResults = async () => {
    if (!results) return;

    const formattedOutput = formatTracerouteForClipboard(results);
    const success = await copyToClipboard(formattedOutput);

    if (success) {
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    }
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

  // Handlers for monitors
  const handleSubmit = (data: MonitorFormData) => {
    if (editingMonitor) {
      updateMutation.mutate({
        ...editingMonitor,
        ...data,
      });
    } else {
      createMutation.mutate(data);
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
      <div className="flex gap-2 p-1 bg-gray-200 dark:bg-gray-800 rounded-lg">
        <button
          onClick={() => setMode("traceroute")}
          className={`flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-md transition-colors ${
            mode === "traceroute"
              ? "bg-white dark:bg-gray-900 text-blue-600 dark:text-blue-400 shadow-sm"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
          }`}
        >
          <GlobeAltIcon className="w-4 h-4" />
          <span className="font-medium">Single Trace</span>
        </button>
        <button
          onClick={() => setMode("monitors")}
          className={`flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-md transition-colors ${
            mode === "monitors"
              ? "bg-white dark:bg-gray-900 text-purple-600 dark:text-purple-400 shadow-sm"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
          }`}
        >
          <ChartBarIcon className="w-4 h-4" />
          <span className="font-medium">Monitors</span>
        </button>
      </div>

      {mode === "traceroute" ? (
        <div className="flex flex-col md:flex-row gap-6 md:items-start">
          {/* Left Column - Server Selection */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="flex-1"
          >
            <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
              <div className="mb-6">
                <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                  Traceroute
                </h2>
                <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                  Trace network paths to see which backbone providers you travel
                  through
                </p>
              </div>

              {/* Manual Input Section */}
              <div className="flex gap-3 mb-6">
                <input
                  type="text"
                  value={host}
                  onChange={(e) => {
                    setHost(e.target.value);
                    setSelectedServer(null);
                  }}
                  placeholder="Enter hostname/IP (e.g., google.com, 8.8.8.8)"
                  className="flex-1 px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                  disabled={tracerouteMutation.isPending}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !tracerouteMutation.isPending) {
                      handleRunTraceroute();
                    }
                  }}
                />
                <Button
                  onClick={handleRunTraceroute}
                  disabled={!host.trim() || tracerouteMutation.isPending}
                  isLoading={tracerouteMutation.isPending}
                  className={`${
                    tracerouteMutation.isPending
                      ? "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-300 dark:border-emerald-500/30 cursor-not-allowed"
                      : !host.trim()
                        ? "bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500 border-gray-200 dark:border-gray-700 cursor-not-allowed"
                        : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                  }`}
                >
                  {tracerouteMutation.isPending ? "Running..." : "Trace"}
                </Button>
              </div>

              {/* Separator with helper text */}
              <div className="relative mb-6">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-300 dark:border-gray-700"></div>
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-4 bg-gray-50 dark:bg-gray-850 text-gray-600 dark:text-gray-400">
                    or select from available servers
                  </span>
                </div>
              </div>

              {/* Server Search and Filter Controls */}
              <div className="flex flex-col md:flex-row gap-4 mb-4">
                {/* Search Input */}
                <div className="flex-1">
                  <input
                    type="text"
                    placeholder="Search servers..."
                    className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                  />
                </div>

                {/* Server Type Filter */}
                <Listbox value={filterType} onChange={setFilterType}>
                  <div className="relative min-w-[160px]">
                    <ListboxButton className="relative w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 rounded-lg text-left text-gray-700 dark:text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                      <span className="block truncate">
                        {serverTypes.find((type) => type.value === filterType)
                          ?.label || "All Types"}
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
                      <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-800 py-1 shadow-lg focus:outline-none">
                        {serverTypes.map((type) => (
                          <ListboxOption
                            key={type.value}
                            value={type.value}
                            className={({ focus }) =>
                              `relative cursor-pointer select-none py-2 px-4 ${
                                focus
                                  ? "bg-blue-500/10 text-blue-600 dark:text-blue-200"
                                  : "text-gray-700 dark:text-gray-300"
                              }`
                            }
                          >
                            {type.label}
                          </ListboxOption>
                        ))}
                      </ListboxOptions>
                    </Transition>
                  </div>
                </Listbox>
              </div>

              {/* Server Grid */}
              <div className="grid grid-cols-1 gap-4 mb-4">
                {filteredServers.slice(0, displayCount).map((server) => (
                  <motion.div
                    key={server.id}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3 }}
                  >
                    <button
                      onClick={() => handleServerSelect(server)}
                      className={`w-full p-3 rounded-lg text-left transition-colors ${
                        selectedServer?.id === server.id
                          ? "bg-blue-500/10 border-blue-400/50 shadow-lg"
                          : "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 shadow-lg"
                      } border`}
                      disabled={tracerouteMutation.isPending}
                    >
                      <div className="flex flex-col gap-1">
                        <span className="text-blue-600 dark:text-blue-300 font-medium truncate">
                          {server.isIperf ? server.name : server.sponsor}
                        </span>
                        <span className="text-gray-600 dark:text-gray-400 text-sm">
                          {server.isIperf ? "iperf3 Server" : server.name}
                          <span
                            className="block truncate text-xs"
                            title={server.host}
                          >
                            {extractHostname(server.host)}
                          </span>
                        </span>
                        <span className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                          {server.isIperf
                            ? "Custom Server"
                            : `${server.country} - ${Math.floor(
                                server.distance,
                              )} km`}
                          {server.isLibrespeed && (
                            <span className="ml-2 text-blue-600 dark:text-blue-400 drop-shadow-[0_0_1px_rgba(96,165,250,0.8)]">
                              librespeed
                            </span>
                          )}
                          {server.isIperf && (
                            <span className="ml-2 text-purple-600 dark:text-purple-400 drop-shadow-[0_0_1px_rgba(168,85,247,0.8)]">
                              iperf3
                            </span>
                          )}
                          {!server.isIperf && !server.isLibrespeed && (
                            <span className="ml-2 text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_1px_rgba(251,191,36,0.8)]">
                              speedtest.net
                            </span>
                          )}
                        </span>
                      </div>
                    </button>
                  </motion.div>
                ))}
              </div>

              {/* Load More Button */}
              {filteredServers.length > displayCount && (
                <div className="flex justify-center mb-4">
                  <button
                    onClick={() => setDisplayCount((prev) => prev + 4)}
                    className="px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300/80 dark:border-gray-800/80 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-800 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800 transition-colors"
                  >
                    Load More
                  </button>
                </div>
              )}
            </div>
          </motion.div>

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
            {tracerouteStatus && !tracerouteStatus.isComplete && (
              <div className="mb-6 p-4 backdrop-blur-sm bg-blue-500/10 border border-blue-500/30 rounded-lg">
                <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 mb-3">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 dark:border-blue-400"></div>
                  <span>Running traceroute to {tracerouteStatus.host}...</span>
                </div>
                <div className="w-full bg-gray-300 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className="bg-blue-500 h-2 rounded-full transition-all duration-500"
                    style={{
                      width: `${Math.min(tracerouteStatus.progress, 100)}%`,
                    }}
                  ></div>
                </div>
                <div className="text-sm text-blue-600 dark:text-blue-300 mt-2">
                  Hop {tracerouteStatus.currentHop} of{" "}
                  {tracerouteStatus.totalHops} (
                  {Math.round(tracerouteStatus.progress)}%)
                </div>
              </div>
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
              <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                    Traceroute Results
                  </h2>
                  <div className="flex items-center gap-2">
                    <motion.button
                      onClick={handleCreateMonitorFromTraceroute}
                      className="flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-xs bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/30 hover:bg-purple-500/20"
                      title="Create packet loss monitor for this host"
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                    >
                      <SignalIcon className="w-3 h-3" />
                      <span>Monitor This Host</span>
                    </motion.button>
                    <motion.button
                      onClick={copyTracerouteResults}
                      className={`flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-xs ${
                        copySuccess
                          ? "bg-emerald-600 text-white"
                          : "bg-gray-300 dark:bg-gray-700 hover:bg-gray-400 dark:hover:bg-gray-600 text-gray-900 dark:text-white"
                      }`}
                      title="Copy traceroute results to clipboard"
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                    >
                      <motion.div
                        animate={{
                          rotate: copySuccess ? 360 : 0,
                          scale: copySuccess ? 1.1 : 1,
                        }}
                        transition={{ duration: 0.3 }}
                      >
                        <ClipboardDocumentIcon className="w-3 h-3" />
                      </motion.div>
                      <motion.span
                        key={copySuccess ? "copied" : "copy"}
                        initial={{ opacity: 0, y: -10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: 10 }}
                        transition={{ duration: 0.2 }}
                      >
                        {copySuccess ? "Copied!" : "Copy"}
                      </motion.span>
                    </motion.button>
                  </div>
                </div>
                <p className="text-gray-600 dark:text-gray-400 text-sm mb-6">
                  Route to {results.destination}
                  {results.ip && results.ip !== results.destination && (
                    <span className="text-gray-500 dark:text-gray-500 ml-1">
                      ({results.ip})
                    </span>
                  )}
                  {" • "}
                  {results.totalHops} hops •{" "}
                  {results.complete ? "Complete" : "Incomplete"}
                </p>

                {/* Desktop Table View */}
                <div className="hidden md:block">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-800">
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          Hop
                        </th>
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          Host
                        </th>
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          Provider
                        </th>
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          RTT 1
                        </th>
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          RTT 2
                        </th>
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          RTT 3
                        </th>
                        <th className="text-center py-3 px-2 text-gray-400 font-medium">
                          Average
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {filterTrailingTimeouts(results.hops).map((hop) => (
                        <motion.tr
                          key={hop.number}
                          initial={{ opacity: 0, y: 20 }}
                          animate={{ opacity: 1, y: 0 }}
                          transition={{ duration: 0.3 }}
                          className="border-b border-gray-300/50 dark:border-gray-800/50 last:border-0 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors"
                        >
                          <td className="py-3 px-2 text-gray-700 dark:text-gray-300 text-center">
                            {hop.number}
                          </td>
                          <td
                            className="py-3 px-2 text-gray-700 dark:text-gray-300 text-center"
                            title={hop.timeout ? "Request timed out" : hop.host}
                          >
                            {hop.timeout ? (
                              <span className="text-gray-500 dark:text-gray-500">
                                Timeout
                              </span>
                            ) : (
                              hop.host
                            )}
                          </td>
                          <td className="py-3 px-2 min-w-[200px] max-w-[200px] text-center">
                            {hop.as ? (
                              <div className="flex items-center justify-center gap-2">
                                <CountryFlag
                                  countryCode={hop.countryCode}
                                  className="w-4 h-3 flex-shrink-0"
                                />
                                <span
                                  className="text-blue-600 dark:text-blue-400 text-xs truncate"
                                  title={hop.as}
                                >
                                  {hop.as}
                                </span>
                              </div>
                            ) : (
                              <span className="text-gray-500 dark:text-gray-500">
                                —
                              </span>
                            )}
                          </td>
                          <td className="py-3 px-2 text-center font-mono">
                            {hop.timeout ? (
                              <span className="text-gray-500">*</span>
                            ) : (
                              <span className="text-emerald-600 dark:text-emerald-400">
                                {formatRTT(hop.rtt1)}
                              </span>
                            )}
                          </td>
                          <td className="py-3 px-2 text-center font-mono">
                            {hop.timeout ? (
                              <span className="text-gray-500">*</span>
                            ) : (
                              <span className="text-yellow-600 dark:text-yellow-400">
                                {formatRTT(hop.rtt2)}
                              </span>
                            )}
                          </td>
                          <td className="py-3 px-2 text-center font-mono">
                            {hop.timeout ? (
                              <span className="text-gray-500">*</span>
                            ) : (
                              <span className="text-orange-600 dark:text-orange-400">
                                {formatRTT(hop.rtt3)}
                              </span>
                            )}
                          </td>
                          <td className="py-3 px-2 text-center font-mono">
                            {hop.timeout ? (
                              <span className="text-gray-500">*</span>
                            ) : (
                              <span className="text-blue-600 dark:text-blue-400">
                                {formatRTT(getAverageRTT(hop))}
                              </span>
                            )}
                          </td>
                        </motion.tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
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
                  <PlusIcon className="w-4 h-4 mr-2" />
                  Add Monitor
                </Button>
              </div>

              {/* Monitor List Component */}
              <PacketLossMonitorList
                monitors={monitorList}
                selectedMonitor={selectedMonitor}
                monitorStatuses={monitorStatuses}
                onMonitorSelect={setSelectedMonitor}
                onEdit={handleEdit}
                onDelete={(id) => deleteMutation.mutate(id)}
                onToggle={handleToggle}
                isLoading={monitorsLoading}
                isToggling={startMutation.isPending || stopMutation.isPending}
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
