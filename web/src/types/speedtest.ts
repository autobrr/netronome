/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export interface SpeedTestOptions {
  serverIds: string[];
  enableDownload?: boolean;
  enableUpload?: boolean;
  useIperf?: boolean;
  useLibrespeed?: boolean;
  serverHost?: string;
}

export interface SpeedTest {
  type: string;
  serverName: string;
  speed: number;
  progress: number;
  isComplete: boolean;
  isScheduled?: boolean;
}

export interface SpeedTestHistory {
  results: SpeedTest[];
  total: number;
  page: number;
  limit: number;
}
