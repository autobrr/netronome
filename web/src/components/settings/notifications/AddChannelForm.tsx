/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
        <Label>
          Channel Name
        </Label>
        <Input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g., Discord Alerts"
          className="mt-1 w-full"
          required
        />
      </div>

      <div>
        <Label>
          Service Type
        </Label>
        <Select value={service} onValueChange={setService}>
          <SelectTrigger className="mt-1 w-full">
            <SelectValue placeholder="Select a service..." />
          </SelectTrigger>
          <SelectContent>
            {SHOUTRRR_SERVICES.map((svc) => (
              <SelectItem key={svc.value} value={svc.value}>
                {svc.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div>
        <div className="flex items-center justify-between mb-1">
          <Label>
            Service URL
          </Label>
          <button
            type="button"
            onClick={() => setShowUrlHelp(!showUrlHelp)}
            className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 transition-colors"
          >
            {showUrlHelp ? "Hide" : "Show"} format
          </button>
        </div>
        <Input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={selectedService?.example || "service://..."}
          className="mt-1 w-full"
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
        <Button
          type="submit"
          isLoading={isLoading}
          className="flex-1"
        >
          {isLoading ? "Creating..." : "Create Channel"}
        </Button>
        <Button
          type="button"
          onClick={onCancel}
          variant="secondary"
          className="flex-1"
        >
          Cancel
        </Button>
      </div>
    </form>
  );
};
