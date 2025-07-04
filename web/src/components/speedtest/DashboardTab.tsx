/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { motion } from "motion/react";
import { SpeedTestResult, TimeRange } from "@/types/types";
import { SpeedHistoryChart } from "./SpeedHistoryChart";
import { FaWaveSquare, FaShare, FaArrowDown, FaArrowUp } from "react-icons/fa";
import { IoIosPulse } from "react-icons/io";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";

interface DashboardTabProps {
  latestTest: SpeedTestResult | null;
  tests: SpeedTestResult[];
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
  isPublic?: boolean;
  hasAnyTests?: boolean;
  onShareClick?: () => void;
}

const MetricCard: React.FC<{
  icon: React.ReactNode;
  title: string;
  value: string;
  unit: string;
  average?: string;
}> = ({ icon, title, value, unit, average }) => (
  <div className="bg-gray-50/95 dark:bg-gray-850/95 p-4 rounded-xl border border-gray-200 dark:border-gray-900 shadow-lg">
    <div className="flex items-center gap-3 mb-2">
      <div className="text-gray-600 dark:text-gray-400">{icon}</div>
      <h3 className="text-gray-700 dark:text-gray-300 font-medium">{title}</h3>
    </div>
    <div className="flex items-baseline gap-2">
      <span className="text-2xl font-bold text-gray-900 dark:text-white">
        {value}
      </span>
      <span className="text-gray-600 dark:text-gray-400">{unit}</span>
    </div>
    {average && (
      <div className="mt-1 text-sm text-gray-600 dark:text-gray-400">
        Average: {average} {unit}
      </div>
    )}
  </div>
);

