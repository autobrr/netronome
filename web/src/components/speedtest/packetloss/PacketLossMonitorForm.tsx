/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { Fragment } from "react";
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  Transition,
  TransitionChild,
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
} from "@headlessui/react";
import { XMarkIcon, ChevronUpDownIcon } from "@heroicons/react/24/solid";
import { Button } from "@/components/ui/Button";
import { PacketLossMonitor } from "@/types/types";
import {
  MonitorFormData,
  intervalOptions,
} from "./constants/packetLossConstants";
import { formatInterval } from "./utils/packetLossUtils";

interface PacketLossMonitorFormProps {
  showForm: boolean;
  onClose: () => void;
  onSubmit: (data: MonitorFormData) => void;
  editingMonitor?: PacketLossMonitor | null;
  formData: MonitorFormData;
  onFormDataChange: (data: MonitorFormData) => void;
  isLoading?: boolean;
}

export const PacketLossMonitorForm: React.FC<PacketLossMonitorFormProps> = ({
  showForm,
  onClose,
  onSubmit,
  editingMonitor,
  formData,
  onFormDataChange,
  isLoading = false,
}) => {
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  return (
    <Transition appear show={showForm} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <TransitionChild
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black bg-opacity-25" />
        </TransitionChild>

        <div className="fixed inset-0 overflow-y-auto backdrop-blur-sm">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <TransitionChild
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <DialogPanel className="w-full max-w-md transform overflow-visible rounded-2xl backdrop-blur-md bg-white dark:bg-gray-850/95 border dark:border-gray-900 p-4 md:p-6 text-left align-middle shadow-xl transition-all">
                <div className="flex items-center justify-between mb-4">
                  <DialogTitle
                    as="h3"
                    className="text-lg font-medium leading-6 text-gray-900 dark:text-white"
                  >
                    {editingMonitor ? "Edit Monitor" : "New Monitor"}
                  </DialogTitle>
                  <button
                    onClick={onClose}
                    className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                  >
                    <XMarkIcon className="h-5 w-5" />
                  </button>
                </div>

                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Host
                      </label>
                      <input
                        type="text"
                        value={formData.host}
                        onChange={(e) =>
                          onFormDataChange({
                            ...formData,
                            host: e.target.value,
                          })
                        }
                        placeholder="e.g., google.com or 8.8.8.8"
                        className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                        required
                      />
                      <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                        Some hosts may block ICMP ping requests
                      </p>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Name (Optional)
                      </label>
                      <input
                        type="text"
                        value={formData.name}
                        onChange={(e) =>
                          onFormDataChange({
                            ...formData,
                            name: e.target.value,
                          })
                        }
                        placeholder="e.g., Google DNS"
                        className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                      />
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          Check Interval
                        </label>
                        <Listbox
                          value={formData.interval}
                          onChange={(value) =>
                            onFormDataChange({ ...formData, interval: value })
                          }
                        >
                          <div className="relative">
                            <ListboxButton className="relative w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-lg text-left text-gray-700 dark:text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                              <span className="block truncate">
                                {intervalOptions.find(
                                  (opt) => opt.value === formData.interval,
                                )?.label ||
                                  `Every ${formatInterval(formData.interval)}`}
                              </span>
                              <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                                <ChevronUpDownIcon
                                  className="h-5 w-5 text-gray-600 dark:text-gray-400"
                                  aria-hidden="true"
                                />
                              </span>
                            </ListboxButton>
                            <Transition
                              enter="transition duration-100 ease-out"
                              enterFrom="transform scale-95 opacity-0"
                              enterTo="transform scale-100 opacity-100"
                              leave="transition duration-75 ease-out"
                              leaveFrom="transform scale-100 opacity-100"
                              leaveTo="transform scale-95 opacity-0"
                            >
                              <ListboxOptions className="absolute z-10 mt-1 max-h-80 w-full overflow-auto rounded-lg bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-900 py-1 shadow-lg focus:outline-none">
                                {intervalOptions.map((option) => (
                                  <ListboxOption
                                    key={option.value}
                                    value={option.value}
                                    className={({ focus }) =>
                                      `relative cursor-pointer select-none py-2 px-4 ${
                                        focus
                                          ? "bg-blue-500/10 text-blue-600 dark:text-blue-200"
                                          : "text-gray-700 dark:text-gray-300"
                                      }`
                                    }
                                  >
                                    {option.label}
                                  </ListboxOption>
                                ))}
                              </ListboxOptions>
                            </Transition>
                          </div>
                        </Listbox>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          Packets per Test
                        </label>
                        <input
                          type="number"
                          value={formData.packetCount}
                          onChange={(e) =>
                            onFormDataChange({
                              ...formData,
                              packetCount: parseInt(e.target.value) || 10,
                            })
                          }
                          min="1"
                          max="100"
                          className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                        />
                      </div>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Alert Threshold (% packet loss)
                      </label>
                      <input
                        type="number"
                        value={formData.threshold}
                        onChange={(e) =>
                          onFormDataChange({
                            ...formData,
                            threshold: parseFloat(e.target.value) || 5.0,
                          })
                        }
                        min="0"
                        max="100"
                        step="0.1"
                        className="w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                      />
                    </div>
                    <div className="flex items-center">
                      <input
                        type="checkbox"
                        id="enabled"
                        checked={formData.enabled}
                        onChange={(e) =>
                          onFormDataChange({
                            ...formData,
                            enabled: e.target.checked,
                          })
                        }
                        className="mr-2"
                      />
                      <label
                        htmlFor="enabled"
                        className="text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        Start monitoring immediately
                      </label>
                    </div>
                  </div>

                  <div className="flex gap-3 pt-4">
                    <Button
                      type="submit"
                      disabled={!formData.host || isLoading}
                      isLoading={isLoading}
                      className="flex-1 bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                    >
                      {editingMonitor ? "Update Monitor" : "Create Monitor"}
                    </Button>
                    <Button
                      type="button"
                      onClick={onClose}
                      className="bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300"
                    >
                      Cancel
                    </Button>
                  </div>
                </form>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
};
