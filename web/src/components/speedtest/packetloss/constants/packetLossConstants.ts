/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export interface IntervalOption {
  value: string; // Changed from number to string
  label: string;
}

export const intervalOptions: IntervalOption[] = [
  { value: "10s", label: "Every 10 seconds" },
  { value: "30s", label: "Every 30 seconds" },
  { value: "1m", label: "Every 1 minute" },
  { value: "5m", label: "Every 5 minutes" },
  { value: "15m", label: "Every 15 minutes" },
  { value: "30m", label: "Every 30 minutes" },
  { value: "1h", label: "Every 1 hour" },
  { value: "6h", label: "Every 6 hours" },
  { value: "12h", label: "Every 12 hours" },
  { value: "24h", label: "Every 24 hours" },
];

export interface TimeOption {
  value: string;
  label: string;
}

export const timeOptions: TimeOption[] = [
  { value: "00:00", label: "12:00 AM" },
  { value: "01:00", label: "1:00 AM" },
  { value: "02:00", label: "2:00 AM" },
  { value: "03:00", label: "3:00 AM" },
  { value: "04:00", label: "4:00 AM" },
  { value: "05:00", label: "5:00 AM" },
  { value: "06:00", label: "6:00 AM" },
  { value: "07:00", label: "7:00 AM" },
  { value: "08:00", label: "8:00 AM" },
  { value: "09:00", label: "9:00 AM" },
  { value: "10:00", label: "10:00 AM" },
  { value: "11:00", label: "11:00 AM" },
  { value: "12:00", label: "12:00 PM" },
  { value: "13:00", label: "1:00 PM" },
  { value: "14:00", label: "2:00 PM" },
  { value: "15:00", label: "3:00 PM" },
  { value: "16:00", label: "4:00 PM" },
  { value: "17:00", label: "5:00 PM" },
  { value: "18:00", label: "6:00 PM" },
  { value: "19:00", label: "7:00 PM" },
  { value: "20:00", label: "8:00 PM" },
  { value: "21:00", label: "9:00 PM" },
  { value: "22:00", label: "10:00 PM" },
  { value: "23:00", label: "11:00 PM" },
];

export interface MonitorFormData {
  host: string;
  name: string;
  interval: string; // Changed from number to string
  scheduleType?: "interval" | "exact"; // New field
  exactTimes?: string[]; // New field for exact times
  packetCount: number;
  threshold: number;
  enabled: boolean;
}

export const defaultFormData: MonitorFormData = {
  host: "",
  name: "",
  interval: "30m", // Default to 30 minutes for reasonable monitoring frequency
  scheduleType: "interval",
  exactTimes: [],
  packetCount: 10,
  threshold: 5.0,
  enabled: true,
};
