/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { Fragment, useState } from "react";
import { motion } from "motion/react";
import { ChevronUpDownIcon } from "@heroicons/react/24/solid";
import { PacketLossResult, PacketLossMonitor } from "@/types/types";
import { MTRResultsDisplay } from "./MTRResultsDisplay";
import { formatRTT, parseMTRData } from "../utils/packetLossUtils";

interface MonitorResultsTableProps {
  historyList: PacketLossResult[];
  selectedMonitor: PacketLossMonitor;
}

export const MonitorResultsTable: React.FC<MonitorResultsTableProps> = ({
  historyList,
  selectedMonitor,
}) => {
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());
  const [displayCount, setDisplayCount] = useState(10);

  const toggleExpanded = (resultId: number) => {
    setExpandedRows((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(resultId)) {
        newSet.delete(resultId);
      } else {
        newSet.add(resultId);
      }
      return newSet;
    });
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const month = date.toLocaleDateString("en-US", { month: "short" });
    const day = date.getDate();
    const hours = date.getHours().toString().padStart(2, "0");
    const minutes = date.getMinutes().toString().padStart(2, "0");
    return `${month} ${day}, ${hours}:${minutes}`;
  };

  return (
    <div>
      <h3 className="text-gray-700 dark:text-gray-300 font-medium mb-3">
        Recent Results{" "}
        {historyList.length > 0 && `(${historyList.length} total)`}
      </h3>

      {/* Desktop Table View */}
      <div className="hidden md:block overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-300 dark:border-gray-800">
              <th className="text-left py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Time
              </th>
              <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Loss
              </th>
              <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Avg RTT
              </th>
              <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Min RTT
              </th>
              <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Max RTT
              </th>
              <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Sent/Recv
              </th>
              <th className="text-center py-2 px-3 text-gray-600 dark:text-gray-400 font-medium">
                Mode
              </th>
            </tr>
          </thead>
          <tbody>
            {historyList.slice(0, displayCount).map((result) => {
              const mtrData = parseMTRData(result.mtrData);
              const isExpanded = expandedRows.has(result.id);

              return (
                <Fragment key={result.id}>
                  <motion.tr
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3 }}
                    className={`border-b border-gray-300/50 dark:border-gray-800/50 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors ${
                      result.usedMtr ? "cursor-pointer" : ""
                    }`}
                    onClick={() => {
                      if (result.usedMtr && mtrData) {
                        toggleExpanded(result.id);
                      }
                    }}
                  >
                    <td className="py-2 px-3 text-gray-700 dark:text-gray-300">
                      {formatDate(result.createdAt)}
                    </td>
                    <td
                      className={`py-2 px-3 text-center font-mono ${
                        result.packetLoss > selectedMonitor.threshold
                          ? "text-red-600 dark:text-red-400 font-medium"
                          : "text-emerald-600 dark:text-emerald-400"
                      }`}
                    >
                      {result.packetLoss.toFixed(1)}%
                    </td>
                    <td className="py-2 px-3 text-center font-mono text-blue-600 dark:text-blue-400">
                      {formatRTT(result.avgRtt)}ms
                    </td>
                    <td className="py-2 px-3 text-center font-mono text-emerald-600 dark:text-emerald-400">
                      {formatRTT(result.minRtt)}ms
                    </td>
                    <td className="py-2 px-3 text-center font-mono text-yellow-600 dark:text-yellow-400">
                      {formatRTT(result.maxRtt)}ms
                    </td>
                    <td className="py-2 px-3 text-center text-gray-600 dark:text-gray-400">
                      {result.packetsSent}/{result.packetsRecv}
                    </td>
                    <td className="py-2 px-3 text-center">
                      {result.usedMtr ? (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                          MTR
                          {result.privilegedMode !== undefined && (
                            <span className="ml-1">
                              ({result.privilegedMode ? "ICMP" : "UDP"})
                            </span>
                          )}
                          {result.hopCount && (
                            <span className="hidden 2xl:inline">
                              {" "}
                              - {result.hopCount} hops
                            </span>
                          )}
                        </span>
                      ) : (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
                          ICMP
                        </span>
                      )}
                    </td>
                  </motion.tr>

                  {/* Expandable MTR Details Row */}
                  {result.usedMtr && mtrData && (
                    <MTRResultsDisplay
                      mtrData={mtrData}
                      isExpanded={isExpanded}
                      isMobile={false}
                    />
                  )}
                </Fragment>
              );
            })}
          </tbody>
        </table>

        {/* Load More Button */}
        {historyList.length > displayCount && (
          <div className="flex justify-center mt-4">
            <button
              onClick={() => setDisplayCount((prev) => prev + 10)}
              className="px-4 py-2 bg-gray-200/30 dark:bg-gray-800/30 border border-gray-300/50 dark:border-gray-900/50 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800/50 transition-colors"
            >
              Load More
            </button>
          </div>
        )}
      </div>

      {/* Mobile Card View */}
      <div className="md:hidden space-y-3">
        {historyList.slice(0, displayCount).map((result) => {
          const mtrData = parseMTRData(result.mtrData);
          const isExpanded = expandedRows.has(result.id);

          return (
            <motion.div
              key={result.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className={`bg-gray-200/50 dark:bg-gray-800/50 rounded-lg p-4 border border-gray-300 dark:border-gray-800 ${
                result.usedMtr
                  ? "cursor-pointer hover:bg-gray-200/70 dark:hover:bg-gray-800/70 transition-colors"
                  : ""
              }`}
              onClick={() => {
                if (result.usedMtr && mtrData) {
                  toggleExpanded(result.id);
                }
              }}
            >
              <div className="flex items-center justify-between mb-3">
                <div className="text-gray-700 dark:text-gray-300 text-sm font-medium">
                  {formatDate(result.createdAt)}
                </div>
                <div className="flex items-center gap-2">
                  {result.usedMtr && (
                    <>
                      <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-500/10 text-purple-600 dark:text-purple-400">
                        MTR
                        {result.privilegedMode !== undefined && (
                          <span className="ml-1 text-xs">
                            ({result.privilegedMode ? "ICMP" : "UDP"})
                          </span>
                        )}
                      </span>
                      <motion.div
                        animate={{
                          rotate: isExpanded ? 180 : 0,
                        }}
                        transition={{ duration: 0.2 }}
                        className="text-gray-500 dark:text-gray-400"
                      >
                        <ChevronUpDownIcon className="w-4 h-4" />
                      </motion.div>
                    </>
                  )}
                  <span
                    className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                      result.packetLoss > selectedMonitor.threshold
                        ? "bg-red-500/10 text-red-600 dark:text-red-400"
                        : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                    }`}
                  >
                    {result.packetLoss.toFixed(1)}% loss
                  </span>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">
                    Avg RTT:
                  </span>
                  <span className="text-blue-600 dark:text-blue-400 font-mono">
                    {formatRTT(result.avgRtt)}ms
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">
                    Min RTT:
                  </span>
                  <span className="text-emerald-600 dark:text-emerald-400 font-mono">
                    {formatRTT(result.minRtt)}ms
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">
                    Max RTT:
                  </span>
                  <span className="text-yellow-600 dark:text-yellow-400 font-mono">
                    {formatRTT(result.maxRtt)}ms
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600 dark:text-gray-400">
                    Packets:
                  </span>
                  <span className="text-gray-600 dark:text-gray-400">
                    {result.packetsRecv}/{result.packetsSent}
                  </span>
                </div>
              </div>

              {/* Expandable MTR Details */}
              {result.usedMtr && mtrData && (
                <MTRResultsDisplay
                  mtrData={mtrData}
                  isExpanded={isExpanded}
                  isMobile={true}
                />
              )}
            </motion.div>
          );
        })}

        {/* Load More Button for Mobile */}
        {historyList.length > displayCount && (
          <div className="flex justify-center mt-4">
            <button
              onClick={() => setDisplayCount((prev) => prev + 10)}
              className="px-4 py-2 bg-gray-200/30 dark:bg-gray-800/30 border border-gray-300/50 dark:border-gray-900/50 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800/50 transition-colors"
            >
              Load More
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
