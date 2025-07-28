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
import { MonitorDataPrefetcher } from "./MonitorDataPrefetcher";
import {
  ArrowLeftIcon,
  PencilIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { DeleteConfirmationDialog } from "@/components/common/DeleteConfirmationDialog";
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
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [agentToDelete, setAgentToDelete] = useState<MonitorAgent | null>(null);
  const queryClient = useQueryClient();

  // Helper functions for managing featured agents
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-monitor-agents");
      const parsed = stored ? JSON.parse(stored) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  };

  const setFeaturedAgentIds = (ids: number[]) => {
    try {
      const validIds = Array.isArray(ids) ? ids : [];
      localStorage.setItem(
        "netronome-featured-monitor-agents",
        JSON.stringify(validIds)
      );
    } catch {
      console.error("Error saving featured agents to localStorage");
    }
  };

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
      const agent = agents.find((a) => a.id === agentId);
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
    onSuccess: (_, variables) => {
      // If agent was disabled, remove it from featured agents
      if (!variables.data.enabled) {
        const featuredAgentIds = getFeaturedAgentIds();
        const updatedFeaturedIds = featuredAgentIds.filter(
          (featuredId) => featuredId !== variables.id
        );
        if (updatedFeaturedIds.length !== featuredAgentIds.length) {
          setFeaturedAgentIds(updatedFeaturedIds);
          // Trigger storage event for other components
          window.dispatchEvent(new Event("storage"));
          window.dispatchEvent(new Event("featured-agents-changed"));
        }
      }

      queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
      setIsFormOpen(false);
      setEditingAgent(null);
    },
  });

  // Delete agent mutation
  const deleteMutation = useMutation({
    mutationFn: deleteMonitorAgent,
    onSuccess: (_, deletedAgentId) => {
      // Remove deleted agent from featured agents
      const featuredAgentIds = getFeaturedAgentIds();
      const updatedFeaturedIds = featuredAgentIds.filter(
        (featuredId) => featuredId !== deletedAgentId
      );
      if (updatedFeaturedIds.length !== featuredAgentIds.length) {
        setFeaturedAgentIds(updatedFeaturedIds);
        // Trigger storage event for other components
        window.dispatchEvent(new Event("storage"));
        window.dispatchEvent(new Event("featured-agents-changed"));
      }

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

  const handleDeleteAgent = (agent: MonitorAgent) => {
    setAgentToDelete(agent);
    setDeleteDialogOpen(true);
  };

  const confirmDelete = () => {
    if (agentToDelete) {
      deleteMutation.mutate(agentToDelete.id);
      setDeleteDialogOpen(false);
      setAgentToDelete(null);
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
      {/* Pre-fetch data for all enabled agents */}
      <MonitorDataPrefetcher agents={agents} />
      
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
                  Monitoring
                </h2>
                <p className="mt-1 text-xs sm:text-sm text-gray-600 dark:text-gray-400">
                  Monitor bandwidth and other stats from Netronome agents •
                  Click an agent to view detailed stats
                </p>
              </div>
              <Button
                onClick={handleCreateAgent}
                variant="default"
                size="default"
              >
                Add Agent
              </Button>
            </div>

            {/* Agent List */}
            <MonitorAgentList
              agents={agents}
              selectedAgent={null}
              onSelectAgent={setSelectedAgent}
              onEditAgent={handleEditAgent}
              onDeleteAgent={(id) => {
                const agent = agents.find((a) => a.id === id);
                if (agent) handleDeleteAgent(agent);
              }}
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
            {/* Simplified Header */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Button
                  onClick={handleBack}
                  variant="secondary"
                  size="sm"
                  className="gap-1.5"
                >
                  <ArrowLeftIcon className="h-4 w-4" />
                  <span>Back</span>
                </Button>
                <h2 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">
                  {selectedAgent.name}
                </h2>
                {selectedAgent && !selectedAgent.enabled && (
                  <Badge variant="secondary">Disabled</Badge>
                )}
              </div>
              <div className="flex items-center gap-2">
                <Button
                  onClick={() => handleEditAgent(selectedAgent)}
                  variant="secondary"
                  size="sm"
                  className="gap-1.5"
                  title="Edit agent"
                >
                  <PencilIcon className="h-4 w-4" />
                  <span className="hidden sm:inline">Edit</span>
                </Button>
                <Button
                  onClick={() => handleDeleteAgent(selectedAgent)}
                  variant="secondary"
                  size="sm"
                  className="gap-1.5 hover:bg-red-100 dark:hover:bg-red-900/30 hover:text-red-600 dark:hover:text-red-400"
                  title="Delete agent"
                >
                  <TrashIcon className="h-4 w-4" />
                  <span className="hidden sm:inline">Delete</span>
                </Button>
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

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmationDialog
        isOpen={deleteDialogOpen}
        onClose={() => {
          setDeleteDialogOpen(false);
          setAgentToDelete(null);
        }}
        onConfirm={confirmDelete}
        itemName={agentToDelete?.name || ""}
        isDeleting={deleteMutation.isPending}
      />
    </div>
  );
};
