/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";

interface Tab {
  id: string;
  label: string;
  icon?: React.ReactNode;
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

export const TabNavigation: React.FC<TabNavigationProps> = ({
  tabs,
  activeTab,
  onTabChange,
}) => {
  return (
    <div className="flex items-center justify-center w-full px-2 sm:px-0">
      <nav
        className="flex space-x-1 sm:space-x-2 bg-gray-100/60 dark:bg-gray-800/20 p-1 sm:p-2 rounded-xl shadow-sm border border-gray-200/60 dark:border-gray-800/80 max-w-full sm:w-auto"
        role="tablist"
      >
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id;
          const colors = getTabColors(tab.id);

          return (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`relative flex-1 sm:flex-none px-2 sm:px-6 py-2 sm:py-3 rounded-lg text-sm sm:text-base font-normal transition-all duration-200 ${
                isActive ? colors.active : colors.inactive
              }`}
              type="button"
              aria-pressed={isActive}
              role="tab"
              aria-selected={isActive}
            >
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
              <span
                className={`relative flex items-center gap-2 sm:gap-3 ${
                  isActive ? colors.active : colors.inactive
                }`}
              >
                {tab.icon}
                <span className="hidden sm:inline">{tab.label}</span>
                <span className="sm:hidden">{tab.label.split(" ")[0]}</span>
              </span>
            </button>
          );
        })}
      </nav>
    </div>
  );
};