export const DashboardTab: React.FC<DashboardTabProps> = ({
  latestTest,
  tests,
  timeRange,
  onTimeRangeChange,
  isPublic = false,
  hasAnyTests = false,
  onShareClick,
}) => {
  const [displayCount, setDisplayCount] = useState(5);
  const [isRecentTestsOpen] = useState(() => {
    const saved = localStorage.getItem("recent-tests-open");
    return saved === null ? true : saved === "true";
  });

  const displayedTests = tests.slice(0, displayCount);
  const formatSpeed = (speed: number) => {
    if (speed >= 1000) {
      return `${(speed / 1000).toFixed(1)} Gbps`;
    }
    return `${speed.toFixed(0)} Mbps`;
  };

  const calculateAverage = (field: keyof SpeedTestResult): string => {
    if (tests.length === 0) return "N/A";

    const validValues = tests
      .map((test) => {
        const value = test[field];
        if (typeof value === "string") {
          return parseFloat(value.replace("ms", ""));
        }
        return Number(value);
      })
      .filter((value) => !isNaN(value));

    if (validValues.length === 0) return "N/A";

    const avg =
      validValues.reduce((sum, value) => sum + value, 0) / validValues.length;
    return avg.toFixed(2);
  };

  return (
    <div className="space-y-6">
      {/* Latest Results */}
      {hasAnyTests && latestTest && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 20 }}
          transition={{ duration: 0.5 }}
          className="mb-6"
        >
          <h2 className="text-gray-900 dark:text-white text-xl ml-1 font-semibold">
            Latest Run
          </h2>
          <div className="flex justify-between ml-1 items-center text-gray-600 dark:text-gray-400 text-sm mb-4">
            <div>
              Last test run:{" "}
              {latestTest?.createdAt
                ? new Date(latestTest.createdAt).toLocaleString(undefined, {
                    dateStyle: "short",
                    timeStyle: "short",
                  })
                : "N/A"}
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6 cursor-default relative">
            <MetricCard
              icon={<IoIosPulse className="w-5 h-5 text-amber-500" />}
              title="Latency"
              value={parseFloat(latestTest.latency).toFixed(2)}
              unit="ms"
              average={calculateAverage("latency")}
            />
            <MetricCard
              icon={<FaArrowDown className="w-5 h-5 text-blue-500" />}
              title="Download"
              value={latestTest.downloadSpeed.toFixed(2)}
              unit="Mbps"
              average={calculateAverage("downloadSpeed")}
            />
            <MetricCard
              icon={<FaArrowUp className="w-5 h-5 text-emerald-500" />}
              title="Upload"
              value={latestTest.uploadSpeed.toFixed(2)}
              unit="Mbps"
              average={calculateAverage("uploadSpeed")}
            />
            <MetricCard
              icon={<FaWaveSquare className="w-5 h-5 text-purple-400" />}
              title="Jitter"
              value={latestTest.jitter?.toFixed(2) ?? "N/A"}
              unit="ms"
              average={
                latestTest.jitter ? calculateAverage("jitter") : undefined
              }
            />

            {/* Floating Share Button over Jitter Card */}
            {!isPublic && onShareClick && (
              <motion.button
                onClick={onShareClick}
                className="absolute top-3 right-3 p-2 bg-blue-500/20 hover:bg-blue-500/30 border border-blue-500/30 hover:border-blue-500/50 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 rounded-lg transition-all duration-200 backdrop-blur-sm z-10 opacity-80 hover:opacity-100"
                aria-label="Share public speed test page"
              >
                <FaShare className="w-2.5 h-2.5" />
              </motion.button>
            )}
          </div>
        </motion.div>
      )}

      {/* Speed History Chart */}
      {hasAnyTests && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 20 }}
          transition={{ duration: 0.5 }}
        >
          <SpeedHistoryChart
            timeRange={timeRange}
            onTimeRangeChange={onTimeRangeChange}
            isPublic={isPublic}
            hasAnyTests={hasAnyTests}
            hasCurrentRangeTests={tests.length > 0}
          />
        </motion.div>
      )}

      {/* Recent Tests Summary */}
      {tests.length > 0 && (
        <Disclosure defaultOpen={isRecentTestsOpen}>
          {({ open }) => {
            useEffect(() => {
              localStorage.setItem("recent-tests-open", open.toString());
            }, [open]);

            return (
              <div className="flex flex-col h-full">
                <DisclosureButton
                  className={`flex justify-between items-center w-full px-4 py-2 bg-gray-50/95 dark:bg-gray-850/95 ${
                    open ? "rounded-t-xl border-b-0" : "rounded-xl"
                  } shadow-lg border-b-0 border-gray-200 dark:border-gray-900 text-left`}
                >
                  <h2 className="text-gray-900 dark:text-white text-xl font-semibold p-1 select-none">
                    Recent Tests
                  </h2>
                  <ChevronDownIcon
                    className={`${
                      open ? "transform rotate-180" : ""
                    } w-5 h-5 text-gray-600 dark:text-gray-400 transition-transform duration-200`}
                  />
                </DisclosureButton>

                {open && (
                  <motion.div
                    initial={{ opacity: 0, y: -20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{
                      duration: 0.5,
                      type: "spring",
                      stiffness: 300,
                      damping: 20,
                    }}
                    className="bg-gray-50/95 dark:bg-gray-850/95 px-4 pt-3 pb-6 rounded-b-xl shadow-lg flex-1"
                  >
                    {/* Desktop Table View */}
                    <div className="hidden md:block overflow-x-auto">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b border-gray-300 dark:border-gray-800">
                            <th className="text-left py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Date
                            </th>
                            <th className="text-left py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Server
                            </th>
                            <th className="text-left py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Type
                            </th>
                            <th className="text-right py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Latency
                            </th>
                            <th className="text-right py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Jitter
                            </th>
                            <th className="text-right py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Download
                            </th>
                            <th className="text-right py-3 px-2 text-gray-600 dark:text-gray-400 font-medium">
                              Upload
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {displayedTests.map((test) => (
                            <motion.tr
                              key={test.id}
                              initial={{ opacity: 0, y: 20 }}
                              animate={{ opacity: 1, y: 0 }}
                              transition={{ duration: 0.3 }}
                              className="border-b border-gray-300/50 dark:border-gray-800/50 last:border-0 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors"
                            >
                              <td className="py-3 px-2 text-gray-700 dark:text-gray-300">
                                {new Date(test.createdAt).toLocaleString(
                                  undefined,
                                  {
                                    month: "short",
                                    day: "numeric",
                                    hour: "2-digit",
                                    minute: "2-digit",
                                  }
                                )}
                              </td>
                              <td
                                className="py-3 px-2 text-gray-700 dark:text-gray-300 truncate max-w-[150px]"
                                title={test.serverName}
                              >
                                {test.serverName}
                              </td>
                              <td className="py-3 px-2">
                                <span
                                  className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                                    test.testType === "iperf3"
                                      ? "bg-purple-500/10 text-purple-600 dark:text-purple-400"
                                      : test.testType === "librespeed"
                                      ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                                      : "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                                  }`}
                                >
                                  {test.testType === "iperf3"
                                    ? "iperf3"
                                    : test.testType === "librespeed"
                                    ? "LibreSpeed"
                                    : "Speedtest.net"}
                                </span>
                              </td>
                              <td className="py-3 px-2 text-right text-amber-600 dark:text-amber-400 font-mono">
                                {parseFloat(test.latency).toFixed(1)}ms
                              </td>
                              <td className="py-3 px-2 text-right text-purple-600 dark:text-purple-400 font-mono">
                                {test.jitter
                                  ? `${test.jitter.toFixed(1)}ms`
                                  : "—"}
                              </td>
                              <td className="py-3 px-2 text-right text-green-600 dark:text-green-400 font-mono">
                                {formatSpeed(test.downloadSpeed)}
                              </td>
                              <td className="py-3 px-2 text-right text-blue-600 dark:text-blue-400 font-mono">
                                {formatSpeed(test.uploadSpeed)}
                              </td>
                            </motion.tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    {/* Mobile Card View */}
                    <div className="md:hidden space-y-3">
                      {displayedTests.map((test) => (
                        <motion.div
                          key={test.id}
                          initial={{ opacity: 0, y: 20 }}
                          animate={{ opacity: 1, y: 0 }}
                          transition={{ duration: 0.3 }}
                          className="bg-gray-200/50 dark:bg-gray-800/50 rounded-lg p-4 border border-gray-300 dark:border-gray-800"
                        >
                          <div className="flex items-center justify-between mb-3">
                            <div className="text-gray-700 dark:text-gray-300 text-sm font-medium truncate flex-1 mr-2">
                              {test.serverName}
                            </div>
                            <span
                              className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium flex-shrink-0 ${
                                test.testType === "iperf3"
                                  ? "bg-purple-500/10 text-purple-600 dark:text-purple-400"
                                  : test.testType === "librespeed"
                                  ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                                  : "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                              }`}
                            >
                              {test.testType === "iperf3"
                                ? "iperf3"
                                : test.testType === "librespeed"
                                ? "LibreSpeed"
                                : "Speedtest.net"}
                            </span>
                          </div>
                          <div className="text-gray-500 dark:text-gray-500 text-xs mb-3">
                            {new Date(test.createdAt).toLocaleString(
                              undefined,
                              {
                                month: "short",
                                day: "numeric",
                                hour: "2-digit",
                                minute: "2-digit",
                              }
                            )}
                          </div>
                          <div className="grid grid-cols-2 gap-3 text-sm">
                            <div className="flex justify-between">
                              <span className="text-gray-600 dark:text-gray-400">
                                Latency:
                              </span>
                              <span className="text-amber-600 dark:text-amber-400 font-mono">
                                {parseFloat(test.latency).toFixed(1)}ms
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-600 dark:text-gray-400">
                                Jitter:
                              </span>
                              <span className="text-purple-600 dark:text-purple-400 font-mono">
                                {test.jitter
                                  ? `${test.jitter.toFixed(1)}ms`
                                  : "—"}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-600 dark:text-gray-400">
                                Download:
                              </span>
                              <span className="text-green-600 dark:text-green-400 font-mono">
                                {formatSpeed(test.downloadSpeed)}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-600 dark:text-gray-400">
                                Upload:
                              </span>
                              <span className="text-blue-600 dark:text-blue-400 font-mono">
                                {formatSpeed(test.uploadSpeed)}
                              </span>
                            </div>
                          </div>
                        </motion.div>
                      ))}
                    </div>
                    {/* Test Count and Load More */}
                    {tests.length > 5 && (
                      <div className="mt-4 space-y-3">
                        {/* Test Count */}
                        <div className="text-center">
                          <span className="text-gray-500 dark:text-gray-500 text-sm">
                            Showing {displayedTests.length} of {tests.length}{" "}
                            tests
                          </span>
                        </div>

                        {/* Load More / Show Less Buttons */}
                        <div className="flex items-center justify-center gap-3">
                          {tests.length > displayCount && (
                            <button
                              onClick={() =>
                                setDisplayCount((prev) => prev + 5)
                              }
                              className="inline-flex items-center px-4 py-2 bg-blue-600/10 hover:bg-blue-600/20 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 rounded-lg transition-colors duration-200 text-sm font-medium"
                            >
                              Load {Math.min(5, tests.length - displayCount)}{" "}
                              more
                              <span className="ml-2">↓</span>
                            </button>
                          )}

                          {displayCount > 5 && (
                            <button
                              onClick={() => setDisplayCount(5)}
                              className="inline-flex items-center px-4 py-2 bg-gray-600/10 hover:bg-gray-600/20 text-gray-600 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg transition-colors duration-200 text-sm font-medium"
                            >
                              Show less
                              <span className="ml-2">↑</span>
                            </button>
                          )}
                        </div>
                      </div>
                    )}
                  </motion.div>
                )}
              </div>
            );
          }}
        </Disclosure>
      )}
    </div>
  );
};
