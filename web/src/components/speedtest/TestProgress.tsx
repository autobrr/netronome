/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { TestProgress as TestProgressType } from "../../types/types";

interface TestProgressProps {
  progress: TestProgressType;
}

export const TestProgress: React.FC<TestProgressProps> = ({ progress }) => {
  const getStatusColor = () => {
    switch (progress.type) {
      case "download":
        return "rgb(59, 130, 246)"; // blue-500
      case "upload":
        return "rgb(16, 185, 129)"; // green-500
      case "ping":
      case "complete":
        return "rgb(245, 158, 11)"; // yellow-500
      default:
        return "rgb(156, 163, 175)"; // gray-400
    }
  };

  const formatSpeed = (speed: number) => {
    if (speed < 1) {
      return `${(speed * 1000).toFixed(0)} Kbps`;
    }
    return `${speed.toFixed(2)} Mbps`;
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-gray-850/95 p-6 rounded-xl shadow-lg mb-6 border border-gray-900"
    >
      <div className="flex flex-col gap-4">
        {/* Test Type Indicator */}
        <div className="flex justify-between items-center">
          <span className="text-gray-400">Test Type:</span>
          <span
            className={`font-medium ${
              progress.isScheduled ? "text-blue-400" : "text-white"
            }`}
          >
            {progress.isScheduled ? "Scheduled Test" : "Manual Test"}
          </span>
        </div>

        {/* Server Info */}
        <div className="flex justify-between items-center">
          <span className="text-gray-400">Testing Server:</span>
          <span className="text-white font-medium">
            {progress.currentServer}
          </span>
        </div>

        {/* Progress Bar */}
        <div className="relative pt-1">
          <div className="flex mb-2 items-center justify-between">
            <div>
              <span className="text-xs font-semibold inline-block py-1 px-2 uppercase rounded-full text-white bg-gray-800">
                {progress.currentTest}
              </span>
            </div>
            <div className="text-right">
              <motion.span
                className="text-xs font-semibold inline-block text-white"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                key={progress.progress}
              >
                {progress.progress.toFixed(1)}%
              </motion.span>
            </div>
          </div>
          <div className="flex h-2 mb-4 overflow-hidden rounded bg-gray-800">
            <motion.div
              initial={{ width: "0%" }}
              animate={{ width: `${progress.progress}%` }}
              transition={{ duration: 0.5, ease: "easeOut" }}
              className="flex flex-col justify-center rounded"
              style={{ backgroundColor: getStatusColor() }}
            />
          </div>
        </div>

        {/* Current Speed */}
        <div className="flex justify-between items-center">
          <span className="text-gray-400">Current Speed:</span>
          <span className="text-white font-medium">
            {formatSpeed(progress.currentSpeed)}
          </span>
        </div>

        {/* Additional Metrics */}
        {progress.latency && (
          <div className="flex justify-between items-center">
            <span className="text-gray-400">Latency:</span>
            <span className="text-white font-medium">{progress.latency}</span>
          </div>
        )}

        {progress.packetLoss !== undefined && (
          <div className="flex justify-between items-center">
            <span className="text-gray-400">Packet Loss:</span>
            <span className="text-white font-medium">
              {progress.packetLoss.toFixed(2)}%
            </span>
          </div>
        )}
      </div>
    </motion.div>
  );
};
