/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion, AnimatePresence } from "motion/react";
import { Switch, Input, Select } from "@headlessui/react";
import { InformationCircleIcon } from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import type { NotificationEvent, NotificationRule, NotificationRuleInput } from "@/api/notifications";

interface EventRuleItemProps {
  event: NotificationEvent;
  ruleState: Partial<NotificationRule>;
  hasPendingChanges: boolean;
  onUpdateRule: (input: Partial<NotificationRuleInput>) => void;
}

const SLIDE_TRANSITION = {
  duration: 0.3,
} as const;

export const EventRuleItem: React.FC<EventRuleItemProps> = ({
  event,
  ruleState,
  hasPendingChanges,
  onUpdateRule,
}) => {
  const isEnabled = ruleState.enabled ?? false;

  return (
    <motion.div
      layout
      className={cn(
        "p-4 rounded-lg transition-all",
        hasPendingChanges 
          ? "bg-blue-500/10 border border-blue-500/30 shadow-md" 
          : "bg-gray-100/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-800"
      )}
    >
      <div className="flex items-start gap-3">
        <Switch
          checked={isEnabled}
          onChange={(checked) => onUpdateRule({ enabled: checked })}
          className={cn(
            "relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2",
            isEnabled ? "bg-blue-600" : "bg-gray-200 dark:bg-gray-700"
          )}
        >
          <span className="sr-only">Enable notification for {event.name}</span>
          <motion.span
            layout
            transition={{
              type: "spring",
              stiffness: 700,
              damping: 30,
            }}
            className={cn(
              "inline-block h-4 w-4 rounded-full bg-white shadow-lg",
              isEnabled ? "translate-x-6" : "translate-x-1"
            )}
          />
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
              <motion.div
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                className="flex items-center gap-1 px-2 py-1 bg-blue-500/20 rounded-full"
              >
                <div className="w-1.5 h-1.5 bg-blue-600 dark:bg-blue-400 rounded-full" />
                <span className="text-xs font-medium text-blue-700 dark:text-blue-300">
                  Modified
                </span>
              </motion.div>
            )}
          </div>

          {/* Threshold Settings */}
          <AnimatePresence>
            {event.supports_threshold && isEnabled && (
              <motion.div
                initial={{ opacity: 0, height: 0, marginTop: 0 }}
                animate={{ opacity: 1, height: "auto", marginTop: 12 }}
                exit={{ opacity: 0, height: 0, marginTop: 0 }}
                transition={SLIDE_TRANSITION}
                className="flex flex-wrap items-center gap-3"
              >
                <div className="flex items-center gap-2 px-3 py-1.5 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg">
                  <InformationCircleIcon className="w-4 h-4 text-gray-500" />
                  <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                    Trigger when value is
                  </span>
                </div>
                
                <Select
                  value={ruleState.threshold_operator || "gt"}
                  onChange={(e) => {
                    onUpdateRule({
                      threshold_operator: e.target.value as NotificationRuleInput["threshold_operator"],
                    });
                  }}
                  className="w-40 px-3 py-1.5 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50 text-sm"
                >
                  <option value="gt">Greater than</option>
                  <option value="lt">Less than</option>
                  <option value="eq">Equal to</option>
                  <option value="gte">Greater or equal</option>
                  <option value="lte">Less or equal</option>
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
                    className="w-24 px-3 py-1.5 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50 text-sm"
                  />
                  <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    {event.threshold_unit}
                  </span>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </motion.div>
  );
};