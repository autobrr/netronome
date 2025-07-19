import { getApiUrl } from "@/utils/baseUrl";

export interface TailscaleStatus {
  enabled: boolean;
  status?: string;
  hostname?: string;
  tailscale_ips?: string[];
  online?: boolean;
  magic_dns_suffix?: string;
  discovered_agents?: number;
}

export interface AgentTailscaleStatus {
  enabled: boolean;
  status: string;
  hostname?: string;
  tailscale_ips?: string[];
  online?: boolean;
}

export const tailscaleAPI = {
  // Get server Tailscale discovery status
  getDiscoveryStatus: async (): Promise<TailscaleStatus> => {
    const response = await fetch(getApiUrl('/api/monitor/tailscale/status'), {
      credentials: 'same-origin',
    });
    if (!response.ok) {
      throw new Error('Failed to fetch Tailscale status');
    }
    return response.json();
  },

  // Get agent Tailscale status
  getAgentStatus: async (agentId: number): Promise<AgentTailscaleStatus> => {
    const response = await fetch(getApiUrl(`/api/monitor/agents/${agentId}/tailscale/status`), {
      credentials: 'same-origin',
    });
    if (!response.ok) {
      throw new Error('Failed to fetch agent Tailscale status');
    }
    return response.json();
  },
};