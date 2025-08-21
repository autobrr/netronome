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
  const [comprehensiveServers, setComprehensiveServers] = useState<Server[]>([]);

  // Fetch speedtest servers (local)
  const { data: speedtestServers = [] } = useQuery({
    queryKey: ["servers", "speedtest"],
    queryFn: () => getServers("speedtest"),
  }) as { data: Server[] };

  // Fetch librespeed servers
  const { data: librespeedServers = [] } = useQuery({
    queryKey: ["servers", "librespeed"],
    queryFn: () => getServers("librespeed"),
  }) as { data: Server[] };

  // Fetch comprehensive speedtest.net servers from localStorage cache first, then fallback to API
  useEffect(() => {
    const COMPREHENSIVE_SERVERS_CACHE_KEY = "netronome-comprehensive-servers";
    
    const loadSpeedtestServers = () => {
      try {
        // Check localStorage cache first
        const cachedData = localStorage.getItem(COMPREHENSIVE_SERVERS_CACHE_KEY);
        if (cachedData) {
          const parsedData = JSON.parse(cachedData);
          if (parsedData.data && parsedData.data.allServers && Array.isArray(parsedData.data.allServers)) {
            setComprehensiveServers(parsedData.data.allServers);
            return; // Use cached data, don't fetch from API
          }
        }
        
        // No valid cache found, servers will be empty
        // This will fallback to the existing speedtest server handling
        setComprehensiveServers([]);
      } catch (error) {
        console.error("Failed to load speedtest servers from cache:", error);
        setComprehensiveServers([]);
      }
    };
    
    // Load servers on mount
    loadSpeedtestServers();
    
    // Listen for localStorage changes (when comprehensive servers are fetched in settings)
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === COMPREHENSIVE_SERVERS_CACHE_KEY) {
        loadSpeedtestServers();
      }
    };
    
    window.addEventListener('storage', handleStorageChange);
    
    // Also listen for custom events from same window (since storage events don't fire in same window)
    const handleCustomStorageChange = () => {
      loadSpeedtestServers();
    };
    
    window.addEventListener('netronome-comprehensive-servers-updated', handleCustomStorageChange);
    
    return () => {
      window.removeEventListener('storage', handleStorageChange);
      window.removeEventListener('netronome-comprehensive-servers-updated', handleCustomStorageChange);
    };
  }, []);

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
    // Use comprehensive servers if available, otherwise fall back to local speedtest servers
    const speedtestServerList = comprehensiveServers.length > 0 
      ? comprehensiveServers 
      : speedtestServers;
        
    return combineServers(speedtestServerList, librespeedServers, iperfServers);
  }, [speedtestServers, librespeedServers, iperfServers, comprehensiveServers]);

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
