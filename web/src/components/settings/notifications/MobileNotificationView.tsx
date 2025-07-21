/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Switch, Button as HeadlessButton } from "@headlessui/react";
import { BellIcon, PlusIcon } from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import { EventCategorySection } from "./EventCategorySection";
import { AddChannelForm } from "./AddChannelForm";
import type { NotificationEvent, NotificationRule } from "@/api/notifications";

interface MobileNotificationViewProps {
  channels: any[];
  activeChannel: any;
  activeChannelId: number | null;
  setActiveChannelId: (id: number | null) => void;
  showAddChannel: boolean;
  setShowAddChannel: (show: boolean) => void;
  hasUnsavedChanges: boolean;
  pendingChanges: Map<number, any>;
  rules: any[];
  rulesLoading: boolean;
  eventsByCategory: Record<string, NotificationEvent[]>;
  getRuleState: (eventId: number) => Partial<NotificationRule>;
  updatePendingChange: (eventId: number, update: any) => void;
  saveChanges: () => Promise<void>;
  cancelChanges: () => void;
  createChannelMutation: any;
  updateChannelMutation: any;
  testChannelMutation: any;
  testingChannelId: number | null;
  setTestingChannelId: (id: number | null) => void;
  updateRuleMutation: any;
  createRuleMutation: any;
}

const FADE_TRANSITION = {
  duration: 0.2,
} as const;

const SLIDE_TRANSITION = {
  duration: 0.3,
} as const;

