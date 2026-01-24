/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export interface MonitorStatus {
  isRunning: boolean;
  isComplete: boolean;
  progress?: number;
  packetLoss?: number;
  avgRtt?: number;
  minRtt?: number;
  maxRtt?: number;
  packetsSent?: number;
  packetsRecv?: number;
  usedMtr?: boolean;
}
