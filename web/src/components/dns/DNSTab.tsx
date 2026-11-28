/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion } from "motion/react";
import {
  useMutation,
  useQueries,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  GlobeAltIcon,
  PencilIcon,
  PlayIcon,
  StopIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { DNSMonitor, DNSMonitorInput, DNSUpdate } from "@/types/types";
import {
  createDNSMonitor,
  deleteDNSMonitor,
  getDNSMonitors,
  getDNSMonitorStatus,
  updateDNSMonitor,
} from "@/api/dns";
import { Button } from "@/components/ui/Button";
import { showToast } from "@/components/common/Toast";
import { DeleteMonitorModal } from "@/components/common/DeleteMonitorModal";
import { formatInterval } from "@/components/speedtest/packetloss/utils/packetLossUtils";
import { DNSMonitorForm } from "./DNSMonitorForm";
import { DNSMonitorDetails } from "./DNSMonitorDetails";
import { defaultDNSFormData, stateBadge } from "./constants";

export const DNSTab: React.FC = () => {
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingMonitor, setEditingMonitor] = useState<DNSMonitor | null>(null);
  const [formData, setFormData] = useState<DNSMonitorInput>(defaultDNSFormData);
  const [monitorToDelete, setMonitorToDelete] = useState<DNSMonitor | null>(
    null,
  );

  const { data: monitors = [], isLoading } = useQuery({
    queryKey: ["dns", "monitors"],
    queryFn: getDNSMonitors,
    staleTime: 30000,
  });

  const selectedMonitor =
    monitors.find((monitor) => monitor.id === selectedId) ?? null;

  // Live status: the server has no event stream for DNS, so each enabled
  // monitor polls its own status route.
  const statuses = useQueries({
    queries: monitors
      .filter((monitor) => monitor.enabled)
      .map((monitor) => ({
        queryKey: ["dns", "status", monitor.id],
        queryFn: () => getDNSMonitorStatus(monitor.id),
        refetchInterval: 5000,
      })),
    combine: (results) =>
      new Map<number, DNSUpdate>(
        results.flatMap((result) =>
          result.data ? [[result.data.monitorId, result.data] as const] : [],
        ),
      ),
  });

  const refreshMonitors = () =>
    queryClient.invalidateQueries({ queryKey: ["dns", "monitors"] });

  const createMutation = useMutation({
    mutationFn: createDNSMonitor,
    onSuccess: (monitor) => {
      void refreshMonitors();
      showToast("DNS monitor created", "success", {
        description: `Monitoring ${monitor.host}`,
      });
      handleCancelForm();
    },
    onError: (error: Error) => {
      showToast("Failed to create DNS monitor", "error", {
        description: error.message,
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: updateDNSMonitor,
    onSuccess: (monitor) => {
      void refreshMonitors();
      void queryClient.invalidateQueries({
        queryKey: ["dns", "status", monitor.id],
      });
      if (editingMonitor) {
        showToast("DNS monitor updated", "success", {
          description: monitor.name || monitor.host,
        });
        handleCancelForm();
      }
    },
    onError: (error: Error) => {
      showToast("Failed to update DNS monitor", "error", {
        description: error.message,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteDNSMonitor,
    onSuccess: (_, monitorId) => {
      void refreshMonitors();
      if (selectedId === monitorId) {
        setSelectedId(null);
      }
      showToast("DNS monitor deleted", "success");
    },
    onError: (error: Error) => {
      showToast("Failed to delete DNS monitor", "error", {
        description: error.message,
      });
    },
  });

  const handleSubmit = (data: DNSMonitorInput) => {
    if (editingMonitor) {
      updateMutation.mutate({ ...data, id: editingMonitor.id });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleEdit = (monitor: DNSMonitor) => {
    setEditingMonitor(monitor);
    setFormData(monitor);
    setShowForm(true);
  };

  const handleCancelForm = () => {
    setShowForm(false);
    setEditingMonitor(null);
    setFormData(defaultDNSFormData);
  };

  return (
    <div className="flex flex-col md:flex-row gap-6 md:items-start">
      {/* Left Column - Monitor List */}
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2 }}
        className="flex-1"
      >
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                DNS Monitors
              </h2>
              <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                Response time and failures per resolver
              </p>
            </div>

            <Button onClick={() => setShowForm(true)} variant="default">
              Add
            </Button>
          </div>

          {isLoading ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400 mx-auto"></div>
              <p className="text-gray-600 dark:text-gray-400 mt-4">
                Loading monitors...
              </p>
            </div>
          ) : monitors.length === 0 ? (
            <div className="text-center py-8">
              <GlobeAltIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600 dark:text-gray-400">
                No DNS monitors configured yet
              </p>
              <p className="text-gray-500 dark:text-gray-500 text-sm mt-2">
                Add a monitor to watch a resolver
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {monitors.map((monitor) => {
                const status = statuses.get(monitor.id);
                // a disabled monitor polls no status, so its stored state
                // fills the badge
                const badge = stateBadge(status?.state || monitor.lastState);

                return (
                  <div
                    key={monitor.id}
                    className={`p-4 rounded-lg border transition-all cursor-pointer ${
                      selectedId === monitor.id
                        ? "bg-blue-500/10 border-blue-400/50 shadow-lg"
                        : "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 hover:shadow-md"
                    }`}
                    onClick={() =>
                      setSelectedId(
                        selectedId === monitor.id ? null : monitor.id,
                      )
                    }
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2 flex-wrap">
                          <h3 className="text-gray-900 dark:text-white font-semibold truncate">
                            {monitor.name || monitor.host}
                          </h3>
                          <span
                            className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                              monitor.enabled
                                ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
                                : "bg-gray-500/10 text-gray-600 dark:text-gray-400 border border-gray-500/20"
                            }`}
                          >
                            {monitor.enabled ? (
                              <span className="relative inline-flex h-1.5 w-1.5 mr-1.5">
                                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                                <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-emerald-500"></span>
                              </span>
                            ) : (
                              <div className="w-1.5 h-1.5 rounded-full mr-1.5 bg-gray-400" />
                            )}
                            {monitor.enabled ? "Active" : "Stopped"}
                          </span>
                          {badge && (
                            <span
                              className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium border ${badge.className}`}
                            >
                              {badge.label}
                            </span>
                          )}
                        </div>

                        {monitor.name && (
                          <p className="text-gray-600 dark:text-gray-400 text-sm mb-2 truncate">
                            {monitor.host}
                          </p>
                        )}

                        <div className="flex flex-wrap mt-3 gap-x-3 gap-y-1 text-xs text-gray-600 dark:text-gray-400">
                          <span>
                            {monitor.protocol === "dot"
                              ? "DoT"
                              : monitor.protocol.toUpperCase()}
                          </span>
                          <span>
                            {monitor.recordType} {monitor.query}
                          </span>
                        </div>
                      </div>

                      {/* Action Buttons */}
                      <div className="flex items-center gap-1 ml-4">
                        <button
                          className={`p-1.5 rounded-md transition-colors duration-200 ${
                            monitor.enabled
                              ? "hover:bg-red-100 dark:hover:bg-red-900/30 text-red-500 hover:text-red-600 dark:text-red-400"
                              : "hover:bg-emerald-100 dark:hover:bg-emerald-900/30 text-emerald-600 hover:text-emerald-700 dark:text-emerald-400"
                          }`}
                          onClick={(event) => {
                            event.stopPropagation();
                            updateMutation.mutate({
                              ...monitor,
                              enabled: !monitor.enabled,
                            });
                          }}
                          disabled={updateMutation.isPending}
                          title={
                            monitor.enabled ? "Stop Monitor" : "Start Monitor"
                          }
                        >
                          {monitor.enabled ? (
                            <StopIcon className="w-3.5 h-3.5" />
                          ) : (
                            <PlayIcon className="w-3.5 h-3.5" />
                          )}
                        </button>
                        <button
                          className="p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-900/30 text-gray-500 hover:text-gray-600 dark:text-gray-400 transition-colors duration-200"
                          onClick={(event) => {
                            event.stopPropagation();
                            handleEdit(monitor);
                          }}
                          title="Edit Monitor"
                        >
                          <PencilIcon className="w-3.5 h-3.5" />
                        </button>
                        <button
                          className="p-1.5 rounded-md hover:bg-red-100 dark:hover:bg-red-900/30 text-red-500 hover:text-red-600 dark:text-red-400 transition-colors duration-200"
                          onClick={(event) => {
                            event.stopPropagation();
                            setMonitorToDelete(monitor);
                          }}
                          title="Delete Monitor"
                        >
                          <TrashIcon className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>

                    <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700 flex items-center justify-between gap-2">
                      <span
                        className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs border ${
                          monitor.enabled
                            ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                            : "bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20"
                        }`}
                      >
                        Every {formatInterval(monitor.interval)}
                      </span>
                      {monitor.enabled && badge && status && (
                        <span
                          className={`text-xs font-mono ${
                            status.success
                              ? "text-blue-600 dark:text-blue-400"
                              : "text-red-600 dark:text-red-400"
                          }`}
                        >
                          {status.success
                            ? `${(status.responseTimeMs ?? 0).toFixed(1)} ms`
                            : status.responseCode || "failed"}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </motion.div>

      {/* Right Column - Monitor Details */}
      {selectedMonitor && (
        <DNSMonitorDetails
          key={selectedMonitor.id}
          monitor={selectedMonitor}
          status={statuses.get(selectedMonitor.id)}
        />
      )}

      <DNSMonitorForm
        showForm={showForm}
        onClose={handleCancelForm}
        onSubmit={handleSubmit}
        editingMonitor={editingMonitor}
        formData={formData}
        onFormDataChange={setFormData}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />

      <DeleteMonitorModal
        isOpen={monitorToDelete !== null}
        onClose={() => setMonitorToDelete(null)}
        onConfirm={() => {
          if (monitorToDelete) {
            deleteMutation.mutate(monitorToDelete.id);
          }
        }}
        monitor={monitorToDelete}
      />
    </div>
  );
};
