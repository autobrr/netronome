/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { Switch, Button as HeadlessButton } from "@headlessui/react";
import { BellIcon, TrashIcon } from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";

interface ChannelDetailsProps {
  channel: any;
  onUpdate: any;
  onDelete: () => void;
  onTest: () => void;
  isDeleting: boolean;
  isTesting: boolean;
}

const SLIDE_TRANSITION = {
  duration: 0.3,
} as const;

export const ChannelDetails: React.FC<ChannelDetailsProps> = ({
  channel,
  onUpdate,
  onDelete,
  onTest,
  isDeleting,
  isTesting,
}) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={SLIDE_TRANSITION}
      className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6"
    >
      <div className="flex items-start justify-between mb-6">
        <div>
          <h4 className="text-xl font-semibold text-gray-900 dark:text-white">
            {channel.name}
          </h4>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Channel configuration and settings
          </p>
        </div>
        
        <div className="flex items-center gap-2">
          <HeadlessButton
            onClick={onTest}
            disabled={isTesting}
            className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isTesting ? (
              <motion.div
                animate={{ rotate: 360 }}
                transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                className="w-4 h-4 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full"
              />
            ) : (
              <BellIcon className="w-4 h-4" />
            )}
            Test
          </HeadlessButton>
          <HeadlessButton
            onClick={onDelete}
            disabled={isDeleting}
            className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-red-600 hover:bg-red-700 dark:bg-red-500 dark:hover:bg-red-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <TrashIcon className="w-4 h-4" />
            Delete
          </HeadlessButton>
        </div>
      </div>

      <div className="space-y-4">
        <div>
          <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Service URL
          </label>
          <div className="mt-1 p-3 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg">
            <code className="text-sm text-gray-800 dark:text-gray-200 break-all">
              {channel.url}
            </code>
          </div>
        </div>

        <div className="flex items-center justify-between p-4 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Channel Status
            </label>
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              {channel.enabled ? "Notifications will be sent to this channel" : "Channel is disabled"}
            </p>
          </div>
          <Switch
            checked={channel.enabled}
            onChange={(checked) => {
              onUpdate.mutate({
                id: channel.id,
                name: channel.name,
                url: channel.url,
                enabled: checked,
              });
            }}
            className={cn(
              "relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2",
              channel.enabled ? "bg-blue-600" : "bg-gray-200 dark:bg-gray-700"
            )}
          >
            <span className="sr-only">Enable channel</span>
            <motion.span
              layout
              transition={{
                type: "spring",
                stiffness: 700,
                damping: 30,
              }}
              className={cn(
                "inline-block h-4 w-4 rounded-full bg-white shadow-lg",
                channel.enabled ? "translate-x-6" : "translate-x-1"
              )}
            />
          </Switch>
        </div>
      </div>
    </motion.div>
  );
};