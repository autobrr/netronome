/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { PresentationChartLineIcon, CheckIcon } from "@heroicons/react/24/outline";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { settingsApi } from "@/api/settings";
import { showToast } from "@/components/common/Toast";

const ROW_OPTIONS = [10, 20, 50, 100];

export const DashboardSettings: React.FC = () => {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["dashboard-settings"],
    queryFn: settingsApi.getDashboardSettings,
  });

  const [pendingRows, setPendingRows] = useState<number | null>(null);
  const persistedRows = data?.recentSpeedtestsRows ?? 20;
  const selectedRows = pendingRows ?? persistedRows;
  const hasChanges = pendingRows !== null && pendingRows !== persistedRows;

  const mutation = useMutation({
    mutationFn: settingsApi.updateDashboardSettings,
    onSuccess: (updated) => {
      queryClient.setQueryData(["dashboard-settings"], updated);
      queryClient.invalidateQueries({ queryKey: ["history"] });
      setPendingRows(null);
      showToast("Dashboard settings saved", "success", {
        description: "Recent Speedtests row count updated",
      });
    },
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : undefined;
      showToast("Failed to save dashboard settings", "error", {
        description: message,
      });
    },
  });

  const updateRecentRows = (value: string) => {
    const parsedValue = Number(value);
    setPendingRows(parsedValue);
  };

  const cancelChanges = () => {
    setPendingRows(null);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[220px]">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full mx-auto mb-4 animate-spin" />
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Loading dashboard settings...
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
            <PresentationChartLineIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
            Dashboard Settings
          </h3>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Configure recent speedtest history display behavior
          </p>
        </div>

        {hasChanges && (
          <div className="flex items-center gap-2">
            <Button
              onClick={() =>
                mutation.mutate({ recentSpeedtestsRows: selectedRows })
              }
              disabled={mutation.isPending}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              <CheckIcon className="w-4 h-4" />
              Save Changes
            </Button>
            <Button onClick={cancelChanges} variant="secondary" disabled={mutation.isPending}>
              Cancel
            </Button>
          </div>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <PresentationChartLineIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            Recent Speedtests
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Rows shown on dashboard
            </label>
            <Select
              value={String(selectedRows)}
              onValueChange={updateRecentRows}
            >
              <SelectTrigger className="w-full sm:w-[240px]">
                <SelectValue placeholder="Select row count" />
              </SelectTrigger>
              <SelectContent>
                {ROW_OPTIONS.map((option) => (
                  <SelectItem key={option} value={String(option)}>
                    {option} rows
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <p className="text-sm text-gray-600 dark:text-gray-400">
            Controls how many recent speedtests are loaded and shown in Dashboard {'>'} Recent Speedtests.
          </p>
        </CardContent>
      </Card>
    </div>
  );
};
