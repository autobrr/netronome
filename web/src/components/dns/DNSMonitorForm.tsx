/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { DNSMonitor, DNSMonitorInput, DNSProtocol } from "@/types/types";
import { intervalOptions } from "@/components/speedtest/packetloss/constants/packetLossConstants";
import { formatInterval } from "@/components/speedtest/packetloss/utils/packetLossUtils";
import { protocolOptions, recordTypes, resolverPresets } from "./constants";

interface DNSMonitorFormProps {
  showForm: boolean;
  onClose: () => void;
  onSubmit: (data: DNSMonitorInput) => void;
  editingMonitor?: DNSMonitor | null;
  formData: DNSMonitorInput;
  onFormDataChange: (data: DNSMonitorInput) => void;
  isLoading?: boolean;
}

export const DNSMonitorForm: React.FC<DNSMonitorFormProps> = ({
  showForm,
  onClose,
  onSubmit,
  editingMonitor,
  formData,
  onFormDataChange,
  isLoading = false,
}) => {
  // the resolver input is `required`, so the browser blocks an empty submit
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({ ...formData, host: formData.host.trim() });
  };

  return (
    <Dialog open={showForm} onOpenChange={onClose}>
      <DialogContent className="w-full max-w-md bg-white dark:bg-gray-850 border dark:border-gray-900 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
            {editingMonitor ? "Edit DNS Monitor" : "New DNS Monitor"}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label className="mb-2">Presets</Label>
            <div className="flex flex-wrap gap-2">
              {resolverPresets.map((preset) => (
                <button
                  key={preset.label}
                  type="button"
                  onClick={() =>
                    onFormDataChange({
                      ...formData,
                      host: preset.host,
                      protocol: preset.protocol,
                      name: formData.name.trim() || preset.label,
                    })
                  }
                  className={`px-2.5 py-1 rounded-md text-xs border transition-colors ${
                    formData.host === preset.host &&
                    formData.protocol === preset.protocol
                      ? "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/30"
                      : "bg-gray-200/50 dark:bg-gray-800/50 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-700/50"
                  }`}
                >
                  {preset.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <Label>Resolver</Label>
            <Input
              type="text"
              value={formData.host}
              onChange={(e) =>
                onFormDataChange({ ...formData, host: e.target.value })
              }
              placeholder="e.g., 192.168.1.1 or dns.example.com:5353"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
              autoComplete="off"
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
              Port 53 by default, 853 for DNS over TLS. Add :port to override.
            </p>
          </div>

          <div>
            <Label>Name</Label>
            <Input
              type="text"
              value={formData.name}
              onChange={(e) =>
                onFormDataChange({ ...formData, name: e.target.value })
              }
              placeholder="e.g., Pi-hole"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
              autoComplete="off"
            />
          </div>

          <div>
            <Label>Protocol</Label>
            <Select
              value={formData.protocol}
              onValueChange={(value) =>
                onFormDataChange({
                  ...formData,
                  protocol: value as DNSProtocol,
                })
              }
            >
              <SelectTrigger className="w-full bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {protocolOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Query</Label>
            <Input
              type="text"
              value={formData.query}
              onChange={(e) =>
                onFormDataChange({ ...formData, query: e.target.value })
              }
              placeholder="google.com"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
              autoComplete="off"
            />
          </div>

          <div>
            <Label>Record Type</Label>
            <Select
              value={formData.recordType}
              onValueChange={(value) =>
                onFormDataChange({ ...formData, recordType: value })
              }
            >
              <SelectTrigger className="w-full bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {recordTypes.map((type) => (
                  <SelectItem key={type} value={type}>
                    {type}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Check Interval</Label>
            <Select
              value={formData.interval}
              onValueChange={(value) =>
                onFormDataChange({ ...formData, interval: value })
              }
            >
              <SelectTrigger className="w-full bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900">
                <SelectValue>
                  {intervalOptions.find(
                    (option) => option.value === formData.interval,
                  )?.label || `Every ${formatInterval(formData.interval)}`}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {intervalOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="dns-enabled"
              checked={formData.enabled}
              onCheckedChange={(checked) =>
                onFormDataChange({ ...formData, enabled: checked as boolean })
              }
            />
            <Label htmlFor="dns-enabled" className="cursor-pointer">
              Start monitoring immediately
            </Label>
          </div>

          <div className="flex gap-3 pt-4">
            <Button
              type="submit"
              disabled={isLoading}
              isLoading={isLoading}
              variant="default"
              className="flex-1"
            >
              {editingMonitor ? "Update Monitor" : "Create Monitor"}
            </Button>
            <Button type="button" onClick={onClose} variant="secondary">
              Cancel
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
