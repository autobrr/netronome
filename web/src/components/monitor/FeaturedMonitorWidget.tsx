/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion } from "motion/react";
import {
  ServerIcon,
  CpuChipIcon,
  ArrowTopRightOnSquareIcon,
} from "@heroicons/react/24/outline";
import { FaArrowDown, FaArrowUp } from "react-icons/fa";
import { useQuery } from "@tanstack/react-query";
import { getMonitorAgents, MonitorAgent } from "@/api/monitor";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import { getAgentIcon } from "@/utils/agentIcons";
import { formatBytes } from "@/utils/formatBytes";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { MonitorUsageModal } from "./MonitorUsageModal";
import { MONITOR_REFRESH_INTERVALS } from "@/constants/monitorRefreshIntervals";
import { TailscaleLogo } from "@/components/icons/TailscaleLogo";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface FeaturedMonitorWidgetProps {
  onNavigateToMonitor: (agentId?: number) => void;
}

export const FeaturedMonitorWidget: React.FC<FeaturedMonitorWidgetProps> = ({
  onNavigateToMonitor,
}) => {
  const [selectedAgent, setSelectedAgent] = useState<MonitorAgent | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  // Get featured agent IDs from localStorage
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-monitor-agents");
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  };

  const featuredAgentIds = getFeaturedAgentIds();

  // Fetch all agents
  const { data: allAgents = [] } = useQuery<MonitorAgent[]>({
    queryKey: ["monitor-agents"],
    queryFn: getMonitorAgents,
    refetchInterval: MONITOR_REFRESH_INTERVALS.AGENTS_LIST,
  });

  // Filter to only featured agents that exist and are enabled
  const featuredAgents = allAgents.filter(
    (agent) => featuredAgentIds.includes(agent.id) && agent.enabled
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
            onOpenModal={() => {
              setSelectedAgent(agent);
              setIsModalOpen(true);
            }}
            onNavigateToAgent={() => onNavigateToMonitor(agent.id)}
          />
        ))}
      </div>

      {/* Usage Modal */}
      {selectedAgent && (
        <MonitorUsageModal
          isOpen={isModalOpen}
          onClose={() => {
            setIsModalOpen(false);
            // Clear selected agent after animation completes
            setTimeout(() => setSelectedAgent(null), 200);
          }}
          agent={selectedAgent}
        />
      )}
    </motion.div>
  );
};

interface FeaturedAgentCardProps {
  agent: MonitorAgent;
  onOpenModal: () => void;
  onNavigateToAgent: () => void;
  onHover?: () => void;
}

const FeaturedAgentCard: React.FC<FeaturedAgentCardProps> = ({
  agent,
  onOpenModal,
  onNavigateToAgent,
}) => {
  // Use the shared hook for agent data
  const { status, nativeData, hardwareStats } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeHardwareStats: true, // Now efficiently cached and shared
  });

  const usage = nativeData ? parseMonitorUsagePeriods(nativeData) : null;

  return (
    <motion.div
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      onClick={onOpenModal}
      className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl px-4 pb-2 pt-4 shadow-lg border border-gray-200 dark:border-gray-800 cursor-pointer transition-all duration-200 hover:shadow-xl hover:border-gray-300 dark:hover:border-gray-700/60 relative group"
    >
      {/* Header with agent name, status and icon */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-2 min-w-0 flex-1">
          {status?.connected ? (
            <span className="relative inline-flex h-2.5 w-2.5 flex-shrink-0">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500"></span>
            </span>
          ) : (
            <div className="h-2.5 w-2.5 rounded-full flex-shrink-0 bg-red-500" />
          )}
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <h3 className="font-semibold text-gray-900 dark:text-white truncate text-base">
              {agent.name}
            </h3>
            {/* Tailscale indicator */}
            {agent.isTailscale && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <div>
                    <TailscaleLogo className="h-4 w-4 flex-shrink-0" title="" />
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Connected through Tailscale</p>
                </TooltipContent>
              </Tooltip>
            )}
            {/* System Stats */}
            {hardwareStats && (
              <div className="flex items-center gap-1">
                <CpuChipIcon className="h-3 w-3 text-gray-400" />
                <span className="text-xs text-gray-600 dark:text-gray-400">
                  CPU: {hardwareStats.cpu.usage_percent.toFixed(0)}% RAM:{" "}
                  {hardwareStats.memory.used_percent.toFixed(0)}%
                </span>
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {(() => {
            const { icon: Icon } = getAgentIcon(agent.name);
            return <Icon className="h-5 w-5 text-gray-400 flex-shrink-0" />;
          })()}
          {/* Navigate to agent button */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onNavigateToAgent();
                }}
                className="p-1.5 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors duration-200"
              >
                <ArrowTopRightOnSquareIcon className="h-4 w-4" />
              </button>
            </TooltipTrigger>
            <TooltipContent>
              <p>View detailed stats</p>
            </TooltipContent>
          </Tooltip>
        </div>
      </div>

      {/* Bandwidth data */}
      {status?.liveData ? (
        <>
          {/* Current speeds with larger text */}
          <div className="grid grid-cols-2 gap-4 mb-4">
            {/* Download */}
            <div className="flex items-center space-x-3">
              <div className="rounded-full bg-blue-100 p-2 dark:bg-blue-900/20 flex-shrink-0">
                <FaArrowDown className="h-4 w-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="flex-1">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Download
                </p>
                <p className="text-lg font-bold text-gray-900 dark:text-white">
                  {status.liveData.rx.ratestring}
                </p>
              </div>
            </div>

            {/* Upload */}
            <div className="flex items-center space-x-3">
              <div className="rounded-full bg-green-100 p-2 dark:bg-green-900/20 flex-shrink-0">
                <FaArrowUp className="h-4 w-4 text-green-600 dark:text-green-400" />
              </div>
              <div className="flex-1">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Upload
                </p>
                <p className="text-lg font-bold text-gray-900 dark:text-white">
                  {status.liveData.tx.ratestring}
                </p>
              </div>
            </div>
          </div>

          {/* Today's usage */}
          {usage && (
            <div className="border-t border-gray-200 dark:border-gray-800 pt-3">
              <div className="flex items-center justify-between mb-1">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Today's Usage
                </p>
                <p className="text-xs font-medium text-gray-700 dark:text-gray-300">
                  {formatBytes(usage["Today"]?.total || 0)}
                </p>
              </div>
              <div className="flex items-center justify-end space-x-2">
                <div className="flex items-center space-x-1">
                  <FaArrowDown className="h-3 w-3 text-blue-500" />
                  <span className="text-xs text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["Today"]?.download || 0)}
                  </span>
                </div>
                <div className="flex items-center space-x-1">
                  <FaArrowUp className="h-3 w-3 text-green-500" />
                  <span className="text-xs text-gray-600 dark:text-gray-400">
                    {formatBytes(usage["Today"]?.upload || 0)}
                  </span>
                </div>
              </div>
            </div>
          )}
        </>
      ) : (
        <div className="text-center py-8">
          <ServerIcon className="h-8 w-8 text-gray-400 mx-auto" />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {status?.connected === false ? "Disconnected" : "No data"}
          </p>
        </div>
      )}
    </motion.div>
  );
};
