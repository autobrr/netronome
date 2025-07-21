/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { Input, Select, Button as HeadlessButton } from "@headlessui/react";
import {
  SHOUTRRR_SERVICES,
  type NotificationChannelInput,
} from "@/api/notifications";

interface AddChannelFormProps {
  onSubmit: (input: NotificationChannelInput) => void;
  onCancel: () => void;
  isLoading: boolean;
}

export const AddChannelForm: React.FC<AddChannelFormProps> = ({
  onSubmit,
  onCancel,
  isLoading,
}) => {
  const [name, setName] = useState("");
  const [service, setService] = useState("");
  const [url, setUrl] = useState("");
  const [showUrlHelp, setShowUrlHelp] = useState(false);

  const selectedService = SHOUTRRR_SERVICES.find((s) => s.value === service);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name && url) {
      onSubmit({ name, url, enabled: true });
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Channel Name
        </label>
        <Input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g., Discord Alerts"
          className="mt-1 w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
          required
        />
      </div>

      <div>
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Service Type
        </label>
        <Select
          value={service}
          onChange={(e) => setService(e.target.value)}
          className="mt-1 w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
        >
          <option value="">Select a service...</option>
          {SHOUTRRR_SERVICES.map((svc) => (
            <option key={svc.value} value={svc.value}>
              {svc.label}
            </option>
          ))}
        </Select>
      </div>

      <div>
        <div className="flex items-center justify-between mb-1">
          <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Service URL
          </label>
          <button
            type="button"
            onClick={() => setShowUrlHelp(!showUrlHelp)}
            className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400"
          >
            {showUrlHelp ? "Hide" : "Show"} format
          </button>
        </div>
        <Input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={selectedService?.example || "service://..."}
          className="mt-1 w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
          required
        />
        {showUrlHelp && selectedService && (
          <div className="mt-2 p-2 bg-blue-500/10 border border-blue-500/30 rounded-lg">
            <p className="text-xs text-blue-700 dark:text-blue-300">
              <span className="font-medium">Format:</span>{" "}
              <code className="font-mono">{selectedService.example}</code>
            </p>
          </div>
        )}
      </div>

      <div className="flex items-center gap-2 pt-2">
        <HeadlessButton
          type="submit"
          disabled={isLoading}
          className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? "Creating..." : "Create Channel"}
        </HeadlessButton>
        <HeadlessButton
          type="button"
          onClick={onCancel}
          className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors"
        >
          Cancel
        </HeadlessButton>
      </div>
    </form>
  );
};
