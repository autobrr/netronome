/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment } from "react";
import { Dialog, Transition } from "@headlessui/react";
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
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md transform overflow-hidden rounded-xl bg-gray-50/95 dark:bg-gray-850/95 border border-gray-200 dark:border-gray-900 p-6 shadow-xl transition-all">
                <Dialog.Title
                  as="h2"
                  className="text-xl font-semibold text-gray-900 dark:text-white mb-4"
                >
                  Delete Monitor
                </Dialog.Title>

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
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
