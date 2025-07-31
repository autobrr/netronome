/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
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
import { PacketLossMonitor } from "@/types/types";
import {
  MonitorFormData,
  intervalOptions,
  timeOptions,
} from "./constants/packetLossConstants";
import { formatInterval } from "./utils/packetLossUtils";
import {
  ClockIcon,
  ArrowPathIcon,
  XMarkIcon as XMarkIconMini,
} from "@heroicons/react/20/solid";

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
  const [errors, setErrors] = useState<{ host?: string; name?: string }>({});

  const validateForm = () => {
    const newErrors: { host?: string; name?: string } = {};

    if (!formData.host.trim()) {
      newErrors.host = "Host is required";
    }

    if (!formData.name.trim()) {
      newErrors.name = "Name is required";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setErrors({});
    onSubmit(formData);
  };

  const handleClose = () => {
    setErrors({});
    onClose();
  };

  return (
    <Dialog open={showForm} onOpenChange={handleClose}>
      <DialogContent className="w-full max-w-md bg-white dark:bg-gray-850 border dark:border-gray-900">
        <DialogHeader>
          <DialogTitle className="text-lg font-medium text-gray-900 dark:text-white">
            {editingMonitor ? "Edit Monitor" : "New Monitor"}
          </DialogTitle>
        </DialogHeader>

                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-4">
                    <div>
                      <Label>Host</Label>
                      <Input
                        type="text"
                        value={formData.host}
                        onChange={(e) =>
                          onFormDataChange({
                            ...formData,
                            host: e.target.value,
                          })
                        }
                        placeholder="e.g., google.com or 8.8.8.8"
                        className={errors.host ? "border-red-500 focus:ring-red-500/50" : ""}
                        data-1p-ignore
                        data-lpignore="true"
                        data-form-type="other"
                        autoComplete="off"
                        required
                      />
                      {errors.host && (
                        <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                          {errors.host}
                        </p>
                      )}
                      {!errors.host && (
                        <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                          Some hosts may block ICMP ping requests
                        </p>
                      )}
                    </div>
                    <div>
                      <Label>Name</Label>
                      <Input
                        type="text"
                        value={formData.name}
                        onChange={(e) =>
                          onFormDataChange({
                            ...formData,
                            name: e.target.value,
                          })
                        }
                        placeholder="e.g., Google DNS"
                        className={errors.name ? "border-red-500 focus:ring-red-500/50" : ""}
                        data-1p-ignore
                        data-lpignore="true"
                        data-form-type="other"
                        autoComplete="off"
                        required
                      />
                      {errors.name && (
                        <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                          {errors.name}
                        </p>
                      )}
                    </div>
                    {/* Schedule Type Toggle */}
                    <div className="mb-4">
                      <Label className="mb-2">
                        Schedule Type
                      </Label>
                      <div className="grid grid-cols-2 gap-2 p-1 bg-gray-200/50 dark:bg-gray-800/30 rounded-lg">
                        <button
                          type="button"
                          onClick={() =>
                            onFormDataChange({
                              ...formData,
                              scheduleType: "interval",
                            })
                          }
                          className={`flex items-center justify-center gap-2 px-4 py-2.5 rounded-md font-medium transition-all duration-200 ${
                            formData.scheduleType === "interval"
                              ? "bg-gray-300 dark:bg-gray-700 text-gray-900 dark:text-gray-200 shadow-lg transform scale-105"
                              : "text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 hover:bg-gray-300/50 dark:hover:bg-gray-800/50"
                          }`}
                        >
                          <ArrowPathIcon className="w-4 h-4" />
                          <span>Interval</span>
                        </button>
                        <button
                          type="button"
                          onClick={() =>
                            onFormDataChange({
                              ...formData,
                              scheduleType: "exact",
                            })
                          }
                          className={`flex items-center justify-center gap-2 px-4 py-2.5 rounded-md font-medium transition-all duration-200 ${
                            formData.scheduleType === "exact"
                              ? "bg-gray-300 dark:bg-gray-700 text-gray-900 dark:text-gray-200 shadow-lg transform scale-105"
                              : "text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 hover:bg-gray-300/50 dark:hover:bg-gray-800/50"
                          }`}
                        >
                          <ClockIcon className="w-4 h-4" />
                          <span>Exact Time</span>
                        </button>
                      </div>
                    </div>

                    {/* Interval or Exact Time Selector */}
                    {formData.scheduleType === "interval" ? (
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
                                (opt) => opt.value === formData.interval
                              )?.label ||
                                `Every ${formatInterval(formData.interval)}`}
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
                    ) : (
                      <div>
                        <Label>Test Times</Label>
                        <div className="space-y-2">
                          {(formData.exactTimes || []).map((time, index) => (
                            <div key={index} className="flex gap-2">
                              <Select
                                value={time}
                                onValueChange={(value) => {
                                  const newTimes = [
                                    ...(formData.exactTimes || []),
                                  ];
                                  newTimes[index] = value;
                                  onFormDataChange({
                                    ...formData,
                                    exactTimes: newTimes,
                                  });
                                }}
                              >
                                <SelectTrigger className="flex-1 bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900">
                                  <SelectValue>
                                    {timeOptions.find(
                                      (opt) => opt.value === time
                                    )?.label || time}
                                  </SelectValue>
                                </SelectTrigger>
                                <SelectContent>
                                  {timeOptions.map((option) => {
                                    const isAlreadySelected =
                                      formData.exactTimes?.includes(
                                        option.value
                                      ) && option.value !== time;
                                    return (
                                      <SelectItem
                                        key={option.value}
                                        value={option.value}
                                        disabled={isAlreadySelected}
                                      >
                                        <div className="flex items-center justify-between w-full">
                                          <span>{option.label}</span>
                                          {isAlreadySelected && (
                                            <span className="text-xs text-gray-500 dark:text-gray-500 ml-2">
                                              Already selected
                                            </span>
                                          )}
                                        </div>
                                      </SelectItem>
                                    );
                                  })}
                                </SelectContent>
                              </Select>
                              <button
                                type="button"
                                onClick={() => {
                                  const newTimes = (
                                    formData.exactTimes || []
                                  ).filter((_, i) => i !== index);
                                  onFormDataChange({
                                    ...formData,
                                    exactTimes: newTimes,
                                  });
                                }}
                                className="p-2 text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                              >
                                <XMarkIconMini className="h-5 w-5" />
                              </button>
                            </div>
                          ))}
                          <button
                            type="button"
                            onClick={() => {
                              // Find the first time that's not already selected
                              const availableTime =
                                timeOptions.find(
                                  (opt) =>
                                    !formData.exactTimes?.includes(opt.value)
                                )?.value || "09:00";

                              const newTimes = [
                                ...(formData.exactTimes || []),
                                availableTime,
                              ];
                              onFormDataChange({
                                ...formData,
                                exactTimes: newTimes,
                              });
                            }}
                            className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
                            disabled={
                              formData.exactTimes?.length === timeOptions.length
                            }
                          >
                            + Add another time
                          </button>
                        </div>
                      </div>
                    )}

                    <div>
                      <Label>Packets per Test</Label>
                      <Input
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
                      />
                    </div>
                    <div>
                      <Label>Alert Threshold (% packet loss)</Label>
                      <Input
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
                      />
                    </div>
                    <div className="flex items-center space-x-2">
                      <Checkbox
                        id="enabled"
                        checked={formData.enabled}
                        onCheckedChange={(checked) =>
                          onFormDataChange({
                            ...formData,
                            enabled: checked as boolean,
                          })
                        }
                      />
                      <Label htmlFor="enabled" className="cursor-pointer">
                        Start monitoring immediately
                      </Label>
                    </div>
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
                    <Button
                      type="button"
                      onClick={handleClose}
                      variant="secondary"
                    >
                      Cancel
                    </Button>
                  </div>
                </form>
      </DialogContent>
    </Dialog>
  );
};
