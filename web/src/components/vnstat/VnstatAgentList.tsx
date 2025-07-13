/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  PencilIcon,
  TrashIcon,
  CheckCircleIcon,
  XCircleIcon,
  ServerIcon,
  StarIcon,
} from "@heroicons/react/24/outline";
import { StarIcon as StarIconSolid } from "@heroicons/react/24/solid";
import { VnstatAgent, VnstatStatus } from "@/api/vnstat";
import { useQuery } from "@tanstack/react-query";
import { getVnstatAgentStatus } from "@/api/vnstat";

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
            <ServerIcon className="mx-auto h-12 w-12 text-gray-400" />
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
  // Fetch agent status
  const { data: status } = useQuery<VnstatStatus>({
    queryKey: ["vnstat-agent-status", agent.id],
    queryFn: () => getVnstatAgentStatus(agent.id),
    refetchInterval: agent.enabled ? 5000 : false, // Poll every 5 seconds if enabled
    enabled: agent.enabled,
  });

  // Featured agents management
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
      JSON.stringify(ids)
    );
  };

  const featuredAgentIds = getFeaturedAgentIds();
  const isFeatured = featuredAgentIds.includes(agent.id);

  const handleToggleFeatured = (e: React.MouseEvent) => {
    e.stopPropagation();

    const currentFeatured = getFeaturedAgentIds();

    if (isFeatured) {
      // Remove from featured
      const newFeatured = currentFeatured.filter((id) => id !== agent.id);
      setFeaturedAgentIds(newFeatured);
    } else {
      // Add to featured (max 3)
      if (currentFeatured.length >= 3) {
        // Could show a toast notification here, but for now just ignore
        return;
      }
      const newFeatured = [...currentFeatured, agent.id];
      setFeaturedAgentIds(newFeatured);
    }
  };

  return (
    <motion.div
      className={`cursor-pointer p-4 transition-colors ${
        isSelected
          ? "bg-blue-50 dark:bg-blue-900/20"
          : "hover:bg-gray-50 dark:hover:bg-gray-700"
      }`}
      onClick={onSelect}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div
            className={`h-2 w-2 rounded-full ${
              !agent.enabled
                ? "bg-gray-400"
                : status?.connected
                ? "bg-green-500"
                : "bg-red-500"
            }`}
          />
          <div>
            <h4 className="font-medium text-gray-900 dark:text-white">
              {agent.name}
            </h4>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {agent.url.replace(/\/events\?stream=live-data$/, "")}
            </p>
            {agent.enabled && status?.liveData && (
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                ↓ {status.liveData.rx.ratestring} | ↑{" "}
                {status.liveData.tx.ratestring}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center space-x-2">
          {agent.enabled ? (
            <CheckCircleIcon className="h-5 w-5 text-green-500" />
          ) : (
            <XCircleIcon className="h-5 w-5 text-gray-400" />
          )}
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
              <StarIconSolid className="h-4 w-4" />
            ) : (
              <StarIcon className="h-4 w-4" />
            )}
          </button>
          <button
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            onClick={(e) => {
              e.stopPropagation();
              onEdit();
            }}
          >
            <PencilIcon className="h-4 w-4" />
          </button>
          <button
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
          >
            <TrashIcon className="h-4 w-4" />
          </button>
        </div>
      </div>
    </motion.div>
  );
};
