/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { ArrowDownIcon, ArrowUpIcon } from "@heroicons/react/24/outline";
import { useQuery } from "@tanstack/react-query";
import {
  getVnstatAgents,
  getVnstatAgentStatus,
  VnstatAgent,
  VnstatStatus,
} from "@/api/vnstat";
import { getAgentIcon } from "@/utils/agentIcons";

interface FeaturedVnstatWidgetProps {
  onNavigateToVnstat: () => void;
}

export const FeaturedVnstatWidget: React.FC<FeaturedVnstatWidgetProps> = ({
  onNavigateToVnstat,
}) => {
  // Get featured agent IDs from localStorage
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-vnstat-agents");
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  };

  const featuredAgentIds = getFeaturedAgentIds();

  // Fetch all agents
  const { data: allAgents = [] } = useQuery<VnstatAgent[]>({
    queryKey: ["vnstat-agents"],
    queryFn: getVnstatAgents,
  });

  // Filter to only featured agents that exist and are enabled
  const featuredAgents = allAgents.filter(
    (agent) => featuredAgentIds.includes(agent.id) && agent.enabled,
  );

  // If no featured agents, don't render the widget
  if (featuredAgents.length === 0) {
    return null;
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="mb-6"
    >
      <h2 className="text-gray-900 dark:text-white text-xl ml-1 font-semibold mb-4">
        Netronome Agents
      </h2>

      {/* Mobile: Stacked layout, Desktop: Grid layout */}
      <div className="space-y-3 md:space-y-0 md:grid md:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 md:gap-4">
        {featuredAgents.map((agent) => (
          <FeaturedAgentCard
            key={agent.id}
            agent={agent}
            onNavigateToVnstat={onNavigateToVnstat}
          />
        ))}
      </div>
    </motion.div>
  );
};

interface FeaturedAgentCardProps {
  agent: VnstatAgent;
  onNavigateToVnstat: () => void;
}

const FeaturedAgentCard: React.FC<FeaturedAgentCardProps> = ({
  agent,
  onNavigateToVnstat,
}) => {
  // Fetch agent status with polling
  const { data: status } = useQuery<VnstatStatus>({
    queryKey: ["vnstat-agent-status", agent.id],
    queryFn: () => getVnstatAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 5000 : false, // Poll every 5 seconds if enabled
    enabled: agent.enabled,
  });

  return (
    <motion.div
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      onClick={onNavigateToVnstat}
      className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-4 shadow-lg border border-gray-200 dark:border-gray-800 cursor-pointer transition-all duration-200 hover:shadow-xl hover:border-gray-300 dark:hover:border-gray-700"
    >
      {/* Header with agent name and status */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-2 min-w-0 flex-1">
          <div
            className={`h-2 w-2 rounded-full flex-shrink-0 ${
              status?.connected ? "bg-green-500" : "bg-red-500"
            }`}
          />
          <h3 className="font-medium text-gray-900 dark:text-white truncate text-sm">
            {agent.name}
          </h3>
        </div>
        {(() => {
          const { icon: Icon } = getAgentIcon(agent.name);
          return <Icon className="h-4 w-4 text-gray-400 flex-shrink-0" />;
        })()}
      </div>

      {/* Bandwidth data */}
      {status?.liveData ? (
        <div className="grid grid-cols-2 gap-3">
          {/* Download */}
          <div className="flex items-center space-x-2">
            <div className="rounded-full bg-blue-100 p-1.5 dark:bg-blue-900/20 flex-shrink-0">
              <ArrowDownIcon className="h-3 w-3 text-blue-600 dark:text-blue-400" />
            </div>
            <div className="flex-1">
              <p className="text-xs text-gray-500 dark:text-gray-400">Down</p>
              <p className="text-sm font-bold text-gray-900 dark:text-white">
                {status.liveData.rx.ratestring}
              </p>
            </div>
          </div>

          {/* Upload */}
          <div className="flex items-center space-x-2">
            <div className="rounded-full bg-green-100 p-1.5 dark:bg-green-900/20 flex-shrink-0">
              <ArrowUpIcon className="h-3 w-3 text-green-600 dark:text-green-400" />
            </div>
            <div className="flex-1">
              <p className="text-xs text-gray-500 dark:text-gray-400">Up</p>
              <p className="text-sm font-bold text-gray-900 dark:text-white">
                {status.liveData.tx.ratestring}
              </p>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center py-2">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {status?.connected === false ? "Disconnected" : "No data"}
          </p>
        </div>
      )}
    </motion.div>
  );
};
