/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  PencilIcon,
  TrashIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";
import { SparklesIcon as SparklesIconSolid } from "@heroicons/react/24/solid";
import { VnstatAgent } from "@/api/vnstat";
import { AgentIcon } from "@/utils/agentIcons";
import { useVnstatAgent } from "@/hooks/useVnstatAgent";

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
  // Force re-render all items when featured agents change
  const [updateKey, setUpdateKey] = React.useState(0);

  React.useEffect(() => {
    const handleStorageChange = () => {
      setUpdateKey((prev) => prev + 1);
    };

    window.addEventListener("storage", handleStorageChange);
    window.addEventListener("featured-agents-changed", handleStorageChange);
    return () => {
      window.removeEventListener("storage", handleStorageChange);
      window.removeEventListener(
        "featured-agents-changed",
        handleStorageChange
      );
    };
  }, []);

  return (
    <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800">
      <div className="border-b border-gray-200 px-4 sm:px-6 py-2.5 dark:border-gray-800">
        <h3 className="text-base font-medium text-gray-900 dark:text-white">
          Netronome Agents
        </h3>
      </div>

      <div className="divide-y divide-gray-200 dark:divide-gray-800">
        {isLoading ? (
          <div className="p-4 text-center">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Loading agents...
            </p>
          </div>
        ) : !agents || agents.length === 0 ? (
          <div className="p-4 text-center">
            <AgentIcon
              name="Server"
              className="mx-auto h-10 w-10 text-gray-400"
            />
            <p className="mt-1.5 text-sm text-gray-500 dark:text-gray-400">
              No agents configured
            </p>
            <p className="text-xs text-gray-400 dark:text-gray-500">
              Add an agent to start monitoring bandwidth
            </p>
          </div>
        ) : (
          agents?.map((agent) => (
            <AgentListItem
              key={`${agent.id}-${updateKey}`}
              agent={agent}
              agents={agents}
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
  agents: VnstatAgent[];
  isSelected: boolean;
  onSelect: () => void;
  onEdit: () => void;
  onDelete: () => void;
}

const AgentListItem: React.FC<AgentListItemProps> = ({
  agent,
  agents,
  isSelected,
  onSelect,
  onEdit,
  onDelete,
}) => {
  const { status } = useVnstatAgent({ agent });

  // Featured agents management with local state for instant feedback
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-vnstat-agents");
      const parsed = stored ? JSON.parse(stored) : [];
      // Ensure it's always an array
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  };

  const setFeaturedAgentIds = (ids: number[]) => {
    try {
      // Ensure we're storing a valid array
      const validIds = Array.isArray(ids) ? ids : [];
      localStorage.setItem(
        "netronome-featured-vnstat-agents",
        JSON.stringify(validIds)
      );
    } catch {
      console.error("Error saving featured agents to localStorage");
    }
  };

  // Use local state for immediate UI feedback
  const [isFeatured, setIsFeatured] = React.useState(() => {
    const featuredIds = getFeaturedAgentIds();
    return featuredIds.includes(agent.id);
  });

  // Listen for changes to featured agents from other components or tabs
  React.useEffect(() => {
    const handleStorageChange = () => {
      const featuredIds = getFeaturedAgentIds();
      setIsFeatured(featuredIds.includes(agent.id));
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, [agent.id]);

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
      // Ensure currentFeatured is an array and filter out any non-numeric values
      const validCurrentFeatured = Array.isArray(currentFeatured)
        ? currentFeatured.filter((id) => typeof id === "number")
        : [];

      // Filter out agent IDs that no longer exist
      const existingAgentIds = agents.map((a) => a.id);
      const existingFeatured = validCurrentFeatured.filter((id) =>
        existingAgentIds.includes(id)
      );

      // Clean up localStorage if we removed any non-existent agents
      if (existingFeatured.length !== validCurrentFeatured.length) {
        setFeaturedAgentIds(existingFeatured);
      }

      if (existingFeatured.length >= 3) {
        alert(
          "You can only feature up to 3 agents at a time. Please unfeature an agent first."
        );
        return;
      }

      // Check if agent is already in the array (shouldn't happen but just in case)
      if (existingFeatured.includes(agent.id)) {
        setIsFeatured(true);
        return;
      }

      const newFeatured = [...existingFeatured, agent.id];
      setFeaturedAgentIds(newFeatured);
      setIsFeatured(true); // Immediate UI update
    }

    // Force re-render of other agent items to update their star state
    window.dispatchEvent(new Event("storage"));
    window.dispatchEvent(new Event("featured-agents-changed"));
  };

  return (
    <motion.div
      className={`cursor-pointer p-3 sm:p-4 transition-colors ${
        isSelected
          ? "bg-blue-50 dark:bg-gray-800"
          : "hover:bg-gray-50 dark:hover:bg-gray-800"
      }`}
      onClick={onSelect}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center space-x-3 flex-1 min-w-0">
          <AgentIcon
            name={agent.name}
            className="h-7 w-7 sm:h-8 sm:w-8 text-gray-400 flex-shrink-0"
          />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="text-sm sm:text-base font-medium text-gray-900 dark:text-white">
                {agent.name}
              </h3>
              <div className="flex items-center space-x-1.5">
                {agent.enabled && status?.connected ? (
                  <span className="relative inline-flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                  </span>
                ) : (
                  <div
                    className={`h-2 w-2 rounded-full ${
                      !agent.enabled ? "bg-gray-400" : "bg-red-500"
                    }`}
                  />
                )}
                <span className="text-xs text-gray-600 dark:text-gray-400">
                  {!agent.enabled
                    ? "Disabled"
                    : status?.connected
                    ? "Connected"
                    : "Disconnected"}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <span className="truncate">
                {agent.url.replace(/\/events\?stream=live-data$/, "")}
              </span>
            </div>
          </div>
        </div>

        <div className="flex flex-col items-end gap-1 flex-shrink-0">
          <div className="flex items-center gap-0.5">
            <button
              className={`p-1 rounded-md transition-colors ${
                isFeatured
                  ? "text-yellow-500 hover:text-yellow-600 dark:text-yellow-400 dark:hover:text-yellow-500"
                  : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              } hover:bg-gray-100 dark:hover:bg-gray-700`}
              onClick={handleToggleFeatured}
              title={isFeatured ? "Remove from featured" : "Add to featured"}
            >
              {isFeatured ? (
                <SparklesIconSolid className="w-3.5 h-3.5" />
              ) : (
                <SparklesIcon className="w-3.5 h-3.5" />
              )}
            </button>
            <button
              className="p-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                onEdit();
              }}
              title="Edit Agent"
            >
              <PencilIcon className="w-3.5 h-3.5" />
            </button>
            <button
              className="p-1 text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-200 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
              }}
              title="Delete Agent"
            >
              <TrashIcon className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>
    </motion.div>
  );
};
