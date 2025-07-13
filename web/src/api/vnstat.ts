import { getApiUrl } from "@/utils/baseUrl";

export interface VnstatAgent {
  id: number;
  name: string;
  url: string;
  enabled: boolean;
  retention_days: number;
  created_at: string;
  updated_at: string;
}

export interface VnstatBandwidth {
  id: number;
  agentId: number;
  rxBytesPerSecond?: number;
  txBytesPerSecond?: number;
  rxPacketsPerSecond?: number;
  txPacketsPerSecond?: number;
  rxRateString?: string;
  txRateString?: string;
  createdAt: string;
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
  retention_days: number;
}

export interface UpdateAgentRequest extends CreateAgentRequest {
  id: number;
}

export interface BandwidthQueryParams {
  limit?: number;
  start?: string;
  end?: string;
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

export async function getVnstatAgentBandwidth(
  id: number,
  params?: BandwidthQueryParams,
): Promise<VnstatBandwidth[]> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append("limit", params.limit.toString());
  if (params?.start) queryParams.append("start", params.start);
  if (params?.end) queryParams.append("end", params.end);

  const url = queryParams.toString()
    ? `${getApiUrl(`/vnstat/agents/${id}/bandwidth`)}?${queryParams}`
    : getApiUrl(`/vnstat/agents/${id}/bandwidth`);

  const response = await fetch(url);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch bandwidth history");
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

export async function getVnstatAgentUsage(
  id: number,
): Promise<
  Record<string, { download: number; upload: number; total: number }>
> {
  const response = await fetch(getApiUrl(`/vnstat/agents/${id}/usage`));
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch agent usage");
  }
  return response.json();
}
