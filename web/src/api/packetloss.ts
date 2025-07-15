/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";
import {
  PacketLossMonitor,
  PacketLossResult,
  PacketLossUpdate,
} from "@/types/types";

export const getPacketLossMonitors = async (): Promise<PacketLossMonitor[]> => {
  const response = await fetch(getApiUrl("/packetloss/monitors"));
  if (!response.ok) {
    throw new Error("Failed to fetch packet loss monitors");
  }
  return response.json();
};

export const createPacketLossMonitor = async (
  monitor: Omit<PacketLossMonitor, "id" | "createdAt" | "updatedAt">,
): Promise<PacketLossMonitor> => {
  const response = await fetch(getApiUrl("/packetloss/monitors"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(monitor),
  });
  if (!response.ok) {
    throw new Error("Failed to create packet loss monitor");
  }
  return response.json();
};

export const updatePacketLossMonitor = async (
  monitor: PacketLossMonitor,
): Promise<PacketLossMonitor> => {
  const response = await fetch(
    getApiUrl(`/packetloss/monitors/${monitor.id}`),
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(monitor),
    },
  );
  if (!response.ok) {
    throw new Error("Failed to update packet loss monitor");
  }
  return response.json();
};

export const deletePacketLossMonitor = async (id: number): Promise<void> => {
  const response = await fetch(getApiUrl(`/packetloss/monitors/${id}`), {
    method: "DELETE",
  });
  if (!response.ok) {
    throw new Error("Failed to delete packet loss monitor");
  }
};

export const getPacketLossMonitorStatus = async (
  id: number,
): Promise<PacketLossUpdate> => {
  const response = await fetch(getApiUrl(`/packetloss/monitors/${id}/status`));
  if (!response.ok) {
    throw new Error("Failed to fetch monitor status");
  }
  return response.json();
};

export const getPacketLossHistory = async (
  id: number,
  limit: number = 100,
): Promise<PacketLossResult[]> => {
  const response = await fetch(
    getApiUrl(`/packetloss/monitors/${id}/history?limit=${limit}`),
  );
  if (!response.ok) {
    throw new Error("Failed to fetch monitor history");
  }
  return response.json();
};

export const startPacketLossMonitor = async (id: number): Promise<void> => {
  const response = await fetch(getApiUrl(`/packetloss/monitors/${id}/start`), {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to start monitor");
  }
};

export const stopPacketLossMonitor = async (id: number): Promise<void> => {
  const response = await fetch(getApiUrl(`/packetloss/monitors/${id}/stop`), {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to stop monitor");
  }
};
