/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { CircleStackIcon, TrashIcon } from "@heroicons/react/24/outline";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DeleteConfirmationDialog } from "@/components/common/DeleteConfirmationDialog";
import { settingsApi } from "@/api/settings";
import { showToast } from "@/components/common/Toast";

const RETENTION_OPTIONS = [
  { value: 7, label: "older than 7 days" },
  { value: 30, label: "older than 30 days" },
  { value: 90, label: "older than 90 days" },
  { value: 365, label: "older than 1 year" },
  { value: 0, label: "everything" },
];

export const DataSettings: React.FC = () => {
  const queryClient = useQueryClient();
  const [olderThanDays, setOlderThanDays] = useState(30);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const mutation = useMutation({
    mutationFn: () => settingsApi.purgeHistory(olderThanDays),
    onSuccess: (result) => {
      setConfirmOpen(false);
      queryClient.invalidateQueries({ queryKey: ["history"] });
      queryClient.invalidateQueries({ queryKey: ["history-chart"] });
      queryClient.invalidateQueries({ queryKey: ["packetloss"] });
      queryClient.invalidateQueries({ queryKey: ["dns"] });
      showToast("History purged", "success", {
        description: `Deleted ${result.speedTests} speedtests, ${result.packetLoss} packet loss records and ${result.dns} DNS results`,
      });
    },
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : undefined;
      showToast("Failed to purge history", "error", {
        description: message,
      });
    },
  });

  const selected = RETENTION_OPTIONS.find((o) => o.value === olderThanDays);

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <CircleStackIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
          Data Settings
        </h3>
        <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
          Purge historic speedtest and packet loss results
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrashIcon className="w-5 h-5 text-red-600 dark:text-red-400" />
            Purge History
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Delete records
            </label>
            <Select
              value={String(olderThanDays)}
              onValueChange={(value) => setOlderThanDays(Number(value))}
            >
              <SelectTrigger className="w-full sm:w-[240px]">
                <SelectValue placeholder="Select retention window" />
              </SelectTrigger>
              <SelectContent>
                {RETENTION_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={String(option.value)}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <p className="text-sm text-gray-600 dark:text-gray-400">
            Permanently deletes speedtest and packet loss history. This action cannot be undone.
          </p>

          <Button
            onClick={() => setConfirmOpen(true)}
            disabled={mutation.isPending}
            variant="destructive"
          >
            <TrashIcon className="w-4 h-4" />
            Purge
          </Button>
        </CardContent>
      </Card>

      <DeleteConfirmationDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => mutation.mutate()}
        title="Purge History"
        itemName=""
        description={`Permanently delete ${selected?.value === 0 ? "all history" : `history ${selected?.label}`}? This action cannot be undone.`}
        isDeleting={mutation.isPending}
      />
    </div>
  );
};
