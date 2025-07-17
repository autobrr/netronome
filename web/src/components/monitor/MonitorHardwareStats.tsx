/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  CpuChipIcon,
  CircleStackIcon,
  ServerIcon,
  FireIcon,
} from "@heroicons/react/24/outline";
import { HardwareStats } from "@/api/monitor";
import { formatBytes } from "@/utils/formatBytes";

interface MonitorHardwareStatsProps {
  hardwareStats: HardwareStats;
}

export const MonitorHardwareStats: React.FC<MonitorHardwareStatsProps> = ({
  hardwareStats,
}) => {
  const getProgressColor = (percent: number) => {
    if (percent < 50) return "#34d399"; // emerald-400
    if (percent < 75) return "#d97706"; // amber-600
    return "#EF4444"; // red
  };

  const formatFrequency = (mhz: number): string => {
    if (mhz >= 1000) {
      return `${(mhz / 1000).toFixed(2)} GHz`;
    }
    return `${mhz.toFixed(0)} MHz`;
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="space-y-6"
    >
      {/* CPU and Memory Row */}
      <div className="grid gap-6 sm:grid-cols-2">
        {/* CPU Stats */}
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center space-x-2">
              <CpuChipIcon className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                CPU
              </h3>
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">
              {hardwareStats.cpu.cores} cores, {hardwareStats.cpu.threads}{" "}
              threads
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600 dark:text-gray-400">Usage</span>
              <span className="text-gray-900 dark:text-white font-medium">
                {hardwareStats.cpu.usage_percent.toFixed(1)}%
              </span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className="h-2 rounded-full transition-all duration-300"
                style={{
                  width: `${hardwareStats.cpu.usage_percent}%`,
                  backgroundColor: getProgressColor(
                    hardwareStats.cpu.usage_percent
                  ),
                }}
              />
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400 truncate">
              {hardwareStats.cpu.model}
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {formatFrequency(hardwareStats.cpu.frequency)}
            </p>
            {hardwareStats.cpu.load_avg && (
              <p className="text-xs text-gray-500 dark:text-gray-500">
                Load:{" "}
                {hardwareStats.cpu.load_avg.map((l) => l.toFixed(2)).join(", ")}
              </p>
            )}
          </div>
        </div>

        {/* Memory Stats */}
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center space-x-2">
              <CircleStackIcon className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                Memory
              </h3>
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">
              {formatBytes(hardwareStats.memory.total)}
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600 dark:text-gray-400">Usage</span>
              <span className="text-gray-900 dark:text-white font-medium">
                {hardwareStats.memory.used_percent.toFixed(1)}%
              </span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className="h-2 rounded-full transition-all duration-300"
                style={{
                  width: `${hardwareStats.memory.used_percent}%`,
                  backgroundColor: getProgressColor(
                    hardwareStats.memory.used_percent
                  ),
                }}
              />
            </div>
            <div className="mt-2">
              <div className="flex justify-between text-xs">
                <span className="text-gray-500 dark:text-gray-500">
                  {formatBytes(hardwareStats.memory.used)} / {formatBytes(hardwareStats.memory.total)}
                </span>
                <span className="text-gray-500 dark:text-gray-500">
                  {formatBytes(hardwareStats.memory.available)} available
                </span>
              </div>
              {hardwareStats.memory.swap_total > 0 && (
                <div className="pt-2 mt-2 border-t border-gray-200 dark:border-gray-700">
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-gray-600 dark:text-gray-400">
                      Swap
                    </span>
                    <span className="text-gray-900 dark:text-white font-medium">
                      {hardwareStats.memory.swap_percent.toFixed(1)}%
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div
                      className="h-2 rounded-full transition-all duration-300"
                      style={{
                        width: `${hardwareStats.memory.swap_percent}%`,
                        backgroundColor: getProgressColor(
                          hardwareStats.memory.swap_percent
                        ),
                      }}
                    />
                  </div>
                  <div className="flex justify-between text-xs mt-1">
                    <span className="text-gray-500 dark:text-gray-500">
                      {formatBytes(hardwareStats.memory.swap_used)} /{" "}
                      {formatBytes(hardwareStats.memory.swap_total)}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Disk Usage */}
      {hardwareStats.disks.length > 0 && (
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center space-x-2 mb-4">
            <ServerIcon className="h-6 w-6 text-purple-600 dark:text-purple-400" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              Disk Usage
            </h3>
          </div>

          <div className="space-y-3">
            {hardwareStats.disks.map((disk, index) => (
              <div
                key={index}
                className="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg"
              >
                <div className="flex justify-between items-center mb-2">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-white truncate">
                      {disk.path}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {disk.device} • {disk.fstype}
                    </p>
                  </div>
                  <div className="text-right ml-4">
                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                      {disk.used_percent.toFixed(1)}%
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {formatBytes(disk.used)} / {formatBytes(disk.total)}
                    </p>
                  </div>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full transition-all duration-300 ${
                      disk.used_percent < 70
                        ? "bg-emerald-500"
                        : disk.used_percent < 85
                        ? "bg-yellow-500"
                        : "bg-red-500"
                    }`}
                    style={{ width: `${disk.used_percent}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Temperature Sensors */}
      {hardwareStats.temperature && hardwareStats.temperature.length > 0 && (
        <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
          <div className="flex items-center space-x-2 mb-4">
            <FireIcon className="h-6 w-6 text-orange-600 dark:text-orange-400" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              Temperature Sensors
            </h3>
          </div>

          <div className="space-y-6">
            {(() => {
              // Categorize temperature sensors (no filtering - show all sensors)
              const filteredTemps = hardwareStats.temperature;

              const getCategory = (temp: (typeof filteredTemps)[0]) => {
                const key = temp.sensor_key.toLowerCase();
                const label = (temp.label || "").toLowerCase();

                // CPU - All processor-related sensors
                if (
                  key.includes("coretemp_core_") ||
                  key.includes("coretemp_package") ||
                  label.includes("core ") ||
                  label.includes("package")
                )
                  return "cpu";
                if (key.includes("pmu tdie") || key.includes("pmu2 tdie"))
                  return "cpu";
                if (key.includes("pmu tdev") || key.includes("pmu2 tdev"))
                  return "cpu";
                // AMD k10temp sensors
                if (key.includes("k10temp_")) return "cpu";
                // AMD zenpower sensors
                if (key.includes("zenpower_")) return "cpu";

                // Storage - All storage devices
                if (key.includes("nvme") || label.includes("nvme"))
                  return "storage";
                if (key.includes("nand") || key.includes("smart_"))
                  return "storage";
                if (label.includes("hdd") || label.includes("ssd"))
                  return "storage";

                // Power - Battery and power management
                if (key.includes("battery") || key.includes("gas gauge"))
                  return "power";

                // System - Everything else (calibration, ACPI, misc)
                return "system";
              };

              // Group sensors by category
              const categories = filteredTemps.reduce((acc, temp) => {
                const category = getCategory(temp);
                if (!acc[category]) acc[category] = [];
                acc[category].push(temp);
                return acc;
              }, {} as Record<string, typeof filteredTemps>);

              // Sort within each category
              Object.keys(categories).forEach((cat) => {
                categories[cat].sort((a, b) =>
                  a.sensor_key.localeCompare(b.sensor_key)
                );
              });

              const categoryOrder = [
                { key: "cpu", title: "CPU" },
                { key: "storage", title: "Storage" },
                { key: "power", title: "Power & Battery" },
                { key: "system", title: "System" },
              ];

              return categoryOrder
                .map(({ key, title }) => {
                  const temps = categories[key];
                  if (!temps || temps.length === 0) return null;

                  return (
                    <div key={key} className="space-y-3">
                      <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700 pb-1">
                        {title} ({temps.length})
                      </h4>
                      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
                        {temps.map((temp, index) => {
                          // For invalid critical temps (> 1000°C), use absolute temperature thresholds
                          const hasValidCritical = temp.critical && temp.critical > 0 && temp.critical < 1000;
                          
                          const percentage = hasValidCritical && temp.critical
                            ? Math.min((temp.temperature / temp.critical) * 100, 100)
                            : (temp.temperature / 100) * 100; // Assume 100°C max if no valid critical temp

                          // Use different thresholds based on sensor type
                          // For HDDs/SSDs, 50°C is warm, 60°C is hot
                          // For CPUs/NVMe, 60°C is warm, 80°C is hot
                          const isStorageSensor = temp.sensor_key.toLowerCase().includes('smart_') || 
                                                  temp.label?.toLowerCase().includes('hdd') ||
                                                  temp.label?.toLowerCase().includes('ssd');
                          
                          const warmThreshold = isStorageSensor ? 50 : 60;
                          const hotThreshold = isStorageSensor ? 60 : 80;
                          
                          const isWarm = temp.temperature > warmThreshold && temp.temperature <= hotThreshold;
                          const isHot = temp.temperature > hotThreshold;

                          const getTemperatureColor = () => {
                            if (isHot) return "#EF4444"; // red
                            if (isWarm) return "#F59E0B"; // amber
                            return "#34d399"; // green
                          };

                          // Use original sensor names, with label if available
                          const getDisplayName = () => {
                            // Use the label if it exists (for SMART drives, etc.)
                            if (temp.label) return temp.label;

                            // Otherwise use the original sensor key
                            return temp.sensor_key;
                          };

                          return (
                            <div
                              key={index}
                              className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3"
                            >
                              <p
                                className="text-xs text-gray-600 dark:text-gray-400 truncate mb-1"
                                title={temp.sensor_key}
                              >
                                {getDisplayName()}
                              </p>
                              <p
                                className={`text-xl font-bold mb-2 ${
                                  isHot
                                    ? "text-red-600 dark:text-red-400"
                                    : isWarm
                                    ? "text-amber-600 dark:text-amber-400"
                                    : "text-emerald-600 dark:text-emerald-400"
                                }`}
                              >
                                {temp.temperature.toFixed(1)}°C
                              </p>
                              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                                <div
                                  className="h-1.5 rounded-full transition-all duration-300"
                                  style={{
                                    width: `${Math.min(percentage, 100)}%`,
                                    backgroundColor: getTemperatureColor(),
                                  }}
                                />
                              </div>
                              {temp.critical &&
                                temp.critical > 0 &&
                                temp.critical < 1000 && (
                                  <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                                    Max: {temp.critical.toFixed(0)}°C
                                  </p>
                                )}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  );
                })
                .filter(Boolean);
            })()}
          </div>
        </div>
      )}

      {/* Update timestamp */}
      <div className="text-center text-xs text-gray-500 dark:text-gray-400">
        {hardwareStats.from_cache ? "Data collected" : "Last updated"}: {new Date(hardwareStats.updated_at).toLocaleTimeString()}
        {hardwareStats.from_cache && " (cached)"}
      </div>
    </motion.div>
  );
};
