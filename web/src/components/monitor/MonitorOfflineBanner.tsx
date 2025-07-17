/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { ServerIcon } from "@heroicons/react/24/outline";

interface MonitorOfflineBannerProps {
  message?: string;
}

export const MonitorOfflineBanner: React.FC<MonitorOfflineBannerProps> = ({
  message = "Showing cached data. Real-time metrics unavailable.",
}) => {
  return (
    <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4">
      <div className="flex items-center space-x-3">
        <ServerIcon className="h-5 w-5 text-amber-500" />
        <div>
          <p className="text-sm font-medium text-amber-600 dark:text-amber-400">
            Agent Offline
          </p>
          <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
            {message}
          </p>
        </div>
      </div>
    </div>
  );
};