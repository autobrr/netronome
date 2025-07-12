/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { Server } from "@/types/types";
import { extractHostname } from "../utils/tracerouteUtils";
import {
  getServerTypeLabel,
  getServerTypeColorClass,
} from "../utils/serverUtils";

interface ServerCardProps {
  server: Server;
  isSelected: boolean;
  onSelect: (server: Server) => void;
  disabled?: boolean;
}

export const ServerCard: React.FC<ServerCardProps> = ({
  server,
  isSelected,
  onSelect,
  disabled = false,
}) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <button
        onClick={() => onSelect(server)}
        className={`w-full p-3 rounded-lg text-left transition-colors ${
          isSelected
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
            <span className={`ml-2 ${getServerTypeColorClass(server)}`}>
              {getServerTypeLabel(server)}
            </span>
          </span>
        </div>
      </button>
    </motion.div>
  );
};
