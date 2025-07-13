/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion } from "motion/react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { VnstatAgentList } from "./VnstatAgentList";
import { VnstatAgentForm } from "./VnstatAgentForm";
import { VnstatAgentDetails } from "./VnstatAgentDetails";
import {
  getVnstatAgents,
  createVnstatAgent,
  updateVnstatAgent,
  deleteVnstatAgent,
  VnstatAgent,
  CreateAgentRequest,
  UpdateAgentRequest,
} from "@/api/vnstat";

export const VnstatTab: React.FC = () => {
  const [selectedAgent, setSelectedAgent] = useState<VnstatAgent | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingAgent, setEditingAgent] = useState<VnstatAgent | null>(null);
  const queryClient = useQueryClient();

  // Fetch agents
  const { data: agents = [], isLoading } = useQuery({
    queryKey: ["vnstat-agents"],
    queryFn: getVnstatAgents,
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  // Create agent mutation
  const createMutation = useMutation({
    mutationFn: createVnstatAgent,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      setIsFormOpen(false);
    },
  });

  // Update agent mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateAgentRequest }) =>
      updateVnstatAgent(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      setIsFormOpen(false);
      setEditingAgent(null);
    },
  });

  // Delete agent mutation
  const deleteMutation = useMutation({
    mutationFn: deleteVnstatAgent,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vnstat-agents"] });
      setSelectedAgent(null);
    },
  });

  const handleCreateAgent = () => {
    setEditingAgent(null);
    setIsFormOpen(true);
  };

  const handleEditAgent = (agent: VnstatAgent) => {
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            Bandwidth Monitoring
          </h2>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Monitor network bandwidth from remote vnstat agents
          </p>
        </div>
        <Button
          onClick={handleCreateAgent}
          className="flex items-center gap-2 bg-blue-500 hover:bg-blue-600 text-white border-blue-600"
        >
          Add Agent
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Agent List */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <VnstatAgentList
            agents={agents}
            selectedAgent={selectedAgent}
            onSelectAgent={setSelectedAgent}
            onEditAgent={handleEditAgent}
            onDeleteAgent={handleDeleteAgent}
            isLoading={isLoading}
          />
        </motion.div>

        {/* Agent Details */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
        >
          {selectedAgent ? (
            <VnstatAgentDetails agent={selectedAgent} />
          ) : (
            <div className="rounded-lg bg-white p-6 text-center shadow-sm dark:bg-gray-800">
              <p className="text-gray-500 dark:text-gray-400">
                Select an agent to view bandwidth details
              </p>
            </div>
          )}
        </motion.div>
      </div>

      {/* Agent Form Modal */}
      {isFormOpen && (
        <VnstatAgentForm
          agent={editingAgent}
          onSubmit={handleFormSubmit}
          onCancel={() => {
            setIsFormOpen(false);
            setEditingAgent(null);
          }}
          isSubmitting={createMutation.isPending || updateMutation.isPending}
        />
      )}
    </div>
  );
};
