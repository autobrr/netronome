/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { 
  ChevronRightIcon,
  RocketLaunchIcon,
  SignalIcon,
  ComputerDesktopIcon,
  BellIcon
} from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import { EventRuleItem } from "./EventRuleItem";
import { Card } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import type {
  NotificationEvent,
  NotificationRule,
  NotificationRuleInput,
} from "@/api/notifications";

interface EventCategorySectionProps {
  category: string;
  events: NotificationEvent[];
  pendingChanges: Map<number, any>;
  getRuleState: (eventId: number) => Partial<NotificationRule>;
  onUpdateRule: (
    eventId: number,
    input: Partial<NotificationRuleInput>
  ) => void;
}

export const EventCategorySection: React.FC<EventCategorySectionProps> = ({
  category,
  events,
  pendingChanges,
  getRuleState,
  onUpdateRule,
}) => {

  const categoryTitle = category.charAt(0).toUpperCase() + category.slice(1);
  
  // Get category icon component
  const getCategoryIcon = () => {
    switch (category) {
      case "speedtest":
        return <RocketLaunchIcon className="w-6 h-6 text-blue-600 dark:text-blue-400" />;
      case "packetloss":
        return <SignalIcon className="w-6 h-6 text-amber-600 dark:text-amber-400" />;
      case "agent":
        return <ComputerDesktopIcon className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />;
      default:
        return <BellIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />;
    }
  };
  
  // Count enabled rules
  const enabledCount = events.filter(event => {
    const ruleState = getRuleState(event.id);
    return ruleState.enabled === true;
  }).length;

  // Category-specific styling
  const categoryStyles = {
    speedtest: "border-blue-500/30 bg-blue-500/5",
    packetloss: "border-amber-500/30 bg-amber-500/5",
    agent: "border-emerald-500/30 bg-emerald-500/5",
  };

  return (
    <Card
      className={cn(
        "overflow-hidden p-0 bg-transparent",
        categoryStyles[category as keyof typeof categoryStyles] ||
          "border-gray-200 dark:border-gray-800"
      )}
    >
      <Collapsible>
        <CollapsibleTrigger className="w-full px-4 py-3 flex items-center justify-between hover:bg-gray-100/50 dark:hover:bg-gray-800/50 transition-colors group">
          <div className="flex items-center gap-3">
            <div className="flex-shrink-0">
              {getCategoryIcon()}
            </div>
            <div className="text-left">
              <h5 className="font-semibold text-gray-900 dark:text-white">
                {categoryTitle}
              </h5>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                {enabledCount > 0 ? (
                  <span className="font-medium text-emerald-600 dark:text-emerald-400">
                    {enabledCount} of {events.length} rules enabled
                  </span>
                ) : (
                  <span>
                    {events.length} {events.length === 1 ? "event" : "events"} available
                  </span>
                )}
              </p>
            </div>
          </div>
          <ChevronRightIcon className="w-5 h-5 text-gray-400 transition-transform duration-200 group-data-[state=open]:rotate-90" />
        </CollapsibleTrigger>

        <CollapsibleContent>
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
        </CollapsibleContent>
      </Collapsible>
    </Card>
  );
};
