/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export interface IntervalOption {
  value: number;
  label: string;
}

export const intervalOptions: IntervalOption[] = [
  { value: 10, label: "Every 10 seconds" },
  { value: 30, label: "Every 30 seconds" },
  { value: 60, label: "Every 1 minute" },
  { value: 300, label: "Every 5 minutes" },
  { value: 900, label: "Every 15 minutes" },
  { value: 1800, label: "Every 30 minutes" },
  { value: 3600, label: "Every 1 hour" },
  { value: 21600, label: "Every 6 hours" },
  { value: 43200, label: "Every 12 hours" },
  { value: 86400, label: "Every 24 hours" },
];

export interface MonitorFormData {
  host: string;
  name: string;
  interval: number;
  packetCount: number;
  threshold: number;
  enabled: boolean;
}

export const defaultFormData: MonitorFormData = {
  host: "",
  name: "",
  interval: 60,
  packetCount: 10,
  threshold: 5.0,
  enabled: true,
};
