/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, useMemo } from "react";
import { motion } from "motion/react";
import { useQuery } from "@tanstack/react-query";
import { ChevronUpDownIcon } from "@heroicons/react/24/solid";
import { Server, SavedIperfServer } from "@/types/types";
import { getServers } from "@/api/speedtest";
import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
  Transition,
} from "@headlessui/react";
import { getApiUrl } from "@/utils/baseUrl";

interface ServerSelectionProps {
  selectedServer: Server | null;
  onServerSelect: (server: Server | null) => void;
  disabled?: boolean;
  className?: string;
  placeholder?: string;
  showSearch?: boolean;
  showTypeFilter?: boolean;
  displayCount?: number;
}

// Utility function to extract hostname from any server host value
export const extractHostname = (hostValue: string): string => {
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

export const ServerSelection: React.FC<ServerSelectionProps> = ({
  selectedServer,
  onServerSelect,
  disabled = false,
  className = "",
  placeholder = "Search and select server...",
  showSearch = true,
  showTypeFilter = true,
  displayCount: initialDisplayCount = 4,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState("");
  const [displayCount, setDisplayCount] = useState(initialDisplayCount);
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

  const handleServerSelect = (server: Server) => {
    const isSelected = selectedServer?.id === server.id;
    if (isSelected) {
      // Unselect the server
      onServerSelect(null);
    } else {
      // Select the server
      onServerSelect(server);
    }
  };

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Search and Filter Controls */}
      {(showSearch || showTypeFilter) && (
        <div className="flex flex-col md:flex-row gap-4">
          {/* Search Input */}
          {showSearch && (
            <div className="flex-1">
              <input
                type="text"
                placeholder={placeholder}
                className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                disabled={disabled}
              />
            </div>
          )}

          {/* Server Type Filter */}
          {showTypeFilter && (
            <Listbox
              value={filterType}
              onChange={setFilterType}
              disabled={disabled}
            >
              <div className="relative min-w-[160px]">
                <ListboxButton className="relative w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 rounded-lg text-left text-gray-700 dark:text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50 disabled:opacity-50 disabled:cursor-not-allowed">
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
          )}
        </div>
      )}

      {/* Server Grid */}
      <div className="grid grid-cols-1 gap-4">
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
              disabled={disabled}
            >
              <div className="flex flex-col gap-1">
                <span className="text-blue-600 dark:text-blue-300 font-medium truncate">
                  {server.isIperf ? server.name : server.sponsor}
                </span>
                <span className="text-gray-600 dark:text-gray-400 text-sm">
                  {server.isIperf ? "iperf3 Server" : server.name}
                  <span className="block truncate text-xs" title={server.host}>
                    {extractHostname(server.host)}
                  </span>
                </span>
                <span className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                  {server.isIperf
                    ? "Custom Server"
                    : `${server.country} - ${Math.floor(server.distance)} km`}
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
        <div className="flex justify-center">
          <button
            onClick={() => setDisplayCount((prev) => prev + 4)}
            className="px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300/80 dark:border-gray-800/80 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-800 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={disabled}
          >
            Load More
          </button>
        </div>
      )}
    </div>
  );
};
