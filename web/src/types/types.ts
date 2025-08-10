/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export interface Server {
  id: string;
  name: string;
  host: string;
  location: string;
  distance: number;
  country: string;
  sponsor: string;
  latitude: number;
  longitude: number;
  isIperf: boolean;
  isLibrespeed?: boolean;
}

export interface SpeedTestResult {
  id: string;
  serverId: string;
  serverName: string;
  serverHost: string;
  testType: "speedtest" | "iperf3" | "librespeed";
  downloadSpeed: number;
  uploadSpeed: number;
  latency: string;
  jitter?: number;
  createdAt: string;
}

export interface TestProgress {
  currentServer: string;
  currentTest: string;
  currentSpeed: number;
  isComplete: boolean;
  type: string;
  speed: number;
  latency: number;
  isScheduled: boolean;
  progress: number;
  isIperf: boolean;
  isLibrespeed: boolean;
}

export interface Schedule {
  id?: number;
  serverIds: string[];
  interval: string;
  nextRun?: string;
  enabled: boolean;
  options: {
    enableDownload: boolean;
    enableUpload: boolean;
    serverIds: string[];
    useIperf: boolean;
    useLibrespeed?: boolean;
    serverHost: string | undefined;
    serverName?: string | undefined;
  };
}

export interface TestOptions {
  enableDownload: boolean;
  enableUpload: boolean;
  enableJitter: boolean;
  multiServer: boolean;
  useIperf: boolean;
  useLibrespeed?: boolean;
  serverIds?: string[];
  serverHost?: string;
  serverName?: string;
}

export type TimeRange = "1d" | "3d" | "1w" | "1m" | "all";

export type TestType = "speedtest" | "iperf" | "librespeed";

export interface SpeedUpdate {
  isComplete: boolean;
  type: "download" | "upload" | "ping" | "complete";
  speed: number;
  progress: number;
  serverName: string;
  latency?: string;
  isScheduled: boolean;
  testType?: string; // "speedtest", "iperf3", "librespeed"
}

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  limit: number;
  total?: number;
}

export interface SavedIperfServer {
  id: number;
  name: string;
  host: string;
  port: number;
  createdAt: string;
  updatedAt: string;
}

export interface TracerouteHop {
  number: number;
  host: string;
  ip: string;
  rtt1: number;
  rtt2: number;
  rtt3: number;
  timeout: boolean;
  as?: string;
  location?: string;
  countryCode?: string;
}

export interface TracerouteResult {
  destination: string;
  ip: string;
  hops: TracerouteHop[];
  totalHops: number;
  complete: boolean;
}

export interface TracerouteUpdate {
  type: string;
  host: string;
  progress: number;
  isComplete: boolean;
  currentHop: number;
  totalHops: number;
  isScheduled: boolean;
  hops: TracerouteHop[];
  destination: string;
  ip: string;
  terminatedEarly?: boolean;
}

export interface PacketLossMonitor {
  id: number;
  host: string;
  name?: string;
  interval: string; // Changed from number to string
  packetCount: number;
  enabled: boolean;
  threshold: number;
  lastRun?: string; // New field
  nextRun?: string; // New field
  createdAt: string;
  updatedAt: string;
}

export interface PacketLossResult {
  id: number;
  monitorId: number;
  packetLoss: number;
  minRtt: number;
  maxRtt: number;
  avgRtt: number;
  stdDevRtt: number;
  packetsSent: number;
  packetsRecv: number;
  usedMtr?: boolean;
  hopCount?: number;
  mtrData?: string; // JSON string containing MTRData
  privilegedMode?: boolean;
  createdAt: string;
}

export interface MTRHop {
  number: number;
  host: string;
  ip?: string;
  loss: number;
  sent: number;
  recv: number;
  last: number;
  avg: number;
  best: number;
  worst: number;
  stddev: number;
  countryCode?: string;
  as?: string;
}

export interface MTRData {
  destination: string;
  ip: string;
  hopCount: number;
  tests: number;
  hops: MTRHop[];
}

export interface PacketLossUpdate {
  type: string;
  monitorId: number;
  host: string;
  isRunning: boolean;
  isComplete: boolean;
  progress: number;
  packetLoss?: number;
  minRtt?: number;
  maxRtt?: number;
  avgRtt?: number;
  stdDevRtt?: number;
  packetsSent?: number;
  packetsRecv?: number;
  usedMtr?: boolean;
  hopCount?: number;
  error?: string;
}