export const MobileNotificationView: React.FC<MobileNotificationViewProps> = ({
  channels,
  activeChannel,
  activeChannelId,
  setActiveChannelId,
  showAddChannel,
  setShowAddChannel,
  hasUnsavedChanges,
  pendingChanges,
  rulesLoading,
  eventsByCategory,
  getRuleState,
  updatePendingChange,
  saveChanges,
  cancelChanges,
  createChannelMutation,
  updateChannelMutation,
  testChannelMutation,
  testingChannelId,
  setTestingChannelId,
  updateRuleMutation,
  createRuleMutation,
}) => {
  const [activeTab, setActiveTab] = useState<"channels" | "rules">("channels");

  return (
    <div className="space-y-4">
      {/* Tab Navigation */}
      <div className="flex bg-gray-200/50 dark:bg-gray-800/50 rounded-lg p-1">
        <button
          onClick={() => setActiveTab("channels")}
          className={cn(
            "flex-1 py-2 px-4 rounded-md font-medium text-sm transition-all",
            activeTab === "channels"
              ? "bg-white dark:bg-gray-800 text-gray-900 dark:text-white shadow-sm"
              : "text-gray-600 dark:text-gray-400"
          )}
        >
          Channels
        </button>
        <button
          onClick={() => setActiveTab("rules")}
          className={cn(
            "flex-1 py-2 px-4 rounded-md font-medium text-sm transition-all relative",
            activeTab === "rules"
              ? "bg-white dark:bg-gray-800 text-gray-900 dark:text-white shadow-sm"
              : "text-gray-600 dark:text-gray-400"
          )}
          disabled={!activeChannel}
        >
          Rules
          {hasUnsavedChanges && (
            <div className="absolute top-1 right-1 w-2 h-2 bg-amber-500 rounded-full" />
          )}
        </button>
      </div>

      {/* Tab Content */}
      <AnimatePresence mode="wait">
        {activeTab === "channels" ? (
          <motion.div
            key="channels"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={FADE_TRANSITION}
            className="space-y-4"
          >
            {/* Add Channel Button */}
            <HeadlessButton
              onClick={() => setShowAddChannel(true)}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors"
            >
              <PlusIcon className="w-4 h-4" />
              Add New Channel
            </HeadlessButton>

            {/* Add Channel Form */}
            <AnimatePresence>
              {showAddChannel && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={SLIDE_TRANSITION}
                  className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-4"
                >
                  <h5 className="font-medium text-gray-900 dark:text-white mb-4">
                    Add New Channel
                  </h5>
                  <AddChannelForm
                    onSubmit={(input) => {
                      createChannelMutation.mutate(input);
                    }}
                    onCancel={() => setShowAddChannel(false)}
                    isLoading={createChannelMutation.isPending}
                  />
                </motion.div>
              )}
            </AnimatePresence>

            {/* Channels List */}
            <div className="space-y-3">
              {channels.map((channel) => (
                <motion.div
                  key={channel.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-4"
                >
                  <div
                    onClick={() => {
                      setActiveChannelId(channel.id);
                      setActiveTab("rules");
                    }}
                    className="cursor-pointer"
                  >
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <div className={cn(
                          "w-2 h-2 rounded-full",
                          channel.enabled ? "bg-emerald-500" : "bg-gray-400"
                        )} />
                        <h5 className="font-medium text-gray-900 dark:text-white">
                          {channel.name}
                        </h5>
                      </div>
                      {activeChannelId === channel.id && (
                        <div className="px-2 py-1 bg-blue-500/10 text-blue-600 dark:text-blue-400 text-xs font-medium rounded-full">
                          Active
                        </div>
                      )}
                    </div>
                    <p className="text-sm text-gray-600 dark:text-gray-400 truncate">
                      {channel.url}
                    </p>
                  </div>

                  <div className="flex items-center gap-2 mt-3 pt-3 border-t border-gray-200 dark:border-gray-800">
                    <HeadlessButton
                      onClick={() => {
                        setTestingChannelId(channel.id);
                        testChannelMutation.mutate(channel.id);
                      }}
                      disabled={testingChannelId === channel.id && testChannelMutation.isPending}
                      className="flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {testingChannelId === channel.id && testChannelMutation.isPending ? (
                        <motion.div
                          animate={{ rotate: 360 }}
                          transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                          className="w-4 h-4 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full"
                        />
                      ) : (
                        <>
                          <BellIcon className="w-4 h-4" />
                          Test
                        </>
                      )}
                    </HeadlessButton>
                    <Switch
                      checked={channel.enabled}
                      onChange={(checked) => {
                        updateChannelMutation.mutate({
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
                      <span className="sr-only">Enable {channel.name}</span>
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
                </motion.div>
              ))}

              {channels.length === 0 && (
                <div className="text-center py-12">
                  <BellIcon className="w-12 h-12 text-gray-400 mx-auto mb-3" />
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    No channels configured
                  </p>
                </div>
              )}
            </div>
          </motion.div>
        ) : (
          <motion.div
            key="rules"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={FADE_TRANSITION}
            className="space-y-4"
          >
            {activeChannel && (
              <>
                {/* Active Channel Info */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h5 className="font-medium text-gray-900 dark:text-white">
                        {activeChannel.name}
                      </h5>
                      <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                        Configure notification rules
                      </p>
                    </div>
                    <button
                      onClick={() => setActiveTab("channels")}
                      className="text-sm text-blue-600 dark:text-blue-400"
                    >
                      Change
                    </button>
                  </div>
                </div>

                {/* Save/Cancel Buttons */}
                <AnimatePresence>
                  {hasUnsavedChanges && (
                    <motion.div
                      initial={{ opacity: 0, y: -10 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -10 }}
                      transition={FADE_TRANSITION}
                      className="flex items-center gap-2"
                    >
                      <HeadlessButton
                        onClick={cancelChanges}
                        className="flex-1 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors"
                      >
                        Cancel
                      </HeadlessButton>
                      <HeadlessButton
                        onClick={saveChanges}
                        disabled={updateRuleMutation.isPending || createRuleMutation.isPending}
                        className="flex-1 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Save Changes
                      </HeadlessButton>
                    </motion.div>
                  )}
                </AnimatePresence>

                {/* Rules */}
                {rulesLoading ? (
                  <div className="text-center py-8">
                    <motion.div
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                      className="w-6 h-6 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full mx-auto mb-3"
                    />
                    <p className="text-sm text-gray-600 dark:text-gray-400">Loading rules...</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {Object.entries(eventsByCategory).map(([category, categoryEvents]) => (
                      <EventCategorySection
                        key={category}
                        category={category}
                        events={categoryEvents}
                        pendingChanges={pendingChanges}
                        getRuleState={getRuleState}
                        onUpdateRule={(eventId, input) => {
                          updatePendingChange(eventId, input);
                        }}
                      />
                    ))}
                  </div>
                )}
              </>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};