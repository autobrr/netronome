/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { Server, SavedIperfServer } from "@/types/types";

/** Server provider options shown by the traceroute selector. */
export const SERVER_TYPE_OPTIONS = [
  { value: "all", label: "All Types" },
  { value: "speedtest", label: "Speedtest.net" },
  { value: "iperf3", label: "iperf3" },
  { value: "librespeed", label: "LibreSpeed" },
];

/** Converts saved iperf3 endpoints to the shared server representation. */
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

/** Combines every provider's selectable servers into one list. */
export const combineServers = (
  speedtestServers: Server[],
  librespeedServers: Server[],
  iperfServers: SavedIperfServer[],
): Server[] => {
  const iperfServerList = convertIperfServersToServerFormat(iperfServers);
  return [...speedtestServers, ...librespeedServers, ...iperfServerList];
};

/** Keeps traceroute selections scoped to their owning server catalogue. */
export const getTracerouteServerSelectionKey = (
  server: Pick<Server, "isIperf" | "isLibrespeed"> | null,
  speedtestSelectionKey: string,
): string => {
  if (server?.isIperf) return "iperf";
  if (server?.isLibrespeed) return "librespeed";
  return speedtestSelectionKey;
};

/** Filters servers by provider and a case-insensitive display-field search. */
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
      filterType === "all" ||
      (filterType === "speedtest" && !server.isIperf && !server.isLibrespeed) ||
      (filterType === "iperf3" && server.isIperf) ||
      (filterType === "librespeed" && server.isLibrespeed);

    return matchesSearch && matchesType;
  });
};

/** Sorts iperf3 endpoints by name and all other entries by distance. */
export const sortServers = (servers: Server[]): Server[] => {
  return servers.sort((a, b) => {
    // Sort iperf servers by name, others by distance
    if (a.isIperf && b.isIperf) {
      return a.name.localeCompare(b.name);
    }
    return a.distance - b.distance;
  });
};

/** Applies traceroute server filtering and provider-specific ordering. */
export const getFilteredAndSortedServers = (
  servers: Server[],
  searchTerm: string,
  filterType: string,
): Server[] => {
  const filtered = filterServers(servers, searchTerm, filterType);
  return sortServers(filtered);
};

/** Returns the provider label displayed for a selectable server. */
export const getServerTypeLabel = (server: Server): string => {
  if (server.isIperf) return "iperf3";
  if (server.isLibrespeed) return "librespeed";
  return "speedtest.net";
};

/** Returns the provider-specific text classes used by the selector. */
export const getServerTypeColorClass = (server: Server): string => {
  if (server.isIperf) {
    return "text-purple-600 dark:text-purple-400 drop-shadow-[0_0_1px_var(--color-purple-500)]";
  }
  if (server.isLibrespeed) {
    return "text-blue-600 dark:text-blue-400 drop-shadow-[0_0_1px_var(--color-blue-400)]";
  }
  return "text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_1px_var(--color-amber-400)]";
};
