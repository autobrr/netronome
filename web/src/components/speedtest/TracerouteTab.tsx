/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, useMemo } from "react";
import { motion } from "motion/react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  PlayIcon,
  StopIcon,
  ChevronUpDownIcon,
} from "@heroicons/react/24/solid";
import { TracerouteResult, Server, SavedIperfServer } from "@/types/types";
import { runTraceroute, getServers } from "@/api/speedtest";
import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
  Transition,
} from "@headlessui/react";
import { getApiUrl } from "@/utils/baseUrl";
import { getCountryFlag } from "@/utils/countryFlags";

// Utility function to extract hostname from any server host value
const extractHostname = (hostValue: string): string => {
  let hostname = hostValue;

  // Extract hostname from URL if it's a full URL (LibreSpeed servers)
  if (hostname.startsWith("http://") || hostname.startsWith("https://")) {
    try {
      const url = new URL(hostname);
      hostname = url.hostname;
    } catch (error) {
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
    <span className={`text-xs leading-none flex-shrink-0 ${className}`} title={countryCode} style={{ verticalAlign: 'baseline' }}>
      {getCountryFlag(countryCode)}
    </span>
  );
};

export const TracerouteTab: React.FC = () => {
  const [host, setHost] = useState("");
  const [results, setResults] = useState<TracerouteResult | null>(null);
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState("");
  const [displayCount, setDisplayCount] = useState(6);
  const [iperfServers, setIperfServers] = useState<SavedIperfServer[]>([]);

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
    onSuccess: (data) => {
      setResults(data);
    },
    onError: (error) => {
      console.error("Traceroute failed:", error);
    },
  });

  const handleRunTraceroute = () => {
    let targetHost = selectedServer ? selectedServer.host : host.trim();
    if (!targetHost) return;

    // Extract hostname for all server types
    targetHost = extractHostname(targetHost);

    setResults(null);
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

  const handleStop = () => {
    tracerouteMutation.reset();
    setResults(null);
  };

  const formatRTT = (rtt: number) => {
    if (rtt === 0) return "*";
    return `${rtt.toFixed(1)}ms`;
  };

  const getAverageRTT = (hop: { rtt1: number; rtt2: number; rtt3: number; timeout: boolean }) => {
    if (hop.timeout) return 0;
    const validRTTs = [hop.rtt1, hop.rtt2, hop.rtt3].filter((rtt) => rtt > 0);
    if (validRTTs.length === 0) return 0;
    return validRTTs.reduce((sum, rtt) => sum + rtt, 0) / validRTTs.length;
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="w-full max-w-6xl mx-auto"
    >
      <div className="bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-900">
        <div className="mb-6">
          <h2 className="text-xl font-semibold text-white">Traceroute</h2>
          <p className="text-gray-400 text-sm mt-1">
            Trace network paths to see which backbone providers you travel
            through
          </p>
        </div>

        {/* Server Search and Filter Controls */}
        <div className="flex flex-col md:flex-row gap-4 mb-4">
          {/* Search Input */}
          <div className="flex-1">
            <input
              type="text"
              placeholder="Search servers..."
              className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>

          {/* Server Type Filter */}
          <Listbox value={filterType} onChange={setFilterType}>
            <div className="relative min-w-[160px]">
              <ListboxButton className="relative w-full px-4 py-2 bg-gray-800/50 border border-gray-900 rounded-lg text-left text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                <span className="block truncate">
                  {serverTypes.find((type) => type.value === filterType)
                    ?.label || "All Types"}
                </span>
                <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                  <ChevronUpDownIcon
                    className="h-5 w-5 text-gray-400"
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
                <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-gray-800 border border-gray-900 py-1 shadow-lg focus:outline-none">
                  {serverTypes.map((type) => (
                    <ListboxOption
                      key={type.value}
                      value={type.value}
                      className={({ focus }) =>
                        `relative cursor-pointer select-none py-2 px-4 ${
                          focus
                            ? "bg-blue-500/10 text-blue-200"
                            : "text-gray-300"
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
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
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
                    : "bg-gray-800/50 border-gray-900 hover:bg-gray-800 shadow-lg"
                } border`}
                disabled={tracerouteMutation.isPending}
              >
                <div className="flex flex-col gap-1">
                  <span className="text-blue-300 font-medium truncate">
                    {server.isIperf ? server.name : server.sponsor}
                  </span>
                  <span className="text-gray-400 text-sm">
                    {server.isIperf ? "iperf3 Server" : server.name}
                    <span
                      className="block truncate text-xs"
                      title={server.host}
                    >
                      {extractHostname(server.host)}
                    </span>
                  </span>
                  <span className="text-gray-400 text-sm mt-1">
                    {server.isIperf
                      ? "Custom Server"
                      : `${server.country} - ${Math.floor(server.distance)} km`}
                    {server.isLibrespeed && (
                      <span className="ml-2 text-blue-400 drop-shadow-[0_0_1px_rgba(96,165,250,0.8)]">
                        librespeed
                      </span>
                    )}
                    {server.isIperf && (
                      <span className="ml-2 text-purple-400 drop-shadow-[0_0_1px_rgba(168,85,247,0.8)]">
                        iperf3
                      </span>
                    )}
                    {!server.isIperf && !server.isLibrespeed && (
                      <span className="ml-2 text-emerald-400 drop-shadow-[0_0_1px_rgba(251,191,36,0.8)]">
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
              onClick={() => setDisplayCount((prev) => prev + 6)}
              className="px-4 py-2 bg-gray-800/50 border border-gray-900/80 text-gray-300/50 hover:text-gray-300 rounded-lg hover:bg-gray-800 transition-colors"
            >
              Load More
            </button>
          </div>
        )}

        {/* Manual Input Section */}
        <div className="flex gap-3 mb-4">
          <input
            type="text"
            value={host}
            onChange={(e) => {
              setHost(e.target.value);
              setSelectedServer(null);
            }}
            placeholder="Or enter custom hostname/IP (e.g., google.com, 8.8.8.8)"
            className="flex-1 px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
            disabled={tracerouteMutation.isPending}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !tracerouteMutation.isPending) {
                handleRunTraceroute();
              }
            }}
          />
          <motion.button
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            onClick={
              tracerouteMutation.isPending ? handleStop : handleRunTraceroute
            }
            disabled={!host.trim() && !tracerouteMutation.isPending}
            className={`px-6 py-2 rounded-lg font-medium transition-colors border shadow-md flex items-center gap-2 min-w-[100px] justify-center ${
              tracerouteMutation.isPending
                ? "bg-red-500 hover:bg-red-600 text-white border-red-600 hover:border-red-700"
                : "bg-blue-500 hover:bg-blue-600 text-white disabled:bg-gray-700 disabled:text-gray-400 disabled:cursor-not-allowed border-blue-600 hover:border-blue-700 disabled:border-gray-900"
            }`}
          >
            {tracerouteMutation.isPending ? (
              <>
                <StopIcon className="w-4 h-4" />
                Stop
              </>
            ) : (
              <>
                <PlayIcon className="w-4 h-4" />
                Trace
              </>
            )}
          </motion.button>
        </div>

        {/* Results */}
        {tracerouteMutation.isPending && (
          <div className="mt-6 p-4 bg-gray-800/50 rounded-lg">
            <div className="flex items-center gap-2 text-blue-400">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-400"></div>
              <span>Running traceroute to {host}...</span>
            </div>
          </div>
        )}

        {results && (
          <div className="mt-6 bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-900">
            <h2 className="text-xl font-semibold text-white mb-4">Traceroute Results</h2>
            <p className="text-gray-400 text-sm mb-4">
              Route to {results.destination}
              {results.ip && results.ip !== results.destination && (
                <span className="text-gray-500 ml-1">({results.ip})</span>
              )}
              {" • "}
              {results.totalHops} hops •{" "}
              {results.complete ? "Complete" : "Incomplete"}
            </p>

            {/* Desktop Table View */}
            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-800">
                    <th className="text-left py-3 px-2 text-gray-400 font-medium">Hop</th>
                    <th className="text-left py-3 px-2 text-gray-400 font-medium">Host</th>
                    <th className="text-left py-3 px-2 text-gray-400 font-medium">Provider</th>
                    <th className="text-right py-3 px-2 text-gray-400 font-medium">RTT 1</th>
                    <th className="text-right py-3 px-2 text-gray-400 font-medium">RTT 2</th>
                    <th className="text-right py-3 px-2 text-gray-400 font-medium">RTT 3</th>
                    <th className="text-right py-3 px-2 text-gray-400 font-medium">Average</th>
                  </tr>
                </thead>
                <tbody>
                  {results.hops.map((hop) => (
                    <motion.tr
                      key={hop.number}
                      initial={{ opacity: 0, y: 20 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.3 }}
                      className="border-b border-gray-800/50 last:border-0 hover:bg-gray-800/30 transition-colors"
                    >
                      <td className="py-3 px-2 text-gray-300">
                        {hop.number}
                      </td>
                      <td className="py-3 px-2 text-gray-300 truncate max-w-[200px]" title={hop.timeout ? "Request timed out" : hop.host}>
                        {hop.timeout ? (
                          <span className="text-red-400">Request timed out</span>
                        ) : (
                          hop.host
                        )}
                      </td>
                      <td className="py-3 px-2 min-w-[200px]">
                        {hop.as ? (
                          <div className="flex items-center gap-2">
                            <CountryFlag
                              countryCode={hop.countryCode}
                              className="w-4 h-3 flex-shrink-0"
                            />
                            <span className="text-blue-400 text-xs truncate" title={hop.as}>
                              {hop.as}
                            </span>
                          </div>
                        ) : (
                          <span className="text-gray-500">—</span>
                        )}
                      </td>
                      <td className="py-3 px-2 text-right text-emerald-400 font-mono">
                        {hop.timeout ? "*" : formatRTT(hop.rtt1)}
                      </td>
                      <td className="py-3 px-2 text-right text-yellow-400 font-mono">
                        {hop.timeout ? "*" : formatRTT(hop.rtt2)}
                      </td>
                      <td className="py-3 px-2 text-right text-orange-400 font-mono">
                        {hop.timeout ? "*" : formatRTT(hop.rtt3)}
                      </td>
                      <td className="py-3 px-2 text-right text-blue-400 font-mono">
                        {hop.timeout ? "*" : formatRTT(getAverageRTT(hop))}
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Mobile Card View */}
            <div className="md:hidden space-y-3">
              {results.hops.map((hop) => (
                <motion.div
                  key={hop.number}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.3 }}
                  className="bg-gray-800/50 rounded-lg p-4 border border-gray-800"
                >
                  <div className="flex items-center justify-between mb-3">
                    <div className="text-gray-300 text-sm font-medium truncate flex-1 mr-2">
                      Hop {hop.number}: {hop.timeout ? "Request timed out" : hop.host}
                    </div>
                  </div>
                  {hop.as && (
                    <div className="flex items-center gap-2 mb-3">
                      <CountryFlag
                        countryCode={hop.countryCode}
                        className="w-4 h-3 flex-shrink-0"
                      />
                      <span className="text-blue-400 text-xs">{hop.as}</span>
                    </div>
                  )}
                  <div className="grid grid-cols-2 gap-3 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-400">RTT 1:</span>
                      <span className="text-emerald-400 font-mono">{hop.timeout ? "*" : formatRTT(hop.rtt1)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-400">RTT 2:</span>
                      <span className="text-yellow-400 font-mono">{hop.timeout ? "*" : formatRTT(hop.rtt2)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-400">RTT 3:</span>
                      <span className="text-orange-400 font-mono">{hop.timeout ? "*" : formatRTT(hop.rtt3)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-400">Average:</span>
                      <span className="text-blue-400 font-mono">{hop.timeout ? "*" : formatRTT(getAverageRTT(hop))}</span>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
          </div>
        )}
      </div>
    </motion.div>
  );
};
