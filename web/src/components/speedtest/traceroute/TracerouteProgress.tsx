/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { TracerouteUpdate } from "@/types/types";
import { STYLES } from "./constants/tracerouteConstants";

interface TracerouteProgressProps {
  tracerouteStatus: TracerouteUpdate;
}

export const TracerouteProgress: React.FC<TracerouteProgressProps> = ({
  tracerouteStatus,
}) => {
  if (tracerouteStatus.isComplete) {
    return null;
  }

  return (
    <div className={STYLES.progressContainer}>
      <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 mb-3">
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 dark:border-blue-400"></div>
        <span>Running traceroute to {tracerouteStatus.host}...</span>
      </div>

      <div className="w-full bg-gray-300 dark:bg-gray-700 rounded-full h-2">
        <div
          className="bg-blue-500 h-2 rounded-full transition-all duration-500"
          style={{
            width: `${Math.min(tracerouteStatus.progress, 100)}%`,
          }}
        ></div>
      </div>

      <div className="text-sm text-blue-600 dark:text-blue-300 mt-2">
        Hop {tracerouteStatus.currentHop} of {tracerouteStatus.totalHops} (
        {Math.round(tracerouteStatus.progress)}%)
      </div>
    </div>
  );
};
