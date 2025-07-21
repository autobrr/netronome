/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo } from "react";
import { motion } from "motion/react";
import { ChevronRightIcon } from "@heroicons/react/24/outline";
import { SHOUTRRR_SERVICES } from "@/api/notifications";
import { cn } from "@/lib/utils";

interface ChannelCardProps {
  channel: any;
  isActive: boolean;
  hasUnsavedChanges: boolean;
  onClick: () => void;
}

export const ChannelCard: React.FC<ChannelCardProps> = ({ channel, isActive, onClick }) => {
  const serviceType = useMemo(() => {
    if (!channel.url) return null;
    const match = channel.url.match(/^(\w+):\/\//);
    return match ? match[1] : null;
  }, [channel.url]);

  const serviceInfo = SHOUTRRR_SERVICES.find(s => s.value === serviceType);

  return (
    <motion.button
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      onClick={onClick}
      className={cn(
        "w-full p-4 rounded-lg border transition-all text-left",
        isActive
          ? "bg-blue-500/10 border-blue-500/30 shadow-md"
          : "bg-white dark:bg-gray-800/50 border-gray-200 dark:border-gray-800 hover:border-gray-300 dark:hover:border-gray-700 hover:shadow-md"
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <div className={cn(
              "w-2 h-2 rounded-full flex-shrink-0",
              channel.enabled ? "bg-emerald-500" : "bg-gray-400"
            )} />
            <h5 className={cn(
              "font-medium truncate",
              isActive ? "text-blue-700 dark:text-blue-300" : "text-gray-900 dark:text-white"
            )}>
              {channel.name}
            </h5>
          </div>
          <p className="text-xs text-gray-600 dark:text-gray-400 truncate">
            {serviceInfo?.label || serviceType || "Unknown service"}
          </p>
        </div>
        <ChevronRightIcon className={cn(
          "w-4 h-4 flex-shrink-0 transition-colors",
          isActive ? "text-blue-600 dark:text-blue-400" : "text-gray-400"
        )} />
      </div>
    </motion.button>
  );
};