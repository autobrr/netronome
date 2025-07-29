/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import {
  XMarkIcon,
  ArrowDownIcon,
  ArrowUpIcon,
  ChartBarIcon,
  CpuChipIcon,
} from "@heroicons/react/24/outline";
import { MonitorAgent } from "@/api/monitor";
import { getAgentIcon } from "@/utils/agentIcons";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import { formatBytes } from "@/utils/formatBytes";
import { TailscaleLogo } from "@/components/icons/TailscaleLogo";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface MonitorUsageModalProps {
  isOpen: boolean;
  onClose: () => void;
  agent: MonitorAgent;
}

export const MonitorUsageModal: React.FC<MonitorUsageModalProps> = ({
  isOpen,
  onClose,
  agent,
}) => {
  const { status, nativeData, hardwareStats } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeSystemInfo: true,
    includeHardwareStats: true,
  });

  const { icon: AgentIcon } = getAgentIcon(agent.name);
  const usage = nativeData ? parseMonitorUsagePeriods(nativeData) : null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="w-full max-w-none sm:max-w-[90vw] md:max-w-6xl p-6 md:p-8 scrollbar-hide" showCloseButton={false}>
        {/* Header */}
        <DialogHeader className="mb-8">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              {status?.connected ? (
                <span className="relative inline-flex h-3 w-3 flex-shrink-0">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
                </span>
              ) : (
                <div className="h-3 w-3 rounded-full flex-shrink-0 bg-red-500" />
              )}
              <AgentIcon className="h-6 w-6 text-gray-600 dark:text-gray-400" />
              <div>
                <div className="flex items-center gap-2">
                  <DialogTitle className="text-lg font-semibold leading-6 text-gray-900 dark:text-white">
                    {agent.name}
                  </DialogTitle>
                  {agent.isTailscale && (
                    <TailscaleLogo
                      className="h-4 w-4 flex-shrink-0"
                      title="Connected through Tailscale"
                    />
                  )}
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Monitor Dashboard
                </p>
              </div>
            </div>
            <button
              type="button"
              className="rounded-md p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-300 hover:bg-gray-200/50 dark:hover:bg-gray-800/50 transition-colors"
              onClick={onClose}
            >
              <span className="sr-only">Close</span>
              <XMarkIcon className="h-6 w-6" aria-hidden="true" />
            </button>
          </div>
        </DialogHeader>

        {/* Content */}
        <div className="max-h-[calc(85vh-12rem)] overflow-y-auto scrollbar-hide">
          {status?.connected ? (
            <div className="space-y-8">
              {/* Key Metrics Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                {/* Current Speed */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-5 shadow-lg border border-gray-200 dark:border-gray-800">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Current Speed
                    </h3>
                    <ChartBarIcon className="h-5 w-5 text-gray-400" />
                  </div>
                  {status?.liveData ? (
                    <>
                      <div className="flex items-center space-x-2 mb-2">
                        <ArrowDownIcon className="h-4 w-4 text-blue-500" />
                        <span className="text-lg font-bold text-gray-900 dark:text-white">
                          {status.liveData.rx.ratestring}
                        </span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <ArrowUpIcon className="h-4 w-4 text-green-500" />
                        <span className="text-lg font-bold text-gray-900 dark:text-white">
                          {status.liveData.tx.ratestring}
                        </span>
                      </div>
                    </>
                  ) : (
                    <p className="text-gray-500 dark:text-gray-400">No data</p>
                  )}
                </div>

                {/* Today's Usage */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-5 shadow-lg border border-gray-200 dark:border-gray-800">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Today's Usage
                    </h3>
                    <ChartBarIcon className="h-5 w-5 text-gray-400" />
                  </div>
                  {usage?.["Today"] ? (
                    <>
                      <p className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                        {formatBytes(usage["Today"].total)}
                      </p>
                      <div className="flex items-center space-x-4 text-sm">
                        <div className="flex items-center space-x-1">
                          <ArrowDownIcon className="h-3 w-3 text-blue-500" />
                          <span className="text-gray-600 dark:text-gray-400">
                            {formatBytes(usage["Today"].download)}
                          </span>
                        </div>
                        <div className="flex items-center space-x-1">
                          <ArrowUpIcon className="h-3 w-3 text-green-500" />
                          <span className="text-gray-600 dark:text-gray-400">
                            {formatBytes(usage["Today"].upload)}
                          </span>
                        </div>
                      </div>
                    </>
                  ) : (
                    <p className="text-gray-500 dark:text-gray-400">No data</p>
                  )}
                </div>

                {/* System Health */}
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-5 shadow-lg border border-gray-200 dark:border-gray-800">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      System Health
                    </h3>
                    <CpuChipIcon className="h-5 w-5 text-gray-400" />
                  </div>
                  {hardwareStats ? (
                    <div className="space-y-2">
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          CPU
                        </span>
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          {hardwareStats.cpu.usage_percent.toFixed(1)}%
                        </span>
                      </div>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div
                          className="h-2 rounded-full transition-all duration-300"
                          style={{
                            width: `${hardwareStats.cpu.usage_percent}%`,
                            backgroundColor:
                              hardwareStats.cpu.usage_percent < 70
                                ? "#10B981"
                                : hardwareStats.cpu.usage_percent < 85
                                  ? "#F59E0B"
                                  : "#EF4444",
                          }}
                        />
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Memory
                        </span>
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          {hardwareStats.memory.used_percent.toFixed(1)}%
                        </span>
                      </div>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div
                          className="h-2 rounded-full transition-all duration-300"
                          style={{
                            width: `${hardwareStats.memory.used_percent}%`,
                            backgroundColor:
                              hardwareStats.memory.used_percent < 70
                                ? "#10B981"
                                : hardwareStats.memory.used_percent < 85
                                  ? "#F59E0B"
                                  : "#EF4444",
                          }}
                        />
                      </div>
                      {/* Temperature Alert */}
                      {hardwareStats.temperature &&
                        hardwareStats.temperature.length > 0 &&
                        (() => {
                          const hotSensors = hardwareStats.temperature.filter(
                            (t) => t.temperature > 80,
                          );
                          const warmSensors = hardwareStats.temperature.filter(
                            (t) => t.temperature > 60 && t.temperature <= 80,
                          );

                          if (hotSensors.length > 0) {
                            return (
                              <div className="flex justify-between items-center">
                                <span className="text-sm text-gray-600 dark:text-gray-400">
                                  Temperature
                                </span>
                                <span className="text-sm font-medium text-red-600 dark:text-red-400">
                                  {hotSensors.length} Hot
                                </span>
                              </div>
                            );
                          } else if (warmSensors.length > 0) {
                            return (
                              <div className="flex justify-between items-center">
                                <span className="text-sm text-gray-600 dark:text-gray-400">
                                  Temperature
                                </span>
                                <span className="text-sm font-medium text-amber-600 dark:text-amber-400">
                                  {warmSensors.length} Warm
                                </span>
                              </div>
                            );
                          } else {
                            return (
                              <div className="flex justify-between items-center">
                                <span className="text-sm text-gray-600 dark:text-gray-400">
                                  Temperature
                                </span>
                                <span className="text-sm font-medium text-green-600 dark:text-green-400">
                                  Normal
                                </span>
                              </div>
                            );
                          }
                        })()}
                    </div>
                  ) : (
                    <p className="text-gray-500 dark:text-gray-400">No data</p>
                  )}
                </div>
              </div>

              {/* Usage Summary */}
              {usage && (
                <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 md:p-8 shadow-lg border border-gray-200 dark:border-gray-800">
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-6">
                    Usage Summary
                  </h3>
                  <div className="space-y-4">
                    {Object.entries(usage).map(([period, data]) => (
                      <div
                        key={period}
                        className="flex items-center justify-between p-4 bg-gray-100 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-900"
                      >
                        <div className="flex-1">
                          <p className="font-medium text-gray-900 dark:text-white">
                            {period}
                          </p>
                          <div className="flex items-center space-x-3 mt-1">
                            <div className="flex items-center space-x-1">
                              <ArrowDownIcon className="h-3 w-3 text-blue-500" />
                              <span className="text-xs text-gray-600 dark:text-gray-400">
                                {formatBytes((data as any).download)}
                              </span>
                            </div>
                            <div className="flex items-center space-x-1">
                              <ArrowUpIcon className="h-3 w-3 text-green-500" />
                              <span className="text-xs text-gray-600 dark:text-gray-400">
                                {formatBytes((data as any).upload)}
                              </span>
                            </div>
                          </div>
                        </div>
                        <div className="text-right">
                          <p className="font-bold text-gray-900 dark:text-white">
                            {formatBytes((data as any).total)}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-12">
              <div className="h-16 w-16 rounded-full bg-red-100 dark:bg-red-900/20 mx-auto mb-4 flex items-center justify-center">
                <AgentIcon className="h-8 w-8 text-red-600 dark:text-red-400" />
              </div>
              <p className="text-lg text-gray-900 dark:text-white font-medium mb-2">
                Agent Disconnected
              </p>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Unable to connect to {agent.url}
              </p>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};
