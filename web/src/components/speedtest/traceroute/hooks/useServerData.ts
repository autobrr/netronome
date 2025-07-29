/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Server, SavedIperfServer } from "@/types/types";
import { getServers } from "@/api/speedtest";
import { getApiUrl } from "@/utils/baseUrl";
import {
  combineServers,
  getFilteredAndSortedServers,
  SERVER_TYPE_OPTIONS,
} from "../utils/serverUtils";
import { DEFAULT_SERVER_DISPLAY_COUNT } from "../constants/tracerouteConstants";

export const useServerData = () => {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState("all");
  const [displayCount, setDisplayCount] = useState(
    DEFAULT_SERVER_DISPLAY_COUNT,
  );
  const [iperfServers, setIperfServers] = useState<SavedIperfServer[]>([]);

  // Fetch speedtest servers
  const { data: speedtestServers = [] } = useQuery({
    queryKey: ["servers", "speedtest"],
    queryFn: () => getServers("speedtest"),
  }) as { data: Server[] };

  // Fetch librespeed servers
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

  // Combine all servers
  const allServers = useMemo(() => {
    return combineServers(speedtestServers, librespeedServers, iperfServers);
  }, [speedtestServers, librespeedServers, iperfServers]);

  // Filter and sort servers
  const filteredServers = useMemo(() => {
    return getFilteredAndSortedServers(allServers, searchTerm, filterType);
  }, [allServers, searchTerm, filterType]);

  // Server management functions
  const loadMoreServers = () => {
    setDisplayCount((prev) => prev + DEFAULT_SERVER_DISPLAY_COUNT);
  };

  const resetDisplayCount = () => {
    setDisplayCount(DEFAULT_SERVER_DISPLAY_COUNT);
  };

  return {
    // Data
    allServers,
    filteredServers,
    displayedServers: filteredServers.slice(0, displayCount),
    serverTypeOptions: SERVER_TYPE_OPTIONS,

    // State
    searchTerm,
    filterType,
    displayCount,

    // Actions
    setSearchTerm,
    setFilterType,
    loadMoreServers,
    resetDisplayCount,

    // Computed
    hasMoreServers: filteredServers.length > displayCount,
    totalServersCount: allServers.length,
    filteredServersCount: filteredServers.length,
  };
};
