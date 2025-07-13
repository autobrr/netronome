/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { ArrowDownIcon, ArrowUpIcon } from "@heroicons/react/24/outline";
import { VnstatStatus } from "@/api/vnstat";

interface VnstatLiveMonitorProps {
  liveData: VnstatStatus["liveData"];
}

export const VnstatLiveMonitor: React.FC<VnstatLiveMonitorProps> = ({
  liveData,
}) => {
  if (!liveData) return null;

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B/s";
    const k = 1024;
    const sizes = ["B/s", "KB/s", "MB/s", "GB/s"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="grid gap-4 sm:grid-cols-2"
    >
      {/* Download Card */}
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="rounded-full bg-blue-100 p-3 dark:bg-blue-900/20">
              <ArrowDownIcon className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
                Download
              </p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {liveData.rx.ratestring}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {formatBytes(liveData.rx.bytespersecond)}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {liveData.rx.packetspersecond.toLocaleString()} pps
            </p>
          </div>
        </div>
      </div>

      {/* Upload Card */}
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="rounded-full bg-green-100 p-3 dark:bg-green-900/20">
              <ArrowUpIcon className="h-6 w-6 text-green-600 dark:text-green-400" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
                Upload
              </p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {liveData.tx.ratestring}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {formatBytes(liveData.tx.bytespersecond)}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {liveData.tx.packetspersecond.toLocaleString()} pps
            </p>
          </div>
        </div>
      </div>
    </motion.div>
  );
};
