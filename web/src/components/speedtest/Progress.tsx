/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";

interface ProgressProps {
  progress: {
    percentage?: number;
    server?: string;
  };
}

export const Progress: React.FC<ProgressProps> = ({ progress }) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      className="bg-gray-850/95 rounded-xl p-4 shadow-lg border border-gray-900"
    >
      <div className="flex items-center justify-between mb-2">
        <span className="text-white font-medium">Test Progress</span>
        <span className="text-gray-400 text-sm">
          {progress.percentage ? `${progress.percentage}%` : "Running..."}
        </span>
      </div>
      <div className="w-full bg-gray-800 rounded-full h-2">
        <div
          className="bg-blue-500 h-2 rounded-full transition-all duration-300"
          style={{ width: `${progress.percentage || 0}%` }}
        />
      </div>
      {progress.server && (
        <div className="mt-2 text-sm text-gray-400">
          Server: {progress.server}
        </div>
      )}
    </motion.div>
  );
};