/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { motion } from "motion/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { PacketLossMonitor } from "@/types/types";
import { CrossTabNavigationHook } from "@/hooks/useCrossTabNavigation";
import {
  getPacketLossMonitors,
  createPacketLossMonitor,
  updatePacketLossMonitor,
  deletePacketLossMonitor,
  startPacketLossMonitor,
  stopPacketLossMonitor,
} from "@/api/packetloss";
import { PacketLossMonitorList } from "./PacketLossMonitorList";
import { PacketLossMonitorForm } from "./PacketLossMonitorForm";
import { PacketLossMonitorDetails } from "./PacketLossMonitorDetails";
import { EmptyStatePlaceholder } from "./components/EmptyStatePlaceholder";
import { usePacketLossMonitorStatus } from "./hooks/usePacketLossMonitorStatus";
import {
  MonitorFormData,
  defaultFormData,
} from "./constants/packetLossConstants";

interface PacketLossTabProps {
  crossTabNavigation?: CrossTabNavigationHook;
}

export const PacketLossTab: React.FC<PacketLossTabProps> = ({
  crossTabNavigation,
}) => {
  const queryClient = useQueryClient();
  const [selectedMonitor, setSelectedMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingMonitor, setEditingMonitor] =
    useState<PacketLossMonitor | null>(null);
  const [formData, setFormData] = useState<MonitorFormData>(defaultFormData);

  // Fetch monitors
  const { data: monitors, isLoading } = useQuery({
    queryKey: ["packetloss", "monitors"],
    queryFn: getPacketLossMonitors,
    staleTime: 30000, // Consider data stale after 30 seconds
  });

  // Ensure monitors is always an array
  const monitorList = monitors || [];

  // Use custom hook for monitor status polling
  const monitorStatuses = usePacketLossMonitorStatus(
    monitorList,
    selectedMonitor?.id,
  );

  // Handle cross-tab data consumption
  useEffect(() => {
    if (crossTabNavigation) {
      const crossTabData = crossTabNavigation.consumePacketLossData();
      if (crossTabData?.host) {
        setFormData((prev) => ({
          ...prev,
          host: crossTabData.host || "",
          name: crossTabData.name || `Monitor for ${crossTabData.host}`,
          enabled: true,
        }));
        if (crossTabData.fromTraceroute) {
          setShowForm(true);
        }
      }
    }
  }, [crossTabNavigation]);

  // Mutations
  const createMutation = useMutation({
    mutationFn: createPacketLossMonitor,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      handleCancelForm();
    },
  });

  const updateMutation = useMutation({
    mutationFn: updatePacketLossMonitor,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      handleCancelForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deletePacketLossMonitor,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      if (selectedMonitor?.id === deleteMutation.variables) {
        setSelectedMonitor(null);
      }
    },
  });

  const startMutation = useMutation({
    mutationFn: startPacketLossMonitor,
    onSuccess: async (_, monitorId) => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      queryClient.invalidateQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
    },
  });

  const stopMutation = useMutation({
    mutationFn: stopPacketLossMonitor,
    onSuccess: async (_, monitorId) => {
      queryClient.invalidateQueries({ queryKey: ["packetloss", "monitors"] });
      queryClient.invalidateQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "monitors"],
      });
      await queryClient.refetchQueries({
        queryKey: ["packetloss", "history", monitorId],
      });
    },
  });

  // Handlers
  const handleSubmit = (data: MonitorFormData) => {
    if (editingMonitor) {
      updateMutation.mutate({
        ...editingMonitor,
        ...data,
      });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleEdit = (monitor: PacketLossMonitor) => {
    setEditingMonitor(monitor);
    setFormData({
      host: monitor.host,
      name: monitor.name || "",
      interval: monitor.interval,
      packetCount: monitor.packetCount,
      threshold: monitor.threshold,
      enabled: monitor.enabled,
    });
    setShowForm(true);
  };

  const handleCancelForm = () => {
    setShowForm(false);
    setEditingMonitor(null);
    setFormData(defaultFormData);
  };

  const handleToggle = (monitorId: number, enabled: boolean) => {
    if (enabled) {
      startMutation.mutate(monitorId);
    } else {
      stopMutation.mutate(monitorId);
    }
  };

  const handleTraceRoute = () => {
    if (!selectedMonitor || !crossTabNavigation) return;

    crossTabNavigation.navigateToTraceroute({
      host: selectedMonitor.host,
      fromPacketLoss: true,
    });
  };

  return (
    <div className="flex flex-col md:flex-row gap-6 md:items-start">
      {/* Left Column - Monitor List */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="flex-1"
      >
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Packet Loss Monitors
              </h2>
              <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                Continuous monitoring with MTR or ICMP ping
              </p>
            </div>
            <Button
              onClick={() => setShowForm(true)}
              className="bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
            >
              Add Monitor
            </Button>
          </div>

          {/* Monitor List Component */}
          <PacketLossMonitorList
            monitors={monitorList}
            selectedMonitor={selectedMonitor}
            monitorStatuses={monitorStatuses}
            onMonitorSelect={setSelectedMonitor}
            onEdit={handleEdit}
            onDelete={(id) => deleteMutation.mutate(id)}
            onToggle={handleToggle}
            isLoading={isLoading}
            isToggling={startMutation.isPending || stopMutation.isPending}
          />
        </div>
      </motion.div>

      {/* Right Column - Monitor Details or Placeholder */}
      {selectedMonitor ? (
        <PacketLossMonitorDetails
          selectedMonitor={selectedMonitor}
          monitorStatuses={monitorStatuses}
          onTraceRoute={crossTabNavigation ? handleTraceRoute : undefined}
        />
      ) : (
        monitorList.length > 0 && <EmptyStatePlaceholder />
      )}

      {/* Add/Edit Monitor Modal */}
      <PacketLossMonitorForm
        showForm={showForm}
        onClose={handleCancelForm}
        onSubmit={handleSubmit}
        editingMonitor={editingMonitor}
        formData={formData}
        onFormDataChange={setFormData}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />
    </div>
  );
};
