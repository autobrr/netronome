/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";

export interface VnstatAgent {
  id: number;
  name: string;
  url: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface VnstatStatus {
  connected: boolean;
  liveData?: {
    rx: {
      bytespersecond: number;
      packetspersecond: number;
      ratestring: string;
    };
    tx: {
      bytespersecond: number;
      packetspersecond: number;
      ratestring: string;
    };
  };
}

export interface CreateAgentRequest {
  name: string;
  url: string;
  enabled: boolean;
}

export interface UpdateAgentRequest extends CreateAgentRequest {
  id: number;
}

// Agent management
export async function getVnstatAgents(): Promise<VnstatAgent[]> {
  const response = await fetch(getApiUrl("/vnstat/agents"));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agents");
  }
  const data = await response.json();
  return Array.isArray(data) ? data : [];
}

export async function getVnstatAgent(id: number): Promise<VnstatAgent> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agent");
  }
  return response.json();
}

export async function createVnstatAgent(
  data: CreateAgentRequest,
): Promise<VnstatAgent> {
  const response = await fetch(getApiUrl("/vnstat/agents"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to create agent");
  }
  return response.json();
}

export async function updateVnstatAgent(
  id: number,
  data: UpdateAgentRequest,
): Promise<VnstatAgent> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to update agent");
  }
  return response.json();
}

export async function deleteVnstatAgent(id: number): Promise<void> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}`), {
    method: "DELETE",
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to delete agent");
  }
}

// Agent monitoring
export async function getVnstatAgentStatus(id: number): Promise<VnstatStatus> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}/status`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agent status");
  }
  return response.json();
}

export async function startVnstatAgent(id: number): Promise<void> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}/start`), {
    method: "POST",
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to start agent");
  }
}

export async function stopVnstatAgent(id: number): Promise<void> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}/stop`), {
    method: "POST",
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to stop agent");
  }
}

// Native vnstat data types
export interface VnstatNativeData {
  vnstatversion: string;
  jsonversion: string;
  interfaces: VnstatInterface[];
  server_time?: string; // RFC3339 formatted server time
  server_time_unix?: number; // Unix timestamp
  timezone_offset?: number; // Offset in seconds from UTC
}

export interface VnstatInterface {
  name: string;
  alias: string;
  traffic: VnstatTraffic;
}

export interface VnstatTraffic {
  total: {
    rx: number;
    tx: number;
  };
  hour: VnstatPeriod[];
  day: VnstatPeriod[];
  month: VnstatPeriod[];
}

export interface VnstatPeriod {
  id: number;
  date: {
    year: number;
    month: number;
    day?: number;
  };
  time?: {
    hour: number;
    minute: number;
  };
  rx: number;
  tx: number;
}

export interface VnstatUsageSummary {
  download: number;
  upload: number;
  total: number;
}

// Native vnstat data fetching
export async function getVnstatAgentNative(
  id: number,
  interfaceName?: string,
): Promise<VnstatNativeData> {
  const queryParams = new URLSearchParams();
  if (interfaceName) queryParams.append("interface", interfaceName);

  const url = queryParams.toString()
    ? `${getApiUrl(`/vnstat/agents/${id}/native`)}?${queryParams}`
    : getApiUrl(`/vnstat/agents/${id}/native`);

  const response = await fetch(url);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch native vnstat data");
  }
  return response.json();
}
