/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion, AnimatePresence } from "motion/react";
import { TestProgress as TestProgressType } from "@/types/types";
import { ArrowDownIcon, ArrowUpIcon } from "@heroicons/react/24/outline";

interface TestProgressProps {
  progress: TestProgressType;
}

export const TestProgress: React.FC<TestProgressProps> = ({ progress }) => {
  // Hide ping/latency test phase
  if (progress.type === "ping") {
    return null;
  }

  // Check iperf3 preparing state BEFORE other conditions to prevent flash
  // Show preparing message for iperf3 during the initial ping phase
  // But hide it between download and upload tests
  if (
    progress.isIperf &&
    (progress.progress === 0 || !progress.progress) &&
    (progress.speed === 0 || !progress.speed)
  ) {
    // If we have a currentTest that's not empty, we're between tests
    if (progress.currentTest && progress.currentTest !== "") {
      return null; // Hide during transition between download/upload
    }

    return (
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="flex items-center justify-center space-x-2 text-xs w-full max-w-[16rem]"
      >
        <div className="flex space-x-1">
          {[0, 1, 2].map((i) => (
            <motion.div
              key={i}
              className="w-1 h-1 bg-gray-600 dark:bg-gray-400 rounded-full"
              animate={{
                opacity: [0.3, 1, 0.3],
              }}
              transition={{
                duration: 1.5,
                repeat: Infinity,
                delay: i * 0.2,
              }}
            />
          ))}
        </div>
        <span className="text-gray-700 dark:text-gray-300 font-medium whitespace-nowrap">
          Preparing test...
        </span>
      </motion.div>
    );
  }

  // Show animated message for LibreSpeed
  if (progress.isLibrespeed) {
    return (
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="flex items-center justify-center space-x-2 text-xs w-full max-w-[16rem]"
      >
        <div className="flex space-x-1">
          {[0, 1, 2].map((i) => (
            <motion.div
              key={i}
              className="w-1 h-1 bg-gray-600 dark:bg-gray-400 rounded-full"
              animate={{
                opacity: [0.3, 1, 0.3],
              }}
              transition={{
                duration: 1.5,
                repeat: Infinity,
                delay: i * 0.2,
              }}
            />
          ))}
        </div>
        <span className="text-gray-700 dark:text-gray-300 font-medium whitespace-nowrap">
          Running LibreSpeed test...
        </span>
      </motion.div>
    );
  }

  const getStatusColor = () => {
    switch (progress.type) {
      case "download":
        return "text-blue-500";
      case "upload":
        return "text-emerald-500";
      case "ping":
      case "complete":
        return "text-yellow-500";
      default:
        return "text-gray-400";
    }
  };

  const getTestPhase = () => {
    switch (progress.type) {
      case "download":
        return "Download Test";
      case "upload":
        return "Upload Test";
      case "ping":
        return "Latency Test";
      default:
        return progress.currentTest;
    }
  };

  // Show test running with live progress
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="flex items-center justify-center space-x-2 text-xs w-full max-w-[16rem] px-2"
    >
      {/* Test type icon */}
      <div className={`${getStatusColor()} flex-shrink-0`}>
        {progress.type === "download" ? (
          <ArrowDownIcon className="w-4 h-4" />
        ) : progress.type === "upload" ? (
          <ArrowUpIcon className="w-4 h-4" />
        ) : null}
      </div>

      {/* Test info */}
      <div className="flex items-baseline space-x-0.5 whitespace-nowrap">
        <span className="text-gray-700 dark:text-gray-300 font-medium">
          {getTestPhase()}:
        </span>

        <div className="relative inline-block w-10 text-right">
          <AnimatePresence>
            <motion.span
              key={progress.currentSpeed}
              initial={{
                opacity: 0,
                filter: "blur(4px)",
              }}
              animate={{
                opacity: 1,
                filter: "blur(0px)",
              }}
              exit={{
                opacity: 0,
                filter: "blur(4px)",
                position: "absolute",
                right: 0,
              }}
              transition={{
                duration: 0.5,
                ease: "easeInOut",
              }}
              className={`font-bold ${getStatusColor()}`}
            >
              {progress.currentSpeed < 1
                ? (progress.currentSpeed * 1000).toFixed(0)
                : progress.currentSpeed.toFixed(1)}
            </motion.span>
          </AnimatePresence>
        </div>

        <span className="text-gray-600 dark:text-gray-400">
          {progress.currentSpeed < 1 ? "Kbps" : "Mbps"}
        </span>

        {progress.progress > 0 && (
          <span className="text-gray-500 dark:text-gray-500">
            ({Math.round(progress.progress)}%)
          </span>
        )}
      </div>
    </motion.div>
  );
};
