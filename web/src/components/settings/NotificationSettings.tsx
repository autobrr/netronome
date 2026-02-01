/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { motion, AnimatePresence } from "motion/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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
  const { t } = useTranslation();
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
      showToast(t('notifications.serviceAdded'), "success", {
        description: `"${newChannel.name}" ${t('notificationSettings.channelAdded')}`,
      });
      setShowAddChannel(false);
      setActiveChannelId(newChannel.id);
    },
    onError: () => {
      showToast(t('notifications.addServiceFailed'), "error");
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
      showToast(t('notifications.serviceUpdated'), "success", {
        description: t('notificationSettings.channelUpdated'),
      });
    },
    onError: () => {
      showToast(t('notifications.updateServiceFailed'), "error");
    },
  });

  const deleteChannelMutation = useMutation({
    mutationFn: notificationsApi.deleteChannel,
    onSuccess: (_, channelId) => {
      // Store the deleted channel data for potential undo
      const deletedChannel = channels.find((ch) => ch.id === channelId);
      
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
      queryClient.invalidateQueries({ queryKey: ["notification-rules"] });
      
      showToast(t('notifications.serviceDeleted'), "success", {
        description: deletedChannel ? `"${deletedChannel.name}" ${t('notificationSettings.channelDeleted')}` : undefined
      });
      setActiveChannelId(null);
    },
    onError: () => {
      showToast(t('notifications.deleteServiceFailed'), "error");
    },
  });

  const testChannelMutation = useMutation({
    mutationFn: notificationsApi.testChannel,
    onSuccess: () => {
      showToast(t('notificationSettings.testNotificationSent'), "success", {
        description: t('notificationSettings.testNotificationDesc'),
      });
    },
    onError: () => {
      showToast(t('notifications.testFailed'), "error");
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
      showToast(t('notificationSettings.failedToCreate'), "error");
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

    const promises: Promise<NotificationRule>[] = [];

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
      showToast(t('notificationSettings.settingsSaved'), "success", {
        description: t('notificationSettings.allChangesSaved'),
      });
      setPendingChanges(new Map());
      setHasUnsavedChanges(false);
    } catch {
      showToast(t('notificationSettings.failedToSave'), "error");
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
      <div className="relative px-4 sm:px-0">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
              <BellIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
              {t('notificationSettings.title')}
            </h3>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              {t('notificationSettings.configure')}
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
                  {t('notificationSettings.unsavedChanges')}
                </span>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>

      {/* Mobile View */}
      {isMobileView ? (
        <div className="h-full px-4">
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
            <Card>
              <CardContent className="p-6">
                <div className="mb-6">
                  <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                    {t('notificationSettings.channels')}
                  </h4>
                <Button
                  onClick={() => setShowAddChannel(true)}
                  className="w-full"
                >
                  <PlusIcon className="w-4 h-4" />
                  {t('notificationSettings.addChannel')}
                </Button>
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
                            confirm(t('notificationSettings.unsavedWarning'))
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
                      {t('notificationSettings.noChannels')}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                      {t('notificationSettings.addChannelPrompt')}
                    </p>
                  </div>
                )}
                </div>
              </CardContent>
            </Card>

            {/* Add Channel Form */}
            <AnimatePresence>
              {showAddChannel && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={SLIDE_TRANSITION}
                >
                  <Card>
                    <CardContent className="p-6">
                      <h5 className="font-medium text-gray-900 dark:text-white mb-4">
                        {t('notificationSettings.addNewChannel')}
                      </h5>
                  <AddChannelForm
                    onSubmit={(input) => {
                      createChannelMutation.mutate(input);
                    }}
                    onCancel={() => setShowAddChannel(false)}
                    isLoading={createChannelMutation.isPending}
                      />
                    </CardContent>
                  </Card>
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
                      confirm(t('notificationSettings.deleteConfirmation'))
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
                <Card>
                  <CardHeader className="p-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <CardTitle>{t('notificationSettings.notificationRules')}</CardTitle>
                        <CardDescription>
                          {t('notificationSettings.chooseEvents')}
                        </CardDescription>
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
                          <Button
                            onClick={cancelChanges}
                            variant="secondary"
                            size="sm"
                          >
                            <XMarkIcon className="w-4 h-4" />
                            {t('common.cancel')}
                          </Button>
                          <Button
                            onClick={saveChanges}
                            disabled={
                              updateRuleMutation.isPending ||
                              createRuleMutation.isPending
                            }
                            size="sm"
                          >
                            <CheckIcon className="w-4 h-4" />
                            {t('common.saveChanges')}
                          </Button>
                        </motion.div>
                      )}
                    </AnimatePresence>
                    </div>
                  </CardHeader>
                  <CardContent className="p-6 pt-0">
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
                  </CardContent>
                </Card>
              </div>
            ) : (
              <div className="space-y-6">
                {/* Welcome Section */}
                <Card className="p-8">
                  <div className="flex items-start gap-6">
                    <div className="flex-shrink-0">
                      <div className="inline-flex items-center justify-center w-14 h-14 bg-blue-500/10 rounded-full">
                        <BellIcon className="w-7 h-7 text-blue-600 dark:text-blue-400" />
                      </div>
                    </div>
                    <div className="flex-1">
                      <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                        {t('notificationSettings.welcomeTitle')}
                      </h3>
                      <p className="text-gray-600 dark:text-gray-400 mb-4">
                        {t('notificationSettings.welcomeMessage')}
                      </p>
                      <div className="flex flex-wrap gap-2">
                        <Badge variant="secondary">Discord</Badge>
                        <Badge variant="secondary">Telegram</Badge>
                        <Badge variant="secondary">Slack</Badge>
                        <Badge variant="secondary">Email</Badge>
                        <Badge variant="secondary">Pushover</Badge>
                        <Badge variant="secondary">And more...</Badge>
                      </div>
                    </div>
                  </div>
                </Card>

                {/* Getting Started Guide */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <Card>
                    <CardContent className="p-6">
                      <div className="flex items-center gap-3 mb-4">
                        <div className="flex items-center justify-center w-10 h-10 bg-emerald-500/10 rounded-lg">
                          <span className="text-lg font-semibold text-emerald-700 dark:text-emerald-400">1</span>
                        </div>
                        <h4 className="font-semibold text-gray-900 dark:text-white">
                          {t('notificationSettings.createChannelStep')}
                        </h4>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {t('notificationSettings.createChannelDesc')}
                      </p>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardContent className="p-6">
                      <div className="flex items-center gap-3 mb-4">
                        <div className="flex items-center justify-center w-10 h-10 bg-blue-500/10 rounded-lg">
                          <span className="text-lg font-semibold text-blue-700 dark:text-blue-400">2</span>
                        </div>
                        <h4 className="font-semibold text-gray-900 dark:text-white">
                          {t('notificationSettings.configureRulesStep')}
                        </h4>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {t('notificationSettings.configureRulesDesc')}
                      </p>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardContent className="p-6">
                      <div className="flex items-center gap-3 mb-4">
                        <div className="flex items-center justify-center w-10 h-10 bg-purple-500/10 rounded-lg">
                          <span className="text-lg font-semibold text-purple-700 dark:text-purple-400">3</span>
                        </div>
                        <h4 className="font-semibold text-gray-900 dark:text-white">
                          {t('notificationSettings.stayInformed')}
                        </h4>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {t('notificationSettings.stayInformedDesc')}
                      </p>
                    </CardContent>
                  </Card>
                </div>

                {/* Available Notifications */}
                <Card>
                  <CardHeader className="p-6 pb-4">
                    <CardTitle>{t('notificationSettings.availableNotifications')}</CardTitle>
                  </CardHeader>
                  <CardContent className="p-6 pt-0">
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="flex items-start gap-3">
                      <div className="flex-shrink-0">
                        <div className="inline-flex items-center justify-center w-12 h-12 bg-blue-500/10 rounded-lg">
                          <RocketLaunchIcon className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                        </div>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-900 dark:text-white">
                          {t('notificationSettings.speedTestEvents')}
                        </h5>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {t('notificationSettings.speedTestEventsDesc')}
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
                          {t('notificationSettings.packetLossMonitoring')}
                        </h5>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {t('notificationSettings.packetLossMonitoringDesc')}
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
                          {t('notificationSettings.systemMonitoring')}
                        </h5>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {t('notificationSettings.systemMonitoringDesc')}
                        </p>
                      </div>
                    </div>
                  </div>
                  </CardContent>
                </Card>

                {/* Call to Action */}
                <Card className="bg-blue-500/10 border-blue-500/30">
                  <CardContent className="p-6 text-center">
                    <p className="text-sm text-blue-700 dark:text-blue-300 mb-3">
                      {t('notificationSettings.readyToStart')}
                    </p>
                    <Button
                      onClick={() => setShowAddChannel(true)}
                      className="inline-flex items-center justify-center gap-2"
                    >
                      <PlusIcon className="w-4 h-4" />
                      {t('notificationSettings.createFirstChannel')}
                    </Button>
                  </CardContent>
                </Card>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
