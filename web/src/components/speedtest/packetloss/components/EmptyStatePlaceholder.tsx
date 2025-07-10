/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  ChartBarIcon,
  WifiIcon,
  ExclamationTriangleIcon,
  ClockIcon,
  SignalIcon,
} from "@heroicons/react/24/outline";

export const EmptyStatePlaceholder: React.FC = () => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.1 }}
      className="flex-1"
    >
      <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
        <div className="text-center mb-6">
          <div className="w-16 h-16 bg-blue-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
            <ChartBarIcon className="w-8 h-8 text-blue-600 dark:text-blue-400" />
          </div>
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
            Network Monitoring Dashboard
          </h3>
          <p className="text-gray-600 dark:text-gray-400 text-sm">
            Select a monitor to view packet loss trends and network performance
            data
          </p>
        </div>

        <div className="space-y-4 mb-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
              <div className="w-8 h-8 bg-emerald-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                <WifiIcon className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                  MTR & ICMP Testing
                </h4>
                <p className="text-gray-600 dark:text-gray-400 text-xs">
                  Advanced network diagnostics with automatic method selection
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
              <div className="w-8 h-8 bg-red-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                <ExclamationTriangleIcon className="w-4 h-4 text-red-600 dark:text-red-400" />
              </div>
              <div>
                <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                  Smart Alerts
                </h4>
                <p className="text-gray-600 dark:text-gray-400 text-xs">
                  Threshold-based notifications for network issues
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
              <div className="w-8 h-8 bg-blue-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                <ChartBarIcon className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                  Performance Trends
                </h4>
                <p className="text-gray-600 dark:text-gray-400 text-xs">
                  Track network health over time
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 p-3 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
              <div className="w-8 h-8 bg-purple-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
                <ClockIcon className="w-4 h-4 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <h4 className="text-gray-900 dark:text-white text-sm font-medium mb-1">
                  Custom Intervals
                </h4>
                <p className="text-gray-600 dark:text-gray-400 text-xs">
                  Set your own monitoring frequency
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-lg">
          <div className="flex items-start gap-3">
            <SignalIcon className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
            <div>
              <h4 className="text-blue-700 dark:text-blue-300 text-sm font-medium mb-1">
                Get Started
              </h4>
              <p className="text-blue-700/80 dark:text-blue-300/80 text-xs">
                Select a monitor to view detailed metrics and performance
                charts.
              </p>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
};
