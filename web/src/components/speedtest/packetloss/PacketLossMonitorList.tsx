/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  TrashIcon,
  PlayIcon,
  StopIcon,
  PencilIcon,
  SignalIcon,
  ChartBarIcon,
  ExclamationTriangleIcon,
} from "@heroicons/react/24/outline";
import { PacketLossMonitor } from "@/types/types";
import { formatInterval } from "./utils/packetLossUtils";
import { DeleteMonitorModal } from "./DeleteMonitorModal";
import { MonitorStatus } from "./types/monitorStatus";

interface MonitorStatusDisplayProps {
  monitor: PacketLossMonitor;
  status: MonitorStatus | undefined;
}

const MonitorStatusDisplay: React.FC<MonitorStatusDisplayProps> = ({
  monitor,
  status,
}) => {
  // If disabled, show the schedule with gray styling
  if (!monitor.enabled) {
    return (
      <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <div className="flex-1 min-w-0">
            {monitor.interval.startsWith("exact:") ? (
              <div className="flex flex-col gap-1">
                <div className="flex flex-wrap gap-1">
                  {monitor.interval
                    .substring(6)
                    .split(",")
                    .slice(0, 3) // Show max 3 times on first line
                    .map((time, index) => (
                      <span
                        key={index}
                        className="inline-flex items-center px-1.5 py-0.5 bg-gray-500/10 text-gray-600 dark:text-gray-400 rounded text-xs border border-gray-500/20"
                      >
                        {new Date(
                          `2000-01-01T${time.trim()}:00`
                        ).toLocaleTimeString([], {
                          hour: "numeric",
                          minute: "2-digit",
                          hour12: true,
                        })}
                      </span>
                    ))}
                  {monitor.interval.substring(6).split(",").length > 3 && (
                    <span className="text-xs text-gray-600 dark:text-gray-400 px-1">
                      +{monitor.interval.substring(6).split(",").length - 3}{" "}
                      more
                    </span>
                  )}
                </div>
              </div>
            ) : (
              <div className="flex flex-wrap gap-1">
                <span className="inline-flex items-center px-1.5 py-0.5 bg-gray-500/10 text-gray-600 dark:text-gray-400 rounded text-xs border border-gray-500/20">
                  Every {formatInterval(monitor.interval)}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // State 1: Actively testing (IsRunning=true + Progress>0)
  if (
    status &&
    status.isRunning &&
    status.progress !== undefined &&
    status.progress > 0
  ) {
    const progressText = status.usedMtr
      ? "MTR"
      : `${Math.round(status.progress || 0)}%`;
    const packetsText =
      !status.usedMtr && status.packetsSent
        ? ` (${status.packetsRecv || 0}/${status.packetsSent})`
        : "";
    return (
      <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-2">
            <span className="relative inline-flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
            </span>
            <span className="text-blue-600 dark:text-blue-400 text-sm font-medium">
              {status.usedMtr
                ? "Running MTR..."
                : `Testing... ${progressText}${packetsText}`}
            </span>
          </div>
        </div>
      </div>
    );
  }

  // State 2: Scheduled monitoring (IsRunning=false, IsComplete=false)
  else if (status && !status.isRunning && !status.isComplete) {
    // Show last result if available, otherwise show monitoring status
    if (status.packetLoss !== undefined) {
      const isHealthy = status.packetLoss <= monitor.threshold;
      return (
        <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div
                className={`w-2 h-2 rounded-full ${
                  isHealthy ? "bg-emerald-500" : "bg-red-500"
                }`}
              />
              <span
                className={`text-sm font-medium ${
                  isHealthy
                    ? "text-emerald-600 dark:text-emerald-400"
                    : "text-red-600 dark:text-red-400"
                }`}
              >
                {status.packetLoss.toFixed(1)}% packet loss
              </span>
            </div>
            {status.avgRtt && (
              <span className="text-xs text-gray-500 dark:text-gray-500">
                {status.avgRtt.toFixed(1)} avg
              </span>
            )}
          </div>
        </div>
      );
    }
  }

  // Default monitoring state
  return (
    <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700">
      <div className="flex items-center gap-2">
        <div className="flex-1 min-w-0">
          {monitor.interval.startsWith("exact:") ? (
            <div className="flex flex-col gap-1">
              <div className="flex flex-wrap gap-1">
                {monitor.interval
                  .substring(6)
                  .split(",")
                  .slice(0, 3) // Show max 3 times on first line
                  .map((time, index) => (
                    <span
                      key={index}
                      className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs border ${
                        monitor.enabled
                          ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                          : "bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20"
                      }`}
                    >
                      {new Date(
                        `2000-01-01T${time.trim()}:00`
                      ).toLocaleTimeString([], {
                        hour: "numeric",
                        minute: "2-digit",
                        hour12: true,
                      })}
                    </span>
                  ))}
                {monitor.interval.substring(6).split(",").length > 3 && (
                  <span
                    className={`text-xs px-1 ${
                      monitor.enabled
                        ? "text-emerald-600 dark:text-emerald-400"
                        : "text-gray-600 dark:text-gray-400"
                    }`}
                  >
                    +{monitor.interval.substring(6).split(",").length - 3} more
                  </span>
                )}
              </div>
            </div>
          ) : (
            <div className="flex flex-wrap gap-1">
              <span
                className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs border ${
                  monitor.enabled
                    ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                    : "bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20"
                }`}
              >
                Every {formatInterval(monitor.interval)}
              </span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

interface PacketLossMonitorListProps {
  monitors: PacketLossMonitor[];
  selectedMonitor: PacketLossMonitor | null;
  monitorStatuses: Map<number, MonitorStatus>;
  onMonitorSelect: (monitor: PacketLossMonitor | null) => void;
  onEdit: (monitor: PacketLossMonitor) => void;
  onDelete: (monitorId: number) => void;
  onToggle: (monitorId: number, enabled: boolean) => void;
  isLoading: boolean;
  togglingMonitorIds?: Set<number>;
}

export const PacketLossMonitorList: React.FC<PacketLossMonitorListProps> = ({
  monitors,
  selectedMonitor,
  monitorStatuses,
  onMonitorSelect,
  onEdit,
  onDelete,
  onToggle,
  isLoading,
  togglingMonitorIds = new Set(),
}) => {
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [monitorToDelete, setMonitorToDelete] =
    useState<PacketLossMonitor | null>(null);
  if (isLoading) {
    return (
      <div className="text-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400 mx-auto"></div>
        <p className="text-gray-600 dark:text-gray-400 mt-4">
          Loading monitors...
        </p>
      </div>
    );
  }

  if (monitors.length === 0) {
    return (
      <div className="text-center py-8">
        <SignalIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <p className="text-gray-600 dark:text-gray-400">
          No monitors configured yet
        </p>
        <p className="text-gray-500 dark:text-gray-500 text-sm mt-2">
          Add a monitor to start tracking packet loss
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-3">
        {monitors.map((monitor) => (
          <div
            key={monitor.id}
            className={`p-4 rounded-lg border transition-all cursor-pointer ${
              selectedMonitor?.id === monitor.id
                ? "bg-blue-500/10 border-blue-400/50 shadow-lg"
                : "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 hover:shadow-md"
            }`}
            onClick={() =>
              onMonitorSelect(
                selectedMonitor?.id === monitor.id ? null : monitor
              )
            }
          >
            <div className="flex items-start justify-between">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3 mb-2">
                  <h3 className="text-gray-900 dark:text-white font-semibold truncate">
                    {monitor.name || monitor.host}
                  </h3>
                  {/* Status Badge */}
                  <div className="flex items-center gap-2">
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
                  </div>
                </div>

                {monitor.name && (
                  <p className="text-gray-600 dark:text-gray-400 text-sm mb-2 truncate">
                    {monitor.host}
                  </p>
                )}

                <div className="flex flex-wrap mt-4 gap-x-3 gap-y-1 text-xs text-gray-600 dark:text-gray-400">
                  <span className="flex items-center gap-1">
                    <ChartBarIcon className="w-3.5 h-3.5 text-emerald-500" />
                    {monitor.packetCount} packets
                  </span>
                  <span className="flex items-center gap-1">
                    <ExclamationTriangleIcon className="w-3.5 h-3.5 text-amber-500" />
                    {monitor.threshold}% threshold
                  </span>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex items-center gap-1 ml-4">
                <button
                  className={`p-1.5 rounded-md transition-colors duration-200 ${
                    togglingMonitorIds.has(monitor.id)
                      ? "opacity-50 cursor-not-allowed"
                      : monitor.enabled
                      ? "hover:bg-red-100 dark:hover:bg-red-900/30 text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400"
                      : "hover:bg-emerald-100 dark:hover:bg-emerald-900/30 text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 dark:hover:text-emerald-400"
                  }`}
                  onClick={(e) => {
                    e.stopPropagation();
                    onToggle(monitor.id, !monitor.enabled);
                  }}
                  disabled={togglingMonitorIds.has(monitor.id)}
                  title={monitor.enabled ? "Stop Monitor" : "Start Monitor"}
                >
                  {togglingMonitorIds.has(monitor.id) ? (
                    <div className="w-3.5 h-3.5 animate-spin rounded-full border-2 border-gray-300 border-t-gray-600 dark:border-gray-600 dark:border-t-gray-300" />
                  ) : monitor.enabled ? (
                    <StopIcon className="w-3.5 h-3.5" />
                  ) : (
                    <PlayIcon className="w-3.5 h-3.5" />
                  )}
                </button>
                <button
                  className="p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-900/30 text-gray-500 hover:text-gray-600 dark:text-gray-400 dark:hover:text-gray-400 transition-colors duration-200"
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(monitor);
                  }}
                  title="Edit Monitor"
                >
                  <PencilIcon className="w-3.5 h-3.5" />
                </button>
                <button
                  className="p-1.5 rounded-md hover:bg-red-100 dark:hover:bg-red-900/30 text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400 transition-colors duration-200"
                  onClick={(e) => {
                    e.stopPropagation();
                    setMonitorToDelete(monitor);
                    setDeleteModalOpen(true);
                  }}
                  title="Delete Monitor"
                >
                  <TrashIcon className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
            <MonitorStatusDisplay
              monitor={monitor}
              status={monitorStatuses.get(monitor.id)}
            />
          </div>
        ))}
      </div>

      {/* Delete Confirmation Modal */}
      <DeleteMonitorModal
        isOpen={deleteModalOpen}
        onClose={() => {
          setDeleteModalOpen(false);
          setMonitorToDelete(null);
        }}
        onConfirm={() => {
          if (monitorToDelete) {
            onDelete(monitorToDelete.id);
          }
          setMonitorToDelete(null);
        }}
        monitor={monitorToDelete}
      />
    </>
  );
};
