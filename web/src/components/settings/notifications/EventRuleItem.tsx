/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { InformationCircleIcon } from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type {
  NotificationEvent,
  NotificationRule,
  NotificationRuleInput,
} from "@/api/notifications";

interface EventRuleItemProps {
  event: NotificationEvent;
  ruleState: Partial<NotificationRule>;
  hasPendingChanges: boolean;
  onUpdateRule: (input: Partial<NotificationRuleInput>) => void;
}

export const EventRuleItem: React.FC<EventRuleItemProps> = ({
  event,
  ruleState,
  hasPendingChanges,
  onUpdateRule,
}) => {
  const isEnabled = ruleState.enabled ?? false;

  return (
    <div
      className={cn(
        "p-4 rounded-lg",
        hasPendingChanges
          ? "bg-blue-500/10 border border-blue-500/30 shadow-md"
          : "bg-gray-100/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-800"
      )}
    >
      <div className="flex items-start gap-3">
        <Switch
          checked={isEnabled}
          onCheckedChange={(checked) => onUpdateRule({ enabled: checked })}
          className="data-[state=checked]:bg-blue-600"
        >
          <span className="sr-only">Enable notification for {event.name}</span>
        </Switch>

        <div className="flex-1">
          <div className="flex items-start justify-between">
            <div>
              <h6 className="font-medium text-gray-900 dark:text-white">
                {event.name}
              </h6>
              {event.description && (
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  {event.description}
                </p>
              )}
            </div>

            {hasPendingChanges && (
              <div className="flex items-center gap-1 px-2 py-1 bg-blue-500/20 rounded-full">
                <div className="w-1.5 h-1.5 bg-blue-600 dark:bg-blue-400 rounded-full" />
                <span className="text-xs font-medium text-blue-700 dark:text-blue-300">
                  Modified
                </span>
              </div>
            )}
          </div>

          {/* Threshold Settings */}
          {event.supports_threshold && isEnabled && (
            <div className="flex flex-wrap items-center gap-3 mt-3">
              <div className="flex items-center gap-2 px-3 py-1.5 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg">
                <InformationCircleIcon className="w-4 h-4 text-gray-500" />
                <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                  Trigger when value is
                </span>
              </div>

              <Select
                value={ruleState.threshold_operator || "gt"}
                onValueChange={(value) => {
                  onUpdateRule({
                    threshold_operator: value as NotificationRuleInput["threshold_operator"],
                  });
                }}
              >
                <SelectTrigger className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="gt">Greater than</SelectItem>
                  <SelectItem value="lt">Less than</SelectItem>
                  <SelectItem value="eq">Equal to</SelectItem>
                  <SelectItem value="gte">Greater or equal</SelectItem>
                  <SelectItem value="lte">Less or equal</SelectItem>
                </SelectContent>
              </Select>

              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  value={ruleState.threshold_value || ""}
                  onChange={(e) => {
                    const value = parseFloat(e.target.value);
                    if (!isNaN(value)) {
                      onUpdateRule({ threshold_value: value });
                    }
                  }}
                  placeholder="0"
                  className="w-24"
                />
                <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  {event.threshold_unit}
                </span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
