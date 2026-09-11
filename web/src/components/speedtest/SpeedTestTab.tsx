/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { ServerList } from "./ServerList";
import ScheduleManager from "./ScheduleManager";
import { Server, TestOptions, TestType, TestProgress as TestProgressType } from "@/types/types";

interface SpeedTestTabProps {
  servers: Server[];
  selectedServers: Server[];
  onServerSelect: (server: Server) => void;
  options: TestOptions;
  onOptionsChange: (options: TestOptions) => void;
  testType: TestType;
  onTestTypeChange: (type: TestType) => void;
  isLoading: boolean;
  onRunTest: () => Promise<void>;
  progress: TestProgressType | null;
  allServers: Server[];
  isServersLoading?: boolean;
  isServersError?: boolean;
  customUrl?: string;
  onCustomUrlChange?: (url: string) => void;
  downloadThreads?: 2 | 4 | 8;
  onDownloadThreadsChange?: (threads: 2 | 4 | 8) => void;
  downloadTimeout?: number;
  onDownloadTimeoutChange?: (timeout: number) => void;
}

export const SpeedTestTab: React.FC<SpeedTestTabProps> = ({
  servers,
  selectedServers,
  onServerSelect,
  testType,
  onTestTypeChange,
  isLoading,
  onRunTest,
  allServers,
  isServersLoading,
  isServersError,
  customUrl,
  onCustomUrlChange,
  downloadThreads,
  onDownloadThreadsChange,
  downloadTimeout,
  onDownloadTimeoutChange,
}) => {
  return (
    <div className="flex flex-col md:flex-row gap-6 md:items-start">
      {/* Server Selection - Primary Tool */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: 20 }}
        transition={{ duration: 0.5 }}
        className="flex-1"
      >
        <ServerList
          servers={servers}
          selectedServers={selectedServers}
          onSelect={onServerSelect}
          multiSelect={false}
          onMultiSelectChange={() => {}}
          onRunTest={onRunTest}
          isLoading={isLoading}
          testType={testType}
          onTestTypeChange={onTestTypeChange}
          isServersLoading={isServersLoading}
          isServersError={isServersError}
          customUrl={customUrl}
          onCustomUrlChange={onCustomUrlChange}
          downloadThreads={downloadThreads}
          onDownloadThreadsChange={onDownloadThreadsChange}
          downloadTimeout={downloadTimeout}
          onDownloadTimeoutChange={onDownloadTimeoutChange}
        />
      </motion.div>

      {/* Schedule Manager - Automation Tool */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: 20 }}
        transition={{ duration: 0.5 }}
        className="flex-1"
      >
        <ScheduleManager
          servers={allServers}
          selectedServers={selectedServers}
          testType={testType}
          customUrl={customUrl}
          downloadThreads={downloadThreads}
          downloadTimeout={downloadTimeout}
        />
      </motion.div>
    </div>
  );
};