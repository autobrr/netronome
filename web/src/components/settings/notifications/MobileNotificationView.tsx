/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  BellIcon,
  PlusIcon,
  InformationCircleIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
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
  deleteChannelMutation: any;
  testChannelMutation: any;
  testingChannelId: number | null;
  setTestingChannelId: (id: number | null) => void;
  updateRuleMutation: any;
  createRuleMutation: any;
}

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
  deleteChannelMutation,
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
      {activeTab === "channels" ? (
        <div className="space-y-4">
          {/* Add Channel Button */}
          <Button
            onClick={() => setShowAddChannel(true)}
            className="w-full"
          >
            <PlusIcon className="w-4 h-4" />
            Add New Channel
          </Button>

          {/* Add Channel Form */}
          {showAddChannel && (
            <Card>
              <CardContent className="pt-4">
                <h5 className="font-medium text-gray-900 dark:text-white mb-3">
                  Add New Channel
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
          )}

          {/* Channels List */}
          <div className="space-y-3">
            {channels.map((channel) => (
              <Card
                key={channel.id}
                className="p-3"
              >
                <div
                  onClick={() => {
                    setActiveChannelId(channel.id);
                    setActiveTab("rules");
                  }}
                  className="cursor-pointer"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <div
                        className={cn(
                          "w-2 h-2 rounded-full flex-shrink-0",
                          channel.enabled ? "bg-emerald-500" : "bg-gray-400"
                        )}
                      />
                      <h5 className="font-medium text-gray-900 dark:text-white truncate">
                        {channel.name}
                      </h5>
                    </div>
                    {activeChannelId === channel.id && (
                      <Badge variant="default" className="text-xs flex-shrink-0">
                        Active
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-500 font-mono truncate">
                    {(() => {
                      const serviceType =
                        channel.url?.match(/^(\w+):\/\//)?.[1];
                      return serviceType
                        ? `${serviceType}://...`
                        : "Unknown service";
                    })()}
                  </p>
                </div>

                <div className="flex items-center gap-2 mt-3 pt-3 border-t border-gray-200 dark:border-gray-800">
                  <Button
                    onClick={() => {
                      setTestingChannelId(channel.id);
                      testChannelMutation.mutate(channel.id);
                    }}
                    disabled={
                      testingChannelId === channel.id &&
                      testChannelMutation.isPending
                    }
                    variant="secondary"
                    size="sm"
                    className="flex-1"
                    isLoading={testingChannelId === channel.id && testChannelMutation.isPending}
                  >
                    {!(testingChannelId === channel.id && testChannelMutation.isPending) && (
                      <BellIcon className="w-4 h-4" />
                    )}
                    Test
                  </Button>
                  <Button
                    onClick={() => {
                      if (
                        confirm(
                          `Are you sure you want to delete "${channel.name}"? This action cannot be undone.`
                        )
                      ) {
                        deleteChannelMutation.mutate(channel.id);
                      }
                    }}
                    disabled={deleteChannelMutation.isPending}
                    variant="destructive"
                    size="icon"
                    className="h-8 w-8"
                  >
                    <TrashIcon className="w-4 h-4" />
                  </Button>
                  <Switch
                    checked={channel.enabled}
                    onCheckedChange={(checked) => {
                      updateChannelMutation.mutate({
                        id: channel.id,
                        name: channel.name,
                        url: channel.url,
                        enabled: checked,
                      });
                    }}
                  />
                </div>
              </Card>
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
        </div>
      ) : (
        <div className="space-y-4">
          {activeChannel ? (
            <>
              {/* Active Channel Info */}
              <Card className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h5 className="font-medium text-gray-900 dark:text-white">
                      {activeChannel.name}
                    </h5>
                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                      Configure notification rules
                    </p>
                  </div>
                  <Button
                    onClick={() => setActiveTab("channels")}
                    variant="link"
                    size="sm"
                    className="h-auto p-0"
                  >
                    Change
                  </Button>
                </div>
              </Card>

              {/* Save/Cancel Buttons */}
              {hasUnsavedChanges && (
                <div className="flex items-center gap-2">
                  <Button
                    onClick={cancelChanges}
                    variant="secondary"
                    size="sm"
                    className="flex-1"
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={saveChanges}
                    disabled={
                      updateRuleMutation.isPending ||
                      createRuleMutation.isPending
                    }
                    size="sm"
                    className="flex-1"
                    isLoading={updateRuleMutation.isPending || createRuleMutation.isPending}
                  >
                    Save Changes
                  </Button>
                </div>
              )}

              {/* Rules */}
              {rulesLoading ? (
                <div className="text-center py-8">
                  <div className="w-6 h-6 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full mx-auto mb-3 animate-spin" />
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Loading rules...
                  </p>
                </div>
              ) : (
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
              )}
            </>
          ) : (
            /* Mobile Empty State */
            <div className="space-y-6">
              <Card className="p-6 text-center">
                <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-500/10 rounded-full mb-4">
                  <BellIcon className="w-8 h-8 text-blue-600 dark:text-blue-400" />
                </div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                  Get Started with Notifications
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
                  Create notification channels to receive alerts about system
                  events and monitoring updates.
                </p>
                <Button
                  onClick={() => setActiveTab("channels")}
                  variant="link"
                  size="sm"
                  className="h-auto"
                >
                  Go to Channels tab to add your first channel
                </Button>
              </Card>

              <Card>
                <CardContent className="p-4">
                  <h4 className="font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                    <InformationCircleIcon className="w-5 h-5 text-gray-500" />
                    Available Notifications
                  </h4>
                  <div className="space-y-3">
                    <div className="flex items-start gap-2">
                      <span className="text-base">🚀</span>
                      <div className="text-sm">
                        <span className="font-medium text-gray-900 dark:text-white">
                          Speed Tests
                        </span>
                        <p className="text-gray-600 dark:text-gray-400 text-xs">
                          Test results and failures
                        </p>
                      </div>
                    </div>
                    <div className="flex items-start gap-2">
                      <span className="text-base">📡</span>
                      <div className="text-sm">
                        <span className="font-medium text-gray-900 dark:text-white">
                          Packet Loss
                        </span>
                        <p className="text-gray-600 dark:text-gray-400 text-xs">
                          Network quality alerts
                        </p>
                      </div>
                    </div>
                    <div className="flex items-start gap-2">
                      <span className="text-base">🖥️</span>
                      <div className="text-sm">
                        <span className="font-medium text-gray-900 dark:text-white">
                          Agent Status
                        </span>
                        <p className="text-gray-600 dark:text-gray-400 text-xs">
                          Online/offline & resources
                        </p>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
