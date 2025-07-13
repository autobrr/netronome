/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/Button";
import { VnstatAgent, CreateAgentRequest } from "@/api/vnstat";

interface VnstatAgentFormProps {
  agent?: VnstatAgent | null;
  onSubmit: (data: CreateAgentRequest) => void;
  onCancel: () => void;
  isSubmitting: boolean;
}

export const VnstatAgentForm: React.FC<VnstatAgentFormProps> = ({
  agent,
  onSubmit,
  onCancel,
  isSubmitting,
}) => {
  const [formData, setFormData] = useState<CreateAgentRequest>({
    name: "",
    url: "",
    enabled: true,
    retentionDays: 365,
  });

  useEffect(() => {
    if (agent) {
      setFormData({
        name: agent.name,
        url: agent.url.replace(/\/events\?stream=live-data$/, ""),
        enabled: agent.enabled,
        retentionDays: agent.retentionDays,
      });
    }
  }, [agent]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
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
    <AnimatePresence>
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        {/* Backdrop */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="absolute inset-0 bg-black/50"
          onClick={onCancel}
        />

        {/* Modal */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          className="relative w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-gray-800"
        >
          {/* Header */}
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              {agent ? "Edit Agent" : "Add Agent"}
            </h3>
            <button
              onClick={onCancel}
              className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
            >
              <XMarkIcon className="h-6 w-6" />
            </button>
          </div>

          {/* Form */}
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
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm"
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
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm"
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
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm"
                min="1"
                max="365"
                required
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="enabled"
                checked={formData.enabled}
                onChange={(e) =>
                  setFormData({ ...formData, enabled: e.target.checked })
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
        </motion.div>
      </div>
    </AnimatePresence>
  );
};
