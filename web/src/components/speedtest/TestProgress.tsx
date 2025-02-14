/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion, AnimatePresence } from "motion/react";
import { TestProgress as TestProgressType } from "@/types/types";
import { FaArrowDown, FaArrowUp } from "react-icons/fa";

interface TestProgressProps {
  progress: TestProgressType;
}

export const TestProgress: React.FC<TestProgressProps> = ({ progress }) => {
  if (progress.isIperf) {
    return (
      <div className="flex items-center justify-center">
        <span className="text-white font-bold text-sm">
          Running iperf3 test...
        </span>
      </div>
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

  const formatSpeed = (speed: number) => {
    if (speed < 1) {
      return `${(speed * 1000).toFixed(0)} Kbps`;
    }
    return `${speed.toFixed(2)} Mbps`;
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

  return (
    <motion.div className="flex justify-end items-center mb-0">
      <motion.div
        className="flex flex-col items-center"
        initial="hidden"
        animate="visible"
        variants={{
          hidden: { opacity: 0 },
          visible: {
            opacity: 1,
            transition: {
              staggerChildren: 0.05,
            },
          },
        }}
      >
        {/* Active Test Display */}
        {progress.type !== "ping" && (
          <div className="flex items-center justify-center">
            {progress.type === "download" ? (
              <FaArrowDown className="text-blue-500" />
            ) : (
              <FaArrowUp className="text-emerald-500" />
            )}
            <span className="text-white font-bold text-sm mr-4">
              {getTestPhase()}
            </span>
          </div>
        )}

        {/* Current Speed Display */}
        {(progress.type === "download" || progress.type === "upload") && (
          <motion.div className="flex items-center justify-center mt-1">
            <span className="text-gray-400 font-semibold text-sm">
              Current Speed:
            </span>
            <div
              className="text-white font-bold text-lg ml-1"
              style={{
                width: "120px",
                whiteSpace: "nowrap",
                overflow: "hidden",
                textOverflow: "ellipsis",
              }}
            >
              <AnimatePresence mode="wait">
                <motion.div
                  key={progress.currentSpeed}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{
                    opacity: 1,
                    y: 0,
                    scale: [1, 1.05, 1], // Subtle pop effect
                  }}
                  exit={{ opacity: 0, y: -10 }}
                  transition={{
                    type: "spring",
                    stiffness: 400,
                    damping: 40,
                    mass: 0.8,
                    scale: {
                      duration: 0.2,
                    },
                  }}
                  className={getStatusColor()}
                >
                  {formatSpeed(progress.currentSpeed)}
                </motion.div>
              </AnimatePresence>
            </div>
          </motion.div>
        )}
      </motion.div>
    </motion.div>
  );
};
