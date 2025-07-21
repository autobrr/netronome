/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button as HeadlessButton } from "@headlessui/react";
import {
  BellIcon,
  PlusIcon,
  CheckIcon,
  XMarkIcon,
  ExclamationTriangleIcon,
  RocketLaunchIcon,
  SignalIcon,
  ComputerDesktopIcon,
} from "@heroicons/react/24/outline";
import {
  notificationsApi,
  type NotificationEvent,
  type NotificationRule,
  type NotificationRuleInput,
} from "@/api/notifications";
import { showToast } from "@/components/common/Toast";
import {
  AddChannelForm,
  ChannelCard,
  ChannelDetails,
  EventCategorySection,
  MobileNotificationView,
} from "./notifications";

const FADE_TRANSITION = {
  duration: 0.2,
} as const;

const SLIDE_TRANSITION = {
  duration: 0.3,
} as const;

// Type for tracking rule changes
interface RuleChange {
  eventId: number;
  enabled?: boolean;
  threshold_value?: number;
  threshold_operator?: "gt" | "lt" | "eq" | "gte" | "lte";
}

export const NotificationSettings: React.FC = () => {
  const queryClient = useQueryClient();
  const [activeChannelId, setActiveChannelId] = useState<number | null>(null);
  const [showAddChannel, setShowAddChannel] = useState(false);
  const [testingChannelId, setTestingChannelId] = useState<number | null>(null);
  const [pendingChanges, setPendingChanges] = useState<Map<number, RuleChange>>(
    new Map()
  );
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [isMobileView, setIsMobileView] = useState(false);

  // Queries
  const { data: channels = [], isLoading: channelsLoading } = useQuery({
    queryKey: ["notification-channels"],
    queryFn: notificationsApi.getChannels,
  });

  const { data: events = [], isLoading: eventsLoading } = useQuery({
    queryKey: ["notification-events"],
    queryFn: () => notificationsApi.getEvents(),
  });

  const { data: rules = [], isLoading: rulesLoading } = useQuery({
    queryKey: ["notification-rules", activeChannelId],
    queryFn: () => notificationsApi.getRules(activeChannelId || undefined),
    enabled: !!activeChannelId,
    staleTime: 5 * 60 * 1000, // 5 minutes
    placeholderData: (previousData) => previousData,
  });

  // Mutations
  const createChannelMutation = useMutation({
    mutationFn: notificationsApi.createChannel,
    onSuccess: (newChannel) => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
      showToast("Notification channel created", "success");
      setShowAddChannel(false);
      setActiveChannelId(newChannel.id);
    },
    onError: () => {
      showToast("Failed to create channel", "error");
    },
  });

  const updateChannelMutation = useMutation({
    mutationFn: ({
      id,
      ...input
    }: { id: number } & Parameters<typeof notificationsApi.updateChannel>[1]) =>
      notificationsApi.updateChannel(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
      showToast("Channel updated", "success");
    },
    onError: () => {
      showToast("Failed to update channel", "error");
    },
  });

  const deleteChannelMutation = useMutation({
    mutationFn: notificationsApi.deleteChannel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
      showToast("Channel deleted", "success");
      setActiveChannelId(null);
    },
    onError: () => {
      showToast("Failed to delete channel", "error");
    },
  });

  const testChannelMutation = useMutation({
    mutationFn: notificationsApi.testChannel,
    onSuccess: () => {
      showToast("Test notification sent!", "success");
    },
    onError: () => {
      showToast("Failed to send test notification", "error");
    },
  });

  const updateRuleMutation = useMutation({
    mutationFn: ({
      id,
      ...input
    }: { id: number } & Partial<NotificationRuleInput>) =>
      notificationsApi.updateRule(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
    },
    onError: () => {
      showToast("Failed to update rule", "error");
    },
  });

  const createRuleMutation = useMutation({
    mutationFn: notificationsApi.createRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
    },
    onError: () => {
      showToast("Failed to create rule", "error");
    },
  });

  // Mobile view detection
  useEffect(() => {
    const checkMobile = () => setIsMobileView(window.innerWidth < 768);
    checkMobile();
    window.addEventListener("resize", checkMobile);
    return () => window.removeEventListener("resize", checkMobile);
  }, []);

  // Reset pending changes when switching channels
  useEffect(() => {
    setPendingChanges(new Map());
    setHasUnsavedChanges(false);
  }, [activeChannelId]);

  // Track when there are unsaved changes
  useEffect(() => {
    setHasUnsavedChanges(pendingChanges.size > 0);
  }, [pendingChanges]);

  // Warn user about unsaved changes when leaving
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (hasUnsavedChanges) {
        e.preventDefault();
        // Modern browsers ignore custom messages
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [hasUnsavedChanges]);

  if (channelsLoading || eventsLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full mx-auto mb-4 animate-spin" />
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Loading notification settings...
          </p>
        </div>
      </div>
    );
  }

  // Ensure channels and events are arrays (handle null/undefined from API)
  const safeChannels = channels || [];
  const safeEvents = events || [];

  // Group events by category
  const eventsByCategory = safeEvents.reduce((acc, event) => {
    if (!acc[event.category]) {
      acc[event.category] = [];
    }
    acc[event.category].push(event);
    return acc;
  }, {} as Record<string, NotificationEvent[]>);

  const activeChannel = safeChannels.find((c) => c.id === activeChannelId);

  // Helper function to get the current state of a rule (including pending changes)
  const getRuleState = (eventId: number): Partial<NotificationRule> => {
    const existingRule = rules.find((r) => r.event_id === eventId);
    const pendingChange = pendingChanges.get(eventId);

    return {
      ...existingRule,
      ...pendingChange,
    };
  };

  // Update a pending change
  const updatePendingChange = (
    eventId: number,
    update: Partial<RuleChange>
  ) => {
    const event = events.find((e) => e.id === eventId);
    const newChanges = new Map(pendingChanges);
    const existingChange = newChanges.get(eventId) || { eventId };

    // Merge the update with existing pending changes
    const updatedChange = { ...existingChange, ...update };

    // If enabling a rule that supports thresholds, ensure default operator is set
    if (
      update.enabled &&
      event?.supports_threshold &&
      !updatedChange.threshold_operator
    ) {
      const existingRule = rules.find((r) => r.event_id === eventId);
      if (!existingRule?.threshold_operator) {
        updatedChange.threshold_operator = "gt";
      }
    }

    newChanges.set(eventId, updatedChange);
    setPendingChanges(newChanges);
  };

  // Save all pending changes
  const saveChanges = async () => {
    if (!activeChannel) return;

    const promises: Promise<any>[] = [];

    for (const [eventId, change] of pendingChanges) {
      const existingRule = rules.find((r) => r.event_id === eventId);

      if (existingRule) {
        // Update existing rule
        promises.push(
          updateRuleMutation.mutateAsync({
            id: existingRule.id,
            ...change,
          })
        );
      } else if (change.enabled) {
        // Create new rule only if it's being enabled
        const newRule: NotificationRuleInput = {
          channel_id: activeChannel.id,
          event_id: eventId,
          enabled: change.enabled,
          threshold_value: change.threshold_value,
          threshold_operator: change.threshold_operator,
        };
        promises.push(createRuleMutation.mutateAsync(newRule));
      }
    }

    try {
      await Promise.all(promises);
      showToast("Notification settings saved", "success");
      setPendingChanges(new Map());
      setHasUnsavedChanges(false);
    } catch (error) {
      showToast("Failed to save some changes", "error");
    }
  };

  // Cancel pending changes
  const cancelChanges = () => {
    setPendingChanges(new Map());
    setHasUnsavedChanges(false);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="relative">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
              <BellIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
              Notification Settings
            </h3>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              Configure notification channels and rules for system alerts
            </p>
          </div>

          <AnimatePresence>
            {hasUnsavedChanges && (
              <motion.div
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
                transition={FADE_TRANSITION}
                className="flex items-center gap-2"
              >
                <ExclamationTriangleIcon className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                <span className="text-sm font-medium text-amber-600 dark:text-amber-400">
                  Unsaved changes
                </span>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>

      {/* Mobile View */}
      {isMobileView ? (
        <div className="h-full">
          <MobileNotificationView
            channels={safeChannels}
            activeChannel={activeChannel}
            activeChannelId={activeChannelId}
            setActiveChannelId={setActiveChannelId}
            showAddChannel={showAddChannel}
            setShowAddChannel={setShowAddChannel}
            hasUnsavedChanges={hasUnsavedChanges}
            pendingChanges={pendingChanges}
            rules={rules}
            rulesLoading={rulesLoading}
            eventsByCategory={eventsByCategory}
            getRuleState={getRuleState}
            updatePendingChange={updatePendingChange}
            saveChanges={saveChanges}
            cancelChanges={cancelChanges}
            createChannelMutation={createChannelMutation}
            updateChannelMutation={updateChannelMutation}
            deleteChannelMutation={deleteChannelMutation}
            testChannelMutation={testChannelMutation}
            testingChannelId={testingChannelId}
            setTestingChannelId={setTestingChannelId}
            updateRuleMutation={updateRuleMutation}
            createRuleMutation={createRuleMutation}
          />
        </div>
      ) : (
        /* Desktop View */
        <div className="grid grid-cols-12 gap-8">
          {/* Channels Sidebar */}
          <div className="col-span-4 space-y-4">
            <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
              <div className="mb-6">
                <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  Channels
                </h4>
                <HeadlessButton
                  onClick={() => setShowAddChannel(true)}
                  className="w-full flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors"
                >
                  <PlusIcon className="w-4 h-4" />
                  Add Channel
                </HeadlessButton>
              </div>

              <div className="space-y-3">
                {safeChannels.map((channel) => (
                  <ChannelCard
                    key={channel.id}
                    channel={channel}
                    isActive={activeChannelId === channel.id}
                    hasUnsavedChanges={hasUnsavedChanges}
                    onMouseEnter={() => {
                      // Prefetch rules when hovering
                      queryClient.prefetchQuery({
                        queryKey: ["notification-rules", channel.id],
                        queryFn: () => notificationsApi.getRules(channel.id),
                        staleTime: 5 * 60 * 1000,
                      });
                    }}
                    onClick={() => {
                      // If clicking the active channel, deselect it
                      if (activeChannelId === channel.id) {
                        setActiveChannelId(null);
                      } else {
                        // If there are unsaved changes, confirm before switching
                        if (hasUnsavedChanges) {
                          if (
                            confirm(
                              "You have unsaved changes. Do you want to discard them?"
                            )
                          ) {
                            setActiveChannelId(channel.id);
                          }
                        } else {
                          setActiveChannelId(channel.id);
                        }
                      }
                    }}
                  />
                ))}

                {safeChannels.length === 0 && (
                  <div className="text-center py-12">
                    <BellIcon className="w-12 h-12 text-gray-400 mx-auto mb-3" />
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      No channels configured
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                      Add a channel to get started
                    </p>
                  </div>
                )}
              </div>
            </div>

            {/* Add Channel Form */}
            <AnimatePresence>
              {showAddChannel && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={SLIDE_TRANSITION}
                  className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6"
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
          </div>

          {/* Channel Details and Rules */}
          <div className="col-span-8">
            {activeChannel ? (
              <div className="space-y-6" key={`channel-${activeChannel.id}`}>
                {/* Channel Details */}
                <ChannelDetails
                  channel={activeChannel}
                  onUpdate={updateChannelMutation}
                  onDelete={() => {
                    if (
                      confirm(
                        "Are you sure you want to delete this channel? This action cannot be undone."
                      )
                    ) {
                      deleteChannelMutation.mutate(activeChannel.id);
                    }
                  }}
                  onTest={() => {
                    setTestingChannelId(activeChannel.id);
                    testChannelMutation.mutate(activeChannel.id);
                  }}
                  isDeleting={deleteChannelMutation.isPending}
                  isTesting={
                    testingChannelId === activeChannel.id &&
                    testChannelMutation.isPending
                  }
                />

                {/* Notification Rules */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
                  <div className="flex items-center justify-between mb-6">
                    <div>
                      <h4 className="text-lg font-semibold text-gray-900 dark:text-white">
                        Notification Rules
                      </h4>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        Choose which events trigger notifications
                      </p>
                    </div>

                    <AnimatePresence>
                      {hasUnsavedChanges && (
                        <motion.div
                          initial={{ opacity: 0, x: 20 }}
                          animate={{ opacity: 1, x: 0 }}
                          exit={{ opacity: 0, x: 20 }}
                          transition={FADE_TRANSITION}
                          className="flex items-center gap-2"
                        >
                          <HeadlessButton
                            onClick={cancelChanges}
                            className="flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors"
                          >
                            <XMarkIcon className="w-4 h-4" />
                            Cancel
                          </HeadlessButton>
                          <HeadlessButton
                            onClick={saveChanges}
                            disabled={
                              updateRuleMutation.isPending ||
                              createRuleMutation.isPending
                            }
                            className="flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <CheckIcon className="w-4 h-4" />
                            Save Changes
                          </HeadlessButton>
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>

                  <div className="space-y-4">
                    {Object.entries(eventsByCategory).map(
                      ([category, categoryEvents]) => (
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
                      )
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="space-y-6">
                {/* Welcome Section */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-8">
                  <div className="flex items-start gap-6">
                    <div className="flex-shrink-0">
                      <div className="inline-flex items-center justify-center w-14 h-14 bg-blue-500/10 rounded-full">
                        <BellIcon className="w-7 h-7 text-blue-600 dark:text-blue-400" />
                      </div>
                    </div>
                    <div className="flex-1">
                      <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                        Welcome to Notification Settings
                      </h3>
                      <p className="text-gray-600 dark:text-gray-400 mb-4">
                        Set up notification channels to receive alerts about
                        system events, speed tests, and monitoring updates.
                        Netronome supports 15+ notification services through
                        Shoutrrr.
                      </p>
                      <div className="flex flex-wrap gap-2">
                        <span className="px-3 py-1 bg-gray-200/50 dark:bg-gray-800/50 rounded-full text-xs font-medium text-gray-700 dark:text-gray-300">
                          Discord
                        </span>
                        <span className="px-3 py-1 bg-gray-200/50 dark:bg-gray-800/50 rounded-full text-xs font-medium text-gray-700 dark:text-gray-300">
                          Telegram
                        </span>
                        <span className="px-3 py-1 bg-gray-200/50 dark:bg-gray-800/50 rounded-full text-xs font-medium text-gray-700 dark:text-gray-300">
                          Slack
                        </span>
                        <span className="px-3 py-1 bg-gray-200/50 dark:bg-gray-800/50 rounded-full text-xs font-medium text-gray-700 dark:text-gray-300">
                          Email
                        </span>
                        <span className="px-3 py-1 bg-gray-200/50 dark:bg-gray-800/50 rounded-full text-xs font-medium text-gray-700 dark:text-gray-300">
                          Pushover
                        </span>
                        <span className="px-3 py-1 bg-gray-200/50 dark:bg-gray-800/50 rounded-full text-xs font-medium text-gray-700 dark:text-gray-300">
                          And more...
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Getting Started Guide */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <div className="flex items-center justify-center w-10 h-10 bg-emerald-500/10 rounded-lg">
                        <span className="text-lg font-semibold text-emerald-700 dark:text-emerald-400">1</span>
                      </div>
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Create a Channel
                      </h4>
                    </div>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      Click "Add Channel" in the sidebar to configure your first
                      notification service. You'll need the service URL from
                      your provider.
                    </p>
                  </div>

                  <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <div className="flex items-center justify-center w-10 h-10 bg-blue-500/10 rounded-lg">
                        <span className="text-lg font-semibold text-blue-700 dark:text-blue-400">2</span>
                      </div>
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Configure Rules
                      </h4>
                    </div>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      Select which events trigger notifications and set
                      thresholds for alerts like high CPU usage or failed speed
                      tests.
                    </p>
                  </div>

                  <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <div className="flex items-center justify-center w-10 h-10 bg-purple-500/10 rounded-lg">
                        <span className="text-lg font-semibold text-purple-700 dark:text-purple-400">3</span>
                      </div>
                      <h4 className="font-semibold text-gray-900 dark:text-white">
                        Stay Informed
                      </h4>
                    </div>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      Receive real-time alerts about system performance, agent
                      status, and network quality directly to your preferred
                      platform.
                    </p>
                  </div>
                </div>

                {/* Available Notifications */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
                  <h4 className="font-semibold text-gray-900 dark:text-white mb-4">
                    Available Notifications
                  </h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="flex items-start gap-3">
                      <div className="flex-shrink-0">
                        <div className="inline-flex items-center justify-center w-12 h-12 bg-blue-500/10 rounded-lg">
                          <RocketLaunchIcon className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                        </div>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-900 dark:text-white">
                          Speed Test Events
                        </h5>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          Completed tests, failures, and performance thresholds
                        </p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3">
                      <div className="flex-shrink-0">
                        <div className="inline-flex items-center justify-center w-12 h-12 bg-emerald-500/10 rounded-lg">
                          <SignalIcon className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                        </div>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-900 dark:text-white">
                          Packet Loss Monitoring
                        </h5>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          Network quality degradation and recovery alerts
                        </p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3">
                      <div className="flex-shrink-0">
                        <div className="inline-flex items-center justify-center w-12 h-12 bg-purple-500/10 rounded-lg">
                          <ComputerDesktopIcon className="w-6 h-6 text-purple-600 dark:text-purple-400" />
                        </div>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-900 dark:text-white">
                          System Monitoring
                        </h5>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          Agent status, CPU, memory, disk, bandwidth, and temperature alerts
                        </p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Call to Action */}
                <div className="bg-blue-500/10 border border-blue-500/30 rounded-xl p-6 text-center">
                  <p className="text-sm text-blue-700 dark:text-blue-300 mb-3">
                    Ready to get started? Create your first notification channel
                    to begin receiving alerts.
                  </p>
                  <HeadlessButton
                    onClick={() => setShowAddChannel(true)}
                    className="inline-flex items-center justify-center gap-2 px-6 py-2.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors"
                  >
                    <PlusIcon className="w-4 h-4" />
                    Create Your First Channel
                  </HeadlessButton>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
