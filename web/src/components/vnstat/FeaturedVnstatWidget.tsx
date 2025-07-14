/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo } from "react";
import { motion } from "motion/react";
import { ServerIcon } from "@heroicons/react/24/outline";
import { FaArrowDown, FaArrowUp } from "react-icons/fa";
import { useQuery } from "@tanstack/react-query";
import {
  getVnstatAgents,
  getVnstatAgentStatus,
  getVnstatAgentBandwidth,
  getVnstatAgentUsage,
  VnstatAgent,
  VnstatStatus,
  VnstatBandwidth,
} from "@/api/vnstat";
import { getAgentIcon } from "@/utils/agentIcons";
import { MiniSparkline } from "./MiniSparkline";
import { formatBytes } from "@/utils/formatBytes";

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

  // Fetch bandwidth history for sparkline (last hour)
  const { data: bandwidthHistory = [] } = useQuery<VnstatBandwidth[]>({
    queryKey: ["vnstat-agent-bandwidth", agent.id, "1h"],
    queryFn: () =>
      getVnstatAgentBandwidth(agent.id, {
        limit: 60, // 1 minute intervals for 1 hour
      }),
    refetchInterval: agent.enabled ? 30000 : false, // Poll every 30 seconds
    enabled: agent.enabled,
  });

  // Fetch usage data
  const { data: usage } = useQuery({
    queryKey: ["vnstat-agent-usage", agent.id],
    queryFn: () => getVnstatAgentUsage(agent.id),
    refetchInterval: agent.enabled ? 60000 : false, // Poll every minute
    enabled: agent.enabled,
  });

  // Prepare sparkline data
  const sparklineData = useMemo(() => {
    if (!bandwidthHistory || bandwidthHistory.length === 0) {
      return { download: [], upload: [] };
    }

    // Reverse to get chronological order and extract speeds
    const reversed = [...bandwidthHistory].reverse();
    return {
      download: reversed.map((item) => item.rxBytesPerSecond || 0),
      upload: reversed.map((item) => item.txBytesPerSecond || 0),
    };
  }, [bandwidthHistory]);

  return (
    <motion.div
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      onClick={onNavigateToVnstat}
      className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-4 shadow-lg border border-gray-200 dark:border-gray-800 cursor-pointer transition-all duration-200 hover:shadow-xl hover:border-gray-300 dark:hover:border-gray-700"
    >
      {/* Header with agent name, status and icon */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-2 min-w-0 flex-1">
          <motion.div
            animate={{
              scale: status?.connected ? [1, 1.2, 1] : 1,
            }}
            transition={{
              duration: 2,
              repeat: status?.connected ? Infinity : 0,
              ease: "easeInOut",
            }}
            className={`h-2.5 w-2.5 rounded-full flex-shrink-0 ${
              status?.connected
                ? "bg-green-500 shadow-green-500/50 shadow-sm"
                : "bg-red-500"
            }`}
          />
          <h3 className="font-semibold text-gray-900 dark:text-white truncate text-base">
            {agent.name}
          </h3>
        </div>
        {(() => {
          const { icon: Icon } = getAgentIcon(agent.name);
          return <Icon className="h-5 w-5 text-gray-400 flex-shrink-0" />;
        })()}
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

          {/* Sparkline charts */}
          {sparklineData.download.length > 1 && (
            <div className="mb-4">
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Last Hour Activity
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-2 flex items-center justify-center">
                  <div className="w-full">
                    <MiniSparkline
                      data={sparklineData.download}
                      color="blue"
                      height={40}
                    />
                  </div>
                </div>
                <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-2 flex items-center justify-center">
                  <div className="w-full">
                    <MiniSparkline
                      data={sparklineData.upload}
                      color="green"
                      height={40}
                    />
                  </div>
                </div>
              </div>
            </div>
          )}

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
          <ServerIcon className="h-8 w-8 text-gray-400 mx-auto mb-2" />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {status?.connected === false ? "Disconnected" : "No data"}
          </p>
        </div>
      )}
    </motion.div>
  );
};
