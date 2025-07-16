/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { MonitorAgent } from "@/api/monitor";
import { MonitorOverviewTab } from "./tabs/MonitorOverviewTab";
import { MonitorBandwidthTab } from "./tabs/MonitorBandwidthTab";
import { MonitorSystemTab } from "./tabs/MonitorSystemTab";

interface MonitorAgentDetailsTabsProps {
  agent: MonitorAgent;
}

type TabType = "overview" | "bandwidth" | "system";

interface Tab {
  id: TabType;
  label: string;
  component: React.ComponentType<{ agent: MonitorAgent }>;
}

const tabs: Tab[] = [
  { id: "overview", label: "Overview", component: MonitorOverviewTab },
  { id: "bandwidth", label: "Bandwidth", component: MonitorBandwidthTab },
  { id: "system", label: "System & Hardware", component: MonitorSystemTab },
];

export const MonitorAgentDetailsTabs: React.FC<MonitorAgentDetailsTabsProps> = ({
  agent,
}) => {
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  const ActiveComponent = tabs.find((tab) => tab.id === activeTab)?.component;

  return (
    <div className="space-y-6">
      {/* Tab Navigation */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`
                whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm
                transition-colors duration-200
                ${
                  activeTab === tab.id
                    ? "border-blue-500 text-blue-600 dark:text-blue-400"
                    : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300"
                }
              `}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <AnimatePresence mode="wait">
        <motion.div
          key={activeTab}
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -10 }}
          transition={{ duration: 0.2 }}
        >
          {ActiveComponent && <ActiveComponent agent={agent} />}
        </motion.div>
      </AnimatePresence>
    </div>
  );
};