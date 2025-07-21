/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { ChevronRightIcon } from "@heroicons/react/24/outline";
import { getEventCategoryIcon } from "@/api/notifications";
import { cn } from "@/lib/utils";
import { EventRuleItem } from "./EventRuleItem";
import type { NotificationEvent, NotificationRule, NotificationRuleInput } from "@/api/notifications";

interface EventCategorySectionProps {
  category: string;
  events: NotificationEvent[];
  pendingChanges: Map<number, any>;
  getRuleState: (eventId: number) => Partial<NotificationRule>;
  onUpdateRule: (eventId: number, input: Partial<NotificationRuleInput>) => void;
}

const SPRING_TRANSITION = {
  type: "spring" as const,
  stiffness: 500,
  damping: 30,
} as const;

const SLIDE_TRANSITION = {
  duration: 0.3,
} as const;

export const EventCategorySection: React.FC<EventCategorySectionProps> = ({
  category,
  events,
  pendingChanges,
  getRuleState,
  onUpdateRule,
}) => {
  const [expanded, setExpanded] = useState(true);

  const categoryTitle = category.charAt(0).toUpperCase() + category.slice(1);
  const categoryIcon = getEventCategoryIcon(category);

  // Category-specific styling
  const categoryStyles = {
    speedtest: "border-blue-500/30 bg-blue-500/5",
    packetloss: "border-amber-500/30 bg-amber-500/5",
    agent: "border-emerald-500/30 bg-emerald-500/5",
  };

  return (
    <div className={cn(
      "rounded-lg border transition-all",
      categoryStyles[category as keyof typeof categoryStyles] || "border-gray-200 dark:border-gray-800"
    )}>
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full px-4 py-3 flex items-center justify-between hover:bg-gray-100/50 dark:hover:bg-gray-800/50 transition-colors rounded-t-lg"
      >
        <div className="flex items-center gap-3">
          <span className="text-2xl">{categoryIcon}</span>
          <div className="text-left">
            <h5 className="font-semibold text-gray-900 dark:text-white">
              {categoryTitle}
            </h5>
            <p className="text-xs text-gray-600 dark:text-gray-400">
              {events.length} {events.length === 1 ? 'event' : 'events'} available
            </p>
          </div>
        </div>
        <motion.div
          animate={{ rotate: expanded ? 90 : 0 }}
          transition={SPRING_TRANSITION}
        >
          <ChevronRightIcon className="w-5 h-5 text-gray-400" />
        </motion.div>
      </button>

      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={SLIDE_TRANSITION}
            className="overflow-hidden"
          >
            <div className="p-4 space-y-3 border-t border-gray-200/50 dark:border-gray-800/50">
              {events.map((event) => (
                <EventRuleItem
                  key={event.id}
                  event={event}
                  ruleState={getRuleState(event.id)}
                  hasPendingChanges={pendingChanges.has(event.id)}
                  onUpdateRule={(input) => onUpdateRule(event.id, input)}
                />
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};