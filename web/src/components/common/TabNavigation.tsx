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
    active: "text-blue-400",
    inactive: "text-gray-400 hover:text-blue-300",
  },
  speedtest: {
    active: "text-emerald-400",
    inactive: "text-gray-400 hover:text-emerald-300",
  },
  traceroute: {
    active: "text-amber-400",
    inactive: "text-gray-400 hover:text-amber-300",
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
    <div className="flex items-center justify-center w-full">
      <nav 
        className="flex space-x-2 bg-gray-900/50 p-2 rounded-xl shadow-inner border border-gray-800/50"
        role="tablist"
      >
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id;
          const colors = getTabColors(tab.id);
          
          return (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`relative px-6 py-3 rounded-lg text-base font-medium transition-all duration-200 ${
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
                  className="absolute inset-0 bg-gray-800/80 rounded-lg shadow-lg border border-gray-700/50"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={SPRING_TRANSITION}
                />
              )}
              <span 
                className={`relative flex items-center gap-3 ${
                  isActive ? colors.active : colors.inactive
                }`}
              >
                {tab.icon}
                {tab.label}
              </span>
            </button>
          );
        })}
      </nav>
    </div>
  );
};