/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect, Fragment } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/Button";
import { VnstatAgent, CreateAgentRequest } from "@/api/vnstat";

interface VnstatAgentFormProps {
  agent?: VnstatAgent | null;
  onSubmit: (data: CreateAgentRequest, importHistorical?: boolean) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  isOpen: boolean;
}

export const VnstatAgentForm: React.FC<VnstatAgentFormProps> = ({
  agent,
  onSubmit,
  onCancel,
  isSubmitting,
  isOpen,
}) => {
  const [formData, setFormData] = useState<CreateAgentRequest>({
    name: "",
    url: "http://",
    enabled: true,
    retentionDays: 365,
  });
  const [importHistorical, setImportHistorical] = useState(false);

  useEffect(() => {
    if (agent) {
      setFormData({
        name: agent.name,
        url: agent.url.replace(/\/events\?stream=live-data$/, ""),
        enabled: agent.enabled,
        retentionDays: agent.retentionDays,
      });
    } else if (isOpen) {
      // Reset form for new agent
      setFormData({
        name: "",
        url: "http://",
        enabled: true,
        retentionDays: 365,
      });
      setImportHistorical(false);
    }
  }, [agent, isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData, !agent ? importHistorical : undefined);
  };

  const handleUrlChange = (value: string) => {
    // Clean up the URL - remove any trailing slashes and the SSE endpoint if present
    let cleanUrl = value.trim();
    if (cleanUrl.endsWith("/")) {
      cleanUrl = cleanUrl.slice(0, -1);
    }
    setFormData({ ...formData, url: cleanUrl });
  };

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onCancel}>
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
                <div className="mb-4 flex items-center justify-between">
                  <Dialog.Title
                    as="h3"
                    className="text-lg font-medium text-gray-900 dark:text-white"
                  >
                    {agent ? "Edit Agent" : "Add Agent"}
                  </Dialog.Title>
                  <button
                    onClick={onCancel}
                    className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                  >
                    <XMarkIcon className="h-6 w-6" />
                  </button>
                </div>

                <form onSubmit={handleSubmit} className="space-y-4">
                  <div>
                    <label
                      htmlFor="name"
                      className="block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      Agent Name
                    </label>
                    <input
                      type="text"
                      id="name"
                      value={formData.name}
                      data-1p-ignore
                      onChange={(e) =>
                        setFormData({ ...formData, name: e.target.value })
                      }
                      className="mt-1 block w-full px-4 py-2 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm"
                      placeholder="Remote Server"
                      required
                    />
                  </div>

                  <div>
                    <label
                      htmlFor="url"
                      className="block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      Agent URL
                    </label>
                    <input
                      type="url"
                      id="url"
                      value={formData.url}
                      onChange={(e) => handleUrlChange(e.target.value)}
                      className="mt-1 block w-full px-4 py-2 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm"
                      placeholder="http://192.168.1.100:8200"
                      required
                    />
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      Enter the base URL of the vnstat agent
                    </p>
                  </div>

                  <div>
                    <label
                      htmlFor="retentionDays"
                      className="block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      Data Retention (days)
                    </label>
                    <input
                      type="number"
                      id="retentionDays"
                      value={formData.retentionDays}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          retentionDays: parseInt(e.target.value),
                        })
                      }
                      className="mt-1 block w-full px-4 py-2 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm"
                      min="1"
                      required
                    />
                  </div>

                  <div className="space-y-3">
                    <div className="flex items-center">
                      <input
                        type="checkbox"
                        id="enabled"
                        checked={formData.enabled}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            enabled: e.target.checked,
                          })
                        }
                        className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700"
                      />
                      <label
                        htmlFor="enabled"
                        className="ml-2 block text-sm text-gray-900 dark:text-gray-300"
                      >
                        Enable monitoring
                      </label>
                    </div>

                    {!agent && (
                      <div className="flex items-center">
                        <input
                          type="checkbox"
                          id="importHistorical"
                          checked={importHistorical}
                          onChange={(e) =>
                            setImportHistorical(e.target.checked)
                          }
                          className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700"
                        />
                        <label
                          htmlFor="importHistorical"
                          className="ml-2 block text-sm text-gray-900 dark:text-gray-300"
                        >
                          Import historical data
                        </label>
                        <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">
                          (one-time import)
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="mt-6 flex justify-end space-x-3">
                    <Button
                      type="button"
                      className="bg-gray-200 hover:bg-gray-300 text-gray-800 border-gray-300"
                      onClick={onCancel}
                      disabled={isSubmitting}
                    >
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      className="bg-blue-500 hover:bg-blue-600 text-white border-blue-600"
                      disabled={isSubmitting}
                    >
                      {isSubmitting ? "Saving..." : agent ? "Update" : "Create"}
                    </Button>
                  </div>
                </form>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
};
