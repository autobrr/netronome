/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PacketLossMonitor } from "@/types/types";

interface DeleteMonitorModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  monitor: PacketLossMonitor | null;
}

export function DeleteMonitorModal({
  isOpen,
  onClose,
  onConfirm,
  monitor,
}: DeleteMonitorModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="w-full max-w-md bg-gray-50/95 dark:bg-gray-850/95 border-gray-200 dark:border-gray-900">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold text-gray-900 dark:text-white">
            Delete Monitor
          </DialogTitle>
        </DialogHeader>

                <p className="text-gray-600 dark:text-gray-400 text-sm mb-4">
                  Are you sure you want to delete the monitor{" "}
                  <span className="font-medium text-gray-900 dark:text-white">
                    {monitor?.name || monitor?.host}
                  </span>
                  ? This action cannot be undone.{" "}
                  <b>All history will be lost!</b>
                </p>

                <div className="mt-6 flex justify-end gap-3">
                  <button
                    type="button"
                    className="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                    onClick={onClose}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    className="px-4 py-2 rounded-lg shadow-md transition-colors border text-sm bg-red-500 hover:bg-red-600 text-white border-red-600 hover:border-red-700"
                    onClick={() => {
                      onConfirm();
                      onClose();
                    }}
                  >
                    Delete Monitor
                  </button>
                </div>
      </DialogContent>
    </Dialog>
  );
}
