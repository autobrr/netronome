export interface Server {
  id: string;
  name: string;
  host: string;
  distance: number;
  country: string;
  sponsor: string;
  cc: string;
  url: string;
  lat: number;
  lon: number;
  provider: string;
}

export interface Schedule {
  id?: number;
  serverIds: string[];
  interval: string;
  nextRun: string;
  enabled: boolean;
  options: {
    enableDownload: boolean;
    enableUpload: boolean;
    enablePacketLoss: boolean;
    serverIds: string[];
  };
}

export interface TestOptions {
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  serverIds: string[];
  isScheduled?: boolean;
}

export type TimeRange = "1d" | "3d" | "1w" | "1m" | "all";

export interface TestProgress {
  type: "download" | "upload" | "ping" | "complete";
  progress: number;
  currentSpeed: number;
  currentServer: string;
  currentTest: "download" | "upload" | "ping" | "complete";
  latency?: string;
  packetLoss?: number;
  isScheduled: boolean;
  speed?: number;
  isComplete: boolean;
}

export interface SpeedTestResult {
  id: number;
  serverId: string;
  serverName: string;
  downloadSpeed: number;
  uploadSpeed: number;
  latency: string;
  jitter?: number;
  packetLoss?: number;
  createdAt: string;
} 