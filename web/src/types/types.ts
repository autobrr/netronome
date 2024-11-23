export interface Server {
  id: string;
  name: string;
  host: string;
  location: string;
  distance: number;
  country: string;
  sponsor: string;
}

export interface SpeedTestResult {
  id: string;
  serverId: string;
  serverName: string;
  serverHost: string;
  downloadSpeed: number;
  uploadSpeed: number;
  latency: string;
  jitter?: number;
  packetLoss?: number;
  createdAt: string;
}

export interface TestProgress {
  currentServer: string;
  currentTest: string;
  currentSpeed: number;
  progress: number;
  isComplete: boolean;
  type: string;
  speed: number;
  latency?: string;
  isScheduled: boolean;
}

export interface Schedule {
  id: string;
  serverId: string;
  serverName: string;
  cron: string;
  enabled: boolean;
  nextRun: string;
  lastRun?: string;
}

export interface TestOptions {
  serverId?: string;
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  multiServer: boolean;
}

export type TimeRange = "1d" | "3d" | "1w" | "1m" | "all";

export interface SpeedUpdate {
  isComplete: boolean;
  type: "download" | "upload" | "ping" | "complete";
  speed: number;
  progress: number;
  serverName: string;
  latency?: string;
  isScheduled: boolean;
}
