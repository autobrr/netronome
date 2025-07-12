/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Server, SavedIperfServer } from "@/types/types";

/**
 * Server type options for filtering
 */
export const SERVER_TYPE_OPTIONS = [
  { value: "", label: "All Types" },
  { value: "speedtest", label: "Speedtest.net" },
  { value: "iperf3", label: "iperf3" },
  { value: "librespeed", label: "LibreSpeed" },
];

/**
 * Convert iperf servers to Server format
 */
export const convertIperfServersToServerFormat = (
  iperfServers: SavedIperfServer[],
): Server[] => {
  return iperfServers.map((server) => ({
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
};

/**
 * Combine all server types into a single array
 */
export const combineServers = (
  speedtestServers: Server[],
  librespeedServers: Server[],
  iperfServers: SavedIperfServer[],
): Server[] => {
  const iperfServerList = convertIperfServersToServerFormat(iperfServers);
  return [...speedtestServers, ...librespeedServers, ...iperfServerList];
};

/**
 * Filter servers based on search term and server type
 */
export const filterServers = (
  servers: Server[],
  searchTerm: string,
  filterType: string,
): Server[] => {
  return servers.filter((server) => {
    const matchesSearch =
      searchTerm === "" ||
      server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.sponsor.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.country.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.host.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesType =
      filterType === "" ||
      (filterType === "speedtest" && !server.isIperf && !server.isLibrespeed) ||
      (filterType === "iperf3" && server.isIperf) ||
      (filterType === "librespeed" && server.isLibrespeed);

    return matchesSearch && matchesType;
  });
};

/**
 * Sort servers - iperf servers by name, others by distance
 */
export const sortServers = (servers: Server[]): Server[] => {
  return servers.sort((a, b) => {
    // Sort iperf servers by name, others by distance
    if (a.isIperf && b.isIperf) {
      return a.name.localeCompare(b.name);
    }
    return a.distance - b.distance;
  });
};

/**
 * Get filtered and sorted servers
 */
export const getFilteredAndSortedServers = (
  servers: Server[],
  searchTerm: string,
  filterType: string,
): Server[] => {
  const filtered = filterServers(servers, searchTerm, filterType);
  return sortServers(filtered);
};

/**
 * Get server type label for display
 */
export const getServerTypeLabel = (server: Server): string => {
  if (server.isIperf) return "iperf3";
  if (server.isLibrespeed) return "librespeed";
  return "speedtest.net";
};

/**
 * Get server type color classes
 */
export const getServerTypeColorClass = (server: Server): string => {
  if (server.isIperf) {
    return "text-purple-600 dark:text-purple-400 drop-shadow-[0_0_1px_rgba(168,85,247,0.8)]";
  }
  if (server.isLibrespeed) {
    return "text-blue-600 dark:text-blue-400 drop-shadow-[0_0_1px_rgba(96,165,250,0.8)]";
  }
  return "text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_1px_rgba(251,191,36,0.8)]";
};
