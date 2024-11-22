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
  lastRun?: string;
  nextRun: string;
  enabled: boolean;
  options: TestOptions;
  createdAt?: string;
}

export interface TestOptions {
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  serverIds: string[];
}

export type TimeRange = "1d" | "3d" | "1w" | "1m" | "all";

export interface TestProgress {
  currentServer: string;
  currentTest: string;
  currentSpeed: number;
  progress: number;
  isComplete: boolean;
  type: "download" | "upload" | "ping";
  speed: number;
  latency?: number;
  packetLoss?: number;
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