/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { MonitorAgentList } from "./MonitorAgentList";
import { MonitorAgentForm } from "./MonitorAgentForm";
import { MonitorAgentDetailsTabs } from "./MonitorAgentDetailsTabs";
import { ArrowLeftIcon } from "@heroicons/react/24/outline";
import {
  getMonitorAgents,
  createMonitorAgent,
  updateMonitorAgent,
  deleteMonitorAgent,
  MonitorAgent,
  CreateAgentRequest,
  UpdateAgentRequest,
} from "@/api/monitor";
import { MONITOR_REFRESH_INTERVALS } from "@/constants/monitorRefreshIntervals";

export const MonitorTab: React.FC = () => {
  const [selectedAgent, setSelectedAgent] = useState<MonitorAgent | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingAgent, setEditingAgent] = useState<MonitorAgent | null>(null);
  const queryClient = useQueryClient();

  // Fetch agents
  const { data: agents = [], isLoading } = useQuery({
    queryKey: ["monitor-agents"],
    queryFn: getMonitorAgents,
    refetchInterval: MONITOR_REFRESH_INTERVALS.AGENTS_LIST,
  });

  // Note: Agent status is fetched by child components using useMonitorAgent hook
  // This prevents duplicate API calls

  // Handle pre-selected agent from navigation
  React.useEffect(() => {
    const preselectedId = sessionStorage.getItem("netronome-preselect-agent");
    if (preselectedId && agents.length > 0 && !selectedAgent) {
      const agentId = parseInt(preselectedId, 10);
      const agent = agents.find(a => a.id === agentId);
      if (agent) {
        setSelectedAgent(agent);
        sessionStorage.removeItem("netronome-preselect-agent");
      }
    }
  }, [agents, selectedAgent]);

  // Create agent mutation
  const createMutation = useMutation({
    mutationFn: createMonitorAgent,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      setIsFormOpen(false);
    },
  });

  // Update agent mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateAgentRequest }) =>
      updateMonitorAgent(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      setIsFormOpen(false);
      setEditingAgent(null);
    },
  });

  // Delete agent mutation
  const deleteMutation = useMutation({
    mutationFn: deleteMonitorAgent,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      setSelectedAgent(null);
    },
  });

  const handleCreateAgent = () => {
    setEditingAgent(null);
    setIsFormOpen(true);
  };

  const handleEditAgent = (agent: MonitorAgent) => {
    setEditingAgent(agent);
    setIsFormOpen(true);
  };

  const handleDeleteAgent = async (id: number) => {
    if (confirm("Are you sure you want to delete this agent?")) {
      deleteMutation.mutate(id);
    }
  };

  const handleFormSubmit = (data: CreateAgentRequest) => {
    if (editingAgent) {
      updateMutation.mutate({
        id: editingAgent.id,
        data: { ...data, id: editingAgent.id },
      });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleBack = () => {
    setSelectedAgent(null);
  };

  return (
    <div className="space-y-6">
      <AnimatePresence mode="wait">
        {!selectedAgent ? (
          <motion.div
            key="agent-list"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="space-y-6"
          >
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
              <div>
                <h2 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">
                  Bandwidth Monitoring
                </h2>
                <p className="mt-1 text-xs sm:text-sm text-gray-600 dark:text-gray-400">
                  Monitor bandwidth from Netronome agents • Click an agent to view detailed stats
                </p>
              </div>
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                onClick={handleCreateAgent}
                className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white border border-blue-600 hover:border-blue-700 rounded-lg shadow-md transition-colors flex items-center gap-2 text-sm sm:text-base font-medium"
              >
                Add Agent
              </motion.button>
            </div>

            {/* Agent List */}
            <MonitorAgentList
              agents={agents}
              selectedAgent={null}
              onSelectAgent={setSelectedAgent}
              onEditAgent={handleEditAgent}
              onDeleteAgent={handleDeleteAgent}
              isLoading={isLoading}
            />
          </motion.div>
        ) : (
          <motion.div
            key="agent-details"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
            className="space-y-6"
          >
            {/* Header with back button */}
            <div className="flex items-center gap-4">
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                onClick={handleBack}
                className="px-3 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-lg shadow-md transition-colors flex items-center gap-2"
              >
                <ArrowLeftIcon className="h-4 w-4" />
                <span className="text-sm font-medium">Back</span>
              </motion.button>
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <h2 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">
                    {selectedAgent.name}
                  </h2>
                  {/* Status badge will be added by child components that have access to real-time status */}
                  {selectedAgent && !selectedAgent.enabled && (
                    <span className="px-3 py-1 bg-gray-500/10 border border-gray-500/30 text-gray-600 dark:text-gray-400 rounded-lg shadow-md text-xs font-medium">
                      Disabled
                    </span>
                  )}
                </div>
                <p className="mt-1 text-xs sm:text-sm text-gray-600 dark:text-gray-400">
                  {selectedAgent.url}
                </p>
              </div>
              <div className="flex gap-2">
                <motion.button
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  onClick={() => handleEditAgent(selectedAgent)}
                  className="px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-lg shadow-md transition-colors text-sm font-medium"
                >
                  Edit
                </motion.button>
                <motion.button
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  onClick={() => handleDeleteAgent(selectedAgent.id)}
                  className="px-4 py-2 bg-red-500/10 border border-red-500/30 hover:bg-red-500/20 text-red-600 dark:text-red-400 rounded-lg shadow-md transition-colors text-sm font-medium"
                >
                  Delete
                </motion.button>
              </div>
            </div>

            {/* Tabbed Details View */}
            <MonitorAgentDetailsTabs agent={selectedAgent} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Agent Form Modal */}
      <MonitorAgentForm
        agent={editingAgent}
        onSubmit={handleFormSubmit}
        onCancel={() => {
          setIsFormOpen(false);
          setEditingAgent(null);
        }}
        isSubmitting={createMutation.isPending || updateMutation.isPending}
        isOpen={isFormOpen}
      />
    </div>
  );
};