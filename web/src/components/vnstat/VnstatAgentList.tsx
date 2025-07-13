/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { PencilIcon, TrashIcon, StarIcon } from "@heroicons/react/24/outline";
import { StarIcon as StarIconSolid } from "@heroicons/react/24/solid";
import { VnstatAgent, VnstatStatus } from "@/api/vnstat";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getVnstatAgentStatus,
  startVnstatAgent,
  stopVnstatAgent,
} from "@/api/vnstat";
import { Button } from "@/components/ui/Button";
import { AgentIcon } from "@/utils/agentIcons";

interface VnstatAgentListProps {
  agents: VnstatAgent[];
  selectedAgent: VnstatAgent | null;
  onSelectAgent: (agent: VnstatAgent) => void;
  onEditAgent: (agent: VnstatAgent) => void;
  onDeleteAgent: (id: number) => void;
  isLoading: boolean;
}

export const VnstatAgentList: React.FC<VnstatAgentListProps> = ({
  agents,
  selectedAgent,
  onSelectAgent,
  onEditAgent,
  onDeleteAgent,
  isLoading,
}) => {
  return (
    <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800">
      <div className="border-b border-gray-200 px-6 py-4 dark:border-gray-700">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white">
          Netronome Agents
        </h3>
      </div>

      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        {isLoading ? (
          <div className="p-6 text-center">
            <p className="text-gray-500 dark:text-gray-400">
              Loading agents...
            </p>
          </div>
        ) : !agents || agents.length === 0 ? (
          <div className="p-6 text-center">
            <AgentIcon
              name="Server"
              className="mx-auto h-12 w-12 text-gray-400"
            />
            <p className="mt-2 text-gray-500 dark:text-gray-400">
              No agents configured
            </p>
            <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">
              Add an agent to start monitoring bandwidth
            </p>
          </div>
        ) : (
          agents?.map((agent) => (
            <AgentListItem
              key={agent.id}
              agent={agent}
              isSelected={selectedAgent?.id === agent.id}
              onSelect={() => onSelectAgent(agent)}
              onEdit={() => onEditAgent(agent)}
              onDelete={() => onDeleteAgent(agent.id)}
            />
          ))
        )}
      </div>
    </div>
  );
};

interface AgentListItemProps {
  agent: VnstatAgent;
  isSelected: boolean;
  onSelect: () => void;
  onEdit: () => void;
  onDelete: () => void;
}

