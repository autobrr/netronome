/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  ServerIcon,
  CpuChipIcon,
  ClockIcon,
} from "@heroicons/react/24/outline";
import { SystemInfo, InterfaceInfo } from "@/api/vnstat";
import { formatBytes } from "@/utils/formatBytes";

interface VnstatSystemInfoProps {
  systemInfo: SystemInfo;
  refreshInterval?: number; // in milliseconds
}

export const VnstatSystemInfo: React.FC<VnstatSystemInfoProps> = ({
  systemInfo,
  refreshInterval = 300000, // default 5 minutes
}) => {
  const formatUptime = (seconds: number): string => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    const parts = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);

    return parts.join(" ") || "0m";
  };

  const formatRefreshInterval = (ms: number): string => {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours} hour${hours > 1 ? 's' : ''}`;
    } else if (minutes > 0) {
      return `${minutes} minute${minutes > 1 ? 's' : ''}`;
    } else {
      return `${seconds} second${seconds > 1 ? 's' : ''}`;
    }
  };

  const activeInterfaces = Object.values(systemInfo.interfaces).filter(
    (iface) =>
      iface.is_up &&
      !iface.name.startsWith("veth") && // Exclude Docker container interfaces
      !iface.name.startsWith("br-") && // Exclude Docker bridge interfaces
      iface.name !== "docker0" // Exclude default Docker bridge
  );

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
    >
      <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
        System Information
      </h3>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {/* Hostname */}
        <div className="flex items-start space-x-3">
          <div className="rounded-full bg-blue-100 p-2 dark:bg-blue-900/20 flex-shrink-0">
            <ServerIcon className="h-5 w-5 text-blue-600 dark:text-blue-400" />
          </div>
          <div>
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400">
              Hostname
            </p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {systemInfo.hostname}
            </p>
          </div>
        </div>

        {/* Kernel */}
        <div className="flex items-start space-x-3">
          <div className="rounded-full bg-green-100 p-2 dark:bg-green-900/20 flex-shrink-0">
            <CpuChipIcon className="h-5 w-5 text-green-600 dark:text-green-400" />
          </div>
          <div>
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400">
              Kernel
            </p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {systemInfo.kernel}
            </p>
          </div>
        </div>

        {/* Uptime */}
        <div className="flex items-start space-x-3">
          <div className="rounded-full bg-purple-100 p-2 dark:bg-purple-900/20 flex-shrink-0">
            <ClockIcon className="h-5 w-5 text-purple-600 dark:text-purple-400" />
          </div>
          <div>
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400">
              Uptime
            </p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {formatUptime(systemInfo.uptime)}
            </p>
          </div>
        </div>
      </div>

      {/* Network Interfaces */}
      {activeInterfaces.length > 0 && (
        <div className="mt-6 pt-6 border-t border-gray-200 dark:border-gray-800">
          <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
            Network Interfaces
          </h4>
          <div className="space-y-2">
            {activeInterfaces.map((iface) => (
              <InterfaceCard key={iface.name} interface={iface} />
            ))}
          </div>
        </div>
      )}

      {/* Agent Health Summary */}
      <div className="mt-6 pt-6 border-t border-gray-200 dark:border-gray-800">
        <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
          Agent Health
        </h4>
        <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <div>
              <p className="text-xs text-gray-600 dark:text-gray-400">Status</p>
              <p className="text-sm font-medium text-green-600 dark:text-green-400">
                Healthy
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                Active Since
              </p>
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                {formatUptime(systemInfo.uptime)}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                Last Sync
              </p>
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                {new Date(systemInfo.updated_at).toLocaleTimeString()}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* vnstat Version */}
      <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-800">
        <p className="text-xs text-gray-500 dark:text-gray-400 flex items-center justify-between">
          <span>{systemInfo.vnstat_version}</span>
          <span>Refresh interval: {formatRefreshInterval(refreshInterval)}</span>
        </p>
      </div>
    </motion.div>
  );
};

interface InterfaceCardProps {
  interface: InterfaceInfo;
}

const InterfaceCard: React.FC<InterfaceCardProps> = ({ interface: iface }) => {
  return (
    <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3">
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-gray-900 dark:text-white">
              {iface.alias || iface.name}
            </span>
            {iface.alias && (
              <span className="text-xs text-gray-500 dark:text-gray-400">
                ({iface.name})
              </span>
            )}
            {iface.link_speed > 0 && (
              <span className="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded-full">
                {iface.link_speed} Mbps
              </span>
            )}
          </div>
          <div className="mt-1 flex items-center space-x-3">
            {iface.ip_address && (
              <span className="text-xs text-gray-600 dark:text-gray-400">
                IP: {iface.ip_address}
              </span>
            )}
            {iface.bytes_total > 0 && (
              <span className="text-xs text-gray-600 dark:text-gray-400">
                Total: {formatBytes(iface.bytes_total)}
              </span>
            )}
          </div>
        </div>
        <div className="flex-shrink-0">
          <span
            className={`inline-flex h-2 w-2 rounded-full ${
              iface.is_up ? "bg-green-500" : "bg-red-500"
            }`}
          />
        </div>
      </div>
    </div>
  );
};
