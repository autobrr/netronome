/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { Server } from "@/types/types";
import { Button } from "@/components/ui/button";
import { ServerFilters } from "./components/ServerFilters";
import { ServerCard } from "./components/ServerCard";
import { useServerData } from "./hooks/useServerData";
import { extractHostname } from "./utils/tracerouteUtils";
import { STYLES } from "./constants/tracerouteConstants";

interface TracerouteServerSelectorProps {
  host: string;
  selectedServer: Server | null;
  onHostChange: (host: string) => void;
  onServerSelect: (server: Server | null) => void;
  onRunTraceroute: () => void;
  isRunning: boolean;
}

export const TracerouteServerSelector: React.FC<
  TracerouteServerSelectorProps
> = ({
  host,
  selectedServer,
  onHostChange,
  onServerSelect,
  onRunTraceroute,
  isRunning,
}) => {
  const {
    displayedServers,
    serverTypeOptions,
    searchTerm,
    filterType,
    setSearchTerm,
    setFilterType,
    loadMoreServers,
    hasMoreServers,
  } = useServerData();

  const handleServerSelect = (server: Server) => {
    const isSelected = selectedServer?.id === server.id;
    if (isSelected) {
      // Unselect the server
      onServerSelect(null);
      onHostChange("");
    } else {
      // Select the server
      onServerSelect(server);
      // Extract hostname for all server types
      onHostChange(extractHostname(server.host));
    }
  };

  const getButtonVariant = () => {
    if (isRunning || !host.trim()) {
      return "secondary";
    }
    return "default";
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="flex-1"
    >
      <div className={STYLES.card}>
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
              onHostChange(e.target.value);
              onServerSelect(null);
            }}
            placeholder="Enter hostname/IP (e.g., google.com, 8.8.8.8)"
            className={`flex-1 ${STYLES.input}`}
            disabled={isRunning}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !isRunning) {
                onRunTraceroute();
              }
            }}
          />
          <Button
            onClick={onRunTraceroute}
            disabled={!host.trim() || isRunning}
            isLoading={isRunning}
            variant={getButtonVariant()}
            className="min-w-[100px]"
          >
            {isRunning ? "Running" : "Trace"}
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
        <ServerFilters
          searchTerm={searchTerm}
          filterType={filterType}
          serverTypeOptions={serverTypeOptions}
          onSearchChange={setSearchTerm}
          onFilterTypeChange={setFilterType}
          disabled={isRunning}
        />

        {/* Server Grid */}
        <div className="grid grid-cols-1 gap-4 mb-4">
          {displayedServers.map((server) => (
            <ServerCard
              key={server.id}
              server={server}
              isSelected={selectedServer?.id === server.id}
              onSelect={handleServerSelect}
              disabled={isRunning}
            />
          ))}
        </div>

        {/* Load More Button */}
        {hasMoreServers && (
          <div className="flex justify-center mb-4">
            <button
              onClick={loadMoreServers}
              className="px-4 py-2 bg-gray-200/30 dark:bg-gray-800/30 border border-gray-300/50 dark:border-gray-900/50 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800/50 transition-colors"
            >
              Load More
            </button>
          </div>
        )}
      </div>
    </motion.div>
  );
};