const AgentListItem: React.FC<AgentListItemProps> = ({
  agent,
  isSelected,
  onSelect,
  onEdit,
  onDelete,
}) => {
  const queryClient = useQueryClient();

  // Fetch agent status
  const { data: status } = useQuery<VnstatStatus>({
    queryKey: ["vnstat-agent-status", agent.id],
    queryFn: () => getVnstatAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 5000 : false, // Poll every 5 seconds if enabled
    enabled: agent.enabled,
  });

  // Start/stop mutations
  const startMutation = useMutation({
    mutationFn: () => startVnstatAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["vnstat-agent-status", agent.id],
      });
    },
  });

  const stopMutation = useMutation({
    mutationFn: () => stopVnstatAgent(agent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      queryClient.invalidateQueries({
        queryKey: ["vnstat-agent-status", agent.id],
      });
    },
  });

  const handleToggleMonitoring = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (agent.enabled && status?.connected) {
      stopMutation.mutate();
    } else {
      startMutation.mutate();
    }
  };

  // Featured agents management with local state for instant feedback
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-vnstat-agents");
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  };

  const setFeaturedAgentIds = (ids: number[]) => {
    localStorage.setItem(
      "netronome-featured-vnstat-agents",
      JSON.stringify(ids),
    );
  };

  // Use local state for immediate UI feedback
  const [isFeatured, setIsFeatured] = React.useState(() => {
    const featuredIds = getFeaturedAgentIds();
    return featuredIds.includes(agent.id);
  });

  const handleToggleFeatured = (e: React.MouseEvent) => {
    e.stopPropagation();

    const currentFeatured = getFeaturedAgentIds();

    if (isFeatured) {
      // Remove from featured
      const newFeatured = currentFeatured.filter((id) => id !== agent.id);
      setFeaturedAgentIds(newFeatured);
      setIsFeatured(false); // Immediate UI update
    } else {
      // Add to featured (max 3)
      if (currentFeatured.length >= 3) {
        // Could show a toast notification here, but for now just ignore
        return;
      }
      const newFeatured = [...currentFeatured, agent.id];
      setFeaturedAgentIds(newFeatured);
      setIsFeatured(true); // Immediate UI update
    }
  };

  return (
    <motion.div
      className={`cursor-pointer p-6 transition-colors ${
        isSelected
          ? "bg-blue-50 dark:bg-gray-800"
          : "hover:bg-gray-50 dark:hover:bg-gray-800"
      }`}
      onClick={onSelect}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start space-x-4 flex-1">
          <AgentIcon
            name={agent.name}
            className="h-10 w-10 text-gray-400 mt-1"
          />
          <div className="flex-1 min-w-0">
            <div className="flex items-center space-x-3 mb-2">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                {agent.name}
              </h3>
              <div
                className={`h-2 w-2 rounded-full ${
                  !agent.enabled
                    ? "bg-gray-400"
                    : status?.connected
                      ? "bg-green-500"
                      : "bg-red-500"
                }`}
              />
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {!agent.enabled
                  ? "Disabled"
                  : status?.connected
                    ? "Connected"
                    : "Disconnected"}
              </span>
            </div>

            <div className="space-y-1">
              <p className="text-sm text-gray-500 dark:text-gray-400">
                <span className="font-medium">URL:</span>{" "}
                {agent.url.replace(/\/events\?stream=live-data$/, "")}
              </p>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                <span className="font-medium">Retention:</span>{" "}
                {agent.retentionDays || 365} days
              </p>
              {agent.enabled && status?.liveData && (
                <div className="flex items-center space-x-4 mt-2">
                  <span className="text-sm text-gray-700 dark:text-gray-300">
                    <span className="font-medium">↓</span>{" "}
                    {status.liveData.rx.ratestring}
                  </span>
                  <span className="text-sm text-gray-700 dark:text-gray-300">
                    <span className="font-medium">↑</span>{" "}
                    {status.liveData.tx.ratestring}
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center space-x-2 ml-4">
          <button
            className={`p-2 rounded-md transition-colors ${
              isFeatured
                ? "text-yellow-500 hover:text-yellow-600 dark:text-yellow-400 dark:hover:text-yellow-500"
                : "text-gray-500 hover:text-yellow-500 dark:text-gray-400 dark:hover:text-yellow-400"
            } hover:bg-gray-100 dark:hover:bg-gray-700`}
            onClick={handleToggleFeatured}
            title={isFeatured ? "Remove from featured" : "Add to featured"}
          >
            {isFeatured ? (
              <StarIconSolid className="h-5 w-5" />
            ) : (
              <StarIcon className="h-5 w-5" />
            )}
          </button>
          <button
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            onClick={(e) => {
              e.stopPropagation();
              onEdit();
            }}
          >
            <PencilIcon className="h-5 w-5" />
          </button>
          <button
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
          >
            <TrashIcon className="h-5 w-5" />
          </button>
          <Button
            onClick={handleToggleMonitoring}
            disabled={startMutation.isPending || stopMutation.isPending}
            className={
              agent.enabled && status?.connected
                ? "bg-red-500/10 hover:bg-red-500/20 text-red-600 dark:text-red-400 border-red-500/20"
                : "bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
            }
          >
            {agent.enabled && status?.connected ? "Stop" : "Start"}
          </Button>
        </div>
      </div>
    </motion.div>
  );
};
