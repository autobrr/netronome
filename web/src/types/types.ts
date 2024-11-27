/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
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
}

export interface SpeedTestResult {
  id: string;
  serverId: string;
  serverName: string;
  serverHost: string;
  testType: 'speedtest' | 'iperf3';
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
  isComplete: boolean;
  type: string;
  speed: number;
  latency: number;
  isScheduled: boolean;
  progress: number;
  isIperf: boolean;
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
    useIperf: boolean;
    serverHost: string | undefined;
  };
}

export interface TestOptions {
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  enableJitter: boolean;
  multiServer: boolean;
  useIperf: boolean;
  serverIds?: string[];
  serverHost?: string;
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

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  limit: number;
  total?: number;
}
