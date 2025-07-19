/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";

export interface MonitorAgent {
  id: number;
  name: string;
  url: string;
  apiKey?: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
  isTailscale?: boolean;
  tailscaleHostname?: string;
  discoveredAt?: string;
}

export interface MonitorStatus {
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

export interface InterfaceInfo {
  name: string;
  alias: string;
  ip_address: string;
  link_speed: number; // Mbps
  is_up: boolean;
  bytes_total: number;
}

export interface SystemInfo {
  hostname: string;
  kernel: string;
  uptime: number; // seconds
  interfaces: Record<string, InterfaceInfo>;
  vnstat_version: string;
  database_size: number; // bytes
  updated_at: string;
  from_cache?: boolean; // True when data is from database, not live agent
}

export interface PeakStats {
  peak_rx: number; // bytes/s
  peak_tx: number; // bytes/s
  peak_rx_string: string;
  peak_tx_string: string;
  peak_rx_timestamp?: string;
  peak_tx_timestamp?: string;
  updated_at: string;
}

export interface HardwareStats {
  cpu: CPUStats;
  memory: MemoryStats;
  disks: DiskStats[];
  temperature?: TemperatureStats[];
  updated_at: string;
  from_cache?: boolean; // True when data is from database, not live agent
}

export interface CPUStats {
  usage_percent: number;
  cores: number;
  threads: number;
  model: string;
  frequency: number; // MHz
  load_avg?: number[]; // 1, 5, 15 min
}

export interface MemoryStats {
  total: number;
  used: number;
  free: number;
  available: number;
  used_percent: number;
  cached: number;
  buffers: number;
  zfs_arc: number;
  swap_total: number;
  swap_used: number;
  swap_percent: number;
}

export interface DiskStats {
  path: string;
  device: string;
  fstype: string;
  total: number;
  used: number;
  free: number;
  used_percent: number;
  model?: string;
  serial?: string;
}

export interface TemperatureStats {
  sensor_key: string;
  temperature: number; // Celsius
  label?: string;
  critical?: number;
}

export interface CreateAgentRequest {
  name: string;
  url: string;
  apiKey?: string;
  enabled: boolean;
}

export interface UpdateAgentRequest extends CreateAgentRequest {
  id: number;
}

// Agent management
export async function getMonitorAgents(): Promise<MonitorAgent[]> {
  const response = await fetch(getApiUrl("/monitor/agents"));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agents");
  }
  const data = await response.json();
  return Array.isArray(data) ? data : [];
}

export async function getMonitorAgent(id: number): Promise<MonitorAgent> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agent");
  }
  return response.json();
}

export async function createMonitorAgent(
  data: CreateAgentRequest,
): Promise<MonitorAgent> {
  const response = await fetch(getApiUrl("/monitor/agents"), {
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

export async function updateMonitorAgent(
  id: number,
  data: UpdateAgentRequest,
): Promise<MonitorAgent> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}`), {
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

export async function deleteMonitorAgent(id: number): Promise<void> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}`), {
    method: "DELETE",
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to delete agent");
  }
}

// Agent monitoring
export async function getMonitorAgentStatus(id: number): Promise<MonitorStatus> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}/status`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agent status");
  }
  return response.json();
}

export async function startMonitorAgent(id: number): Promise<void> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}/start`), {
    method: "POST",
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to start agent");
  }
}

export async function stopMonitorAgent(id: number): Promise<void> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}/stop`), {
    method: "POST",
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to stop agent");
  }
}

// Native vnstat data types
export interface MonitorNativeData {
  vnstatversion: string;
  jsonversion: string;
  interfaces: MonitorInterface[];
  server_time?: string; // RFC3339 formatted server time
  server_time_unix?: number; // Unix timestamp
  timezone_offset?: number; // Offset in seconds from UTC
  from_cache?: boolean; // True when data is from database, not live agent
  cache_timestamp?: string; // When the cached data was saved
}

export interface MonitorInterface {
  name: string;
  alias: string;
  traffic: MonitorTraffic;
}

export interface MonitorTraffic {
  total: {
    rx: number;
    tx: number;
  };
  hour: MonitorPeriod[];
  day: MonitorPeriod[];
  month: MonitorPeriod[];
}

export interface MonitorPeriod {
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

export interface MonitorUsageSummary {
  download: number;
  upload: number;
  total: number;
}

// Native vnstat data fetching
export async function getMonitorAgentNative(
  id: number,
  interfaceName?: string,
): Promise<MonitorNativeData> {
  const queryParams = new URLSearchParams();
  if (interfaceName) queryParams.append("interface", interfaceName);

  const url = queryParams.toString()
    ? `${getApiUrl(`/monitor/agents/${id}/native`)}?${queryParams}`
    : getApiUrl(`/monitor/agents/${id}/native`);

  const response = await fetch(url);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch native vnstat data");
  }
  return response.json();
}

// System info fetching
export async function getMonitorAgentSystemInfo(
  id: number,
): Promise<SystemInfo> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}/system`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch system info");
  }
  return response.json();
}

// Hardware stats fetching
export async function getMonitorAgentHardwareStats(
  id: number,
): Promise<HardwareStats> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}/hardware`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch hardware stats");
  }
  return response.json();
}

// Peak stats fetching
export async function getMonitorAgentPeakStats(
  id: number,
): Promise<PeakStats> {
  const response = await fetch(getApiUrl(`/monitor/agents/${id}/peaks`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch peak stats");
  }
  return response.json();
}
