/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";

interface MetricCardProps {
  icon: React.ReactNode;
  title: string;
  value: string;
  unit: string;
  average?: string;
  className?: string;
  status?: "normal" | "warning" | "error" | "success";
}

export const MetricCard: React.FC<MetricCardProps> = ({
  icon,
  title,
  value,
  unit,
  average,
  className = "",
  status = "normal",
}) => {
  const statusColors = {
    normal: "",
    success: "ring-1 ring-emerald-500/20 bg-emerald-500/5",
    warning: "ring-1 ring-amber-500/20 bg-amber-500/5",
    error: "ring-1 ring-red-500/20 bg-red-500/5",
  };

  const valueColors = {
    normal: "text-gray-900 dark:text-white",
    success: "text-emerald-600 dark:text-emerald-400",
    warning: "text-amber-600 dark:text-amber-400",
    error: "text-red-600 dark:text-red-400",
  };

  return (
    <div
      className={`bg-gray-50/95 dark:bg-gray-850/95 p-3 sm:p-4 rounded-xl border border-gray-200 dark:border-gray-800 shadow-lg ${statusColors[status]} ${className}`}
    >
      <div className="flex items-center gap-2 sm:gap-3 mb-2">
        <div className="text-gray-600 dark:text-gray-400 flex-shrink-0">{icon}</div>
        <h3 className="text-gray-700 dark:text-gray-300 font-medium text-sm sm:text-base truncate">
          {title}
        </h3>
      </div>
      <div className="flex items-baseline gap-1 sm:gap-2">
        <span className={`text-xl sm:text-2xl font-bold ${valueColors[status]}`}>
          {value}
        </span>
        <span className="text-gray-600 dark:text-gray-400 text-sm sm:text-base">{unit}</span>
      </div>
      {average && (
        <div className="mt-1 text-xs sm:text-sm text-gray-600 dark:text-gray-400 truncate">
          <span className="hidden sm:inline">Average: </span>
          <span className="sm:hidden">Avg: </span>
          {average} {unit}
        </div>
      )}
    </div>
  );
};
