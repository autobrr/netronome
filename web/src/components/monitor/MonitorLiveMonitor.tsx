/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { ArrowDownIcon, ArrowUpIcon } from "@heroicons/react/24/outline";
import { MonitorStatus } from "@/api/monitor";

interface MonitorLiveMonitorProps {
  liveData: MonitorStatus["liveData"];
  thresholds?: {
    download?: number; // bytes/s
    upload?: number; // bytes/s
  };
}

export const MonitorLiveMonitor: React.FC<MonitorLiveMonitorProps> = ({
  liveData,
  thresholds,
}) => {
  if (!liveData) return null;

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B/s";
    const k = 1024;
    const sizes = ["B/s", "KiB/s", "MiB/s", "GiB/s"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  };

  // Check if thresholds are exceeded
  const downloadExceeded = thresholds?.download && liveData.rx.bytespersecond > thresholds.download;
  const uploadExceeded = thresholds?.upload && liveData.tx.bytespersecond > thresholds.upload;

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="grid gap-4 sm:grid-cols-2"
    >
      {/* Download Card */}
      <div className={`bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border ${
        downloadExceeded 
          ? 'border-red-500 dark:border-red-500' 
          : 'border-gray-200 dark:border-gray-800'
      } relative overflow-hidden`}>
        {/* Threshold indicator bar */}
        {downloadExceeded && (
          <div className="absolute top-0 left-0 right-0 h-1 bg-red-500 animate-pulse" />
        )}
        
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className={`rounded-full p-3 ${
              downloadExceeded
                ? 'bg-red-100 dark:bg-red-900/20'
                : 'bg-blue-100 dark:bg-blue-900/20'
            }`}>
              <ArrowDownIcon className={`h-6 w-6 ${
                downloadExceeded
                  ? 'text-red-600 dark:text-red-400'
                  : 'text-blue-600 dark:text-blue-400'
              }`} />
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
            {thresholds?.download && (
              <p className={`text-xs mt-1 ${
                downloadExceeded
                  ? 'text-red-600 dark:text-red-400 font-medium'
                  : 'text-gray-500 dark:text-gray-400'
              }`}>
                Limit: {formatBytes(thresholds.download)}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Upload Card */}
      <div className={`bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border ${
        uploadExceeded 
          ? 'border-red-500 dark:border-red-500' 
          : 'border-gray-200 dark:border-gray-800'
      } relative overflow-hidden`}>
        {/* Threshold indicator bar */}
        {uploadExceeded && (
          <div className="absolute top-0 left-0 right-0 h-1 bg-red-500 animate-pulse" />
        )}
        
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className={`rounded-full p-3 ${
              uploadExceeded
                ? 'bg-red-100 dark:bg-red-900/20'
                : 'bg-green-100 dark:bg-green-900/20'
            }`}>
              <ArrowUpIcon className={`h-6 w-6 ${
                uploadExceeded
                  ? 'text-red-600 dark:text-red-400'
                  : 'text-green-600 dark:text-green-400'
              }`} />
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
            {thresholds?.upload && (
              <p className={`text-xs mt-1 ${
                uploadExceeded
                  ? 'text-red-600 dark:text-red-400 font-medium'
                  : 'text-gray-500 dark:text-gray-400'
              }`}>
                Limit: {formatBytes(thresholds.upload)}
              </p>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  );
};
