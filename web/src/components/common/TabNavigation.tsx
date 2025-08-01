/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion, AnimatePresence } from "motion/react";
import * as TabsPrimitive from "@radix-ui/react-tabs";
import { cn } from "@/lib/utils";

interface Tab {
  id: string;
  label: string;
  icon?: React.ReactNode;
  shortLabel?: string; // Optional short label for mobile
}

interface TabNavigationProps {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (tabId: string) => void;
}

// Define tab color variants as a const assertion for better type safety
const TAB_COLORS = {
  dashboard: {
    active: "text-blue-600 dark:text-blue-400",
    inactive:
      "text-gray-500 dark:text-gray-400 hover:text-blue-500 dark:hover:text-blue-300",
  },
  speedtest: {
    active: "text-emerald-600 dark:text-emerald-400",
    inactive:
      "text-gray-500 dark:text-gray-400 hover:text-emerald-500 dark:hover:text-emerald-300",
  },
  traceroute: {
    active: "text-amber-600 dark:text-amber-400",
    inactive:
      "text-gray-500 dark:text-gray-400 hover:text-amber-500 dark:hover:text-amber-300",
  },
  monitor: {
    active: "text-purple-600 dark:text-purple-400",
    inactive:
      "text-gray-500 dark:text-gray-400 hover:text-purple-500 dark:hover:text-purple-300",
  },
} as const;

type TabId = keyof typeof TAB_COLORS;

const getTabColors = (tabId: string): (typeof TAB_COLORS)[TabId] => {
  return TAB_COLORS[tabId as TabId] ?? TAB_COLORS.dashboard;
};

// Animation configuration moved outside component to prevent re-creation
const SPRING_TRANSITION = {
  type: "spring" as const,
  stiffness: 500,
  damping: 30,
} as const;

// Smart abbreviation mapping for common tab names
const SMART_ABBREVIATIONS: Record<string, string> = {
  dashboard: "Dash",
  "speed test": "Tests",
  traceroute: "Traces",
  agents: "Agents",
  "packet loss": "Packet",
  settings: "Set",
  monitoring: "Monitor",
  analytics: "Stats",
  history: "History",
  reports: "Reports",
  configuration: "Config",
} as const;

// Function to generate smart abbreviations
const getSmartAbbreviation = (label: string): string => {
  const lowerLabel = label.toLowerCase();

  // Check if we have a predefined abbreviation
  if (SMART_ABBREVIATIONS[lowerLabel]) {
    return SMART_ABBREVIATIONS[lowerLabel];
  }

  // For multi-word labels, use first letter of each word
  const words = label.split(" ");
  if (words.length > 1) {
    // For 2-word labels, take first 3-4 letters of each
    if (words.length === 2) {
      return words.map((w) => w.substring(0, 3)).join("");
    }
    // For 3+ words, use initials
    return words
      .map((w) => w[0])
      .join("")
      .toUpperCase();
  }

  // For single words, intelligently truncate
  if (label.length > 6) {
    // Remove vowels from middle if needed
    const consonants = label.replace(/[aeiou]/gi, "");
    if (consonants.length >= 4 && consonants.length <= 6) {
      return (
        consonants.charAt(0).toUpperCase() + consonants.slice(1).toLowerCase()
      );
    }
    // Otherwise just truncate
    return label.substring(0, 5);
  }

  return label;
};

export const TabNavigation: React.FC<TabNavigationProps> = ({
  tabs,
  activeTab,
  onTabChange,
}) => {
  return (
    <div className="flex items-center justify-center w-full sm:w-auto px-2 sm:px-4">
      <TabsPrimitive.Root
        value={activeTab}
        onValueChange={onTabChange}
        className="w-full sm:w-auto"
      >
        <TabsPrimitive.List
          className="flex space-x-0.5 sm:space-x-2 bg-gray-100/60 dark:bg-gray-800/20 p-1 sm:p-2 rounded-xl shadow-sm border border-gray-200/60 dark:border-gray-800/80 w-full sm:w-auto overflow-x-auto scrollbar-hide"
          aria-label="Navigation tabs"
        >
          {tabs.map((tab) => {
            const colors = getTabColors(tab.id);
            const isActive = activeTab === tab.id;

            return (
              <TabsPrimitive.Trigger
                key={tab.id}
                value={tab.id}
                className={cn(
                  "relative flex-1 sm:flex-none min-w-0 px-2 xs:px-3 sm:px-6 py-2 sm:py-3 rounded-lg text-xs xs:text-sm sm:text-base font-normal transition-all duration-200 whitespace-nowrap",
                  "focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-gray-100 dark:focus-visible:ring-offset-gray-800 focus-visible:ring-blue-500/30",
                  "disabled:pointer-events-none disabled:opacity-50",
                  isActive ? colors.active : colors.inactive
                )}
                title={tab.label} // Show full label on hover/long-press
              >
                <AnimatePresence>
                  {isActive && (
                    <motion.div
                      layoutId="activeTab"
                      className="absolute inset-0 bg-white/80 dark:bg-gray-800/80 rounded-lg shadow-sm border border-gray-200/40 dark:border-gray-700/80"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      transition={SPRING_TRANSITION}
                    />
                  )}
                </AnimatePresence>
                <span
                  className={cn(
                    "relative flex flex-col sm:flex-row items-center justify-center gap-0.5 xs:gap-1 sm:gap-3",
                    isActive ? colors.active : colors.inactive
                  )}
                >
                  <span className="flex-shrink-0">{tab.icon}</span>
                  {/* Desktop: show full label */}
                  <span className="hidden sm:inline">{tab.label}</span>
                  {/* Mobile: show abbreviated label with smaller text */}
                  <span className="inline sm:hidden text-[10px]">
                    {tab.shortLabel || getSmartAbbreviation(tab.label)}
                  </span>
                </span>
              </TabsPrimitive.Trigger>
            );
          })}
        </TabsPrimitive.List>
      </TabsPrimitive.Root>
    </div>
  );
};