/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion, AnimatePresence } from "motion/react";
import { MTRData } from "@/types/types";
import { CountryFlag } from "./CountryFlag";

interface MTRResultsDisplayProps {
  mtrData: MTRData;
  isExpanded: boolean;
  isMobile?: boolean;
}

export const MTRResultsDisplay: React.FC<MTRResultsDisplayProps> = ({
  mtrData,
  isExpanded,
  isMobile = false,
}) => {
  if (isMobile) {
    return (
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="mt-4 overflow-hidden"
          >
            <div className="border-t border-gray-300 dark:border-gray-700 pt-3">
              <div className="text-sm text-gray-700 dark:text-gray-300 mb-2">
                <strong>MTR Route Analysis</strong> • {mtrData.hopCount} hops
              </div>
              <div className="space-y-2">
                {mtrData.hops.map((hop) => (
                  <div
                    key={hop.number}
                    className="flex justify-between items-center text-xs p-2 bg-gray-100 dark:bg-gray-900/50 rounded"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-gray-500 dark:text-gray-400">
                          {hop.number}.
                        </span>
                        <span className="text-gray-700 dark:text-gray-300">
                          {hop.host || "* * *"}
                        </span>
                        {hop.as && (
                          <div className="flex items-center gap-1">
                            <CountryFlag
                              countryCode={hop.countryCode}
                              className="w-3 h-2.5 flex-shrink-0"
                            />
                            <span className="text-xs text-gray-500 dark:text-gray-500 truncate">
                              {hop.as}
                            </span>
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="flex gap-3">
                      <span
                        className={`font-mono ${
                          hop.loss > 0
                            ? "text-red-600 dark:text-red-400"
                            : "text-green-600 dark:text-green-400"
                        }`}
                      >
                        {hop.loss.toFixed(1)}%
                      </span>
                      <span className="font-mono text-blue-600 dark:text-blue-400">
                        {hop.avg.toFixed(1)}ms
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    );
  }

  // Desktop table view
  return (
    <AnimatePresence>
      {isExpanded && (
        <motion.tr
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.3 }}
          className="bg-gray-100/50 dark:bg-gray-900/50"
        >
          <td colSpan={8} className="p-4">
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                MTR Hop-by-Hop Analysis
              </h4>
              <div className="overflow-x-auto">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b border-gray-300 dark:border-gray-700">
                      <th className="text-left py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        Hop
                      </th>
                      <th className="text-left py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        Host
                      </th>
                      <th className="text-left py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        ASN
                      </th>
                      <th className="text-center py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        Loss %
                      </th>
                      <th className="text-center py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        Avg
                      </th>
                      <th className="text-center py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        Min
                      </th>
                      <th className="text-center py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        Max
                      </th>
                      <th className="text-center py-2 px-2 text-gray-600 dark:text-gray-400 font-medium">
                        StdDev
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {mtrData.hops.map((hop) => (
                      <tr
                        key={hop.number}
                        className="border-b border-gray-200 dark:border-gray-800 text-gray-600 dark:text-gray-400"
                      >
                        <td className="py-2 px-2">{hop.number}</td>
                        <td
                          className="py-2 px-2 truncate max-w-xs"
                          title={hop.host}
                        >
                          {hop.host || "* * *"}
                        </td>
                        <td className="py-2 px-2">
                          {hop.as ? (
                            <div className="flex items-center gap-2">
                              <CountryFlag
                                countryCode={hop.countryCode}
                                className="w-4 h-3 flex-shrink-0"
                              />
                              <span className="text-xs text-gray-600 dark:text-gray-400 truncate">
                                {hop.as}
                              </span>
                            </div>
                          ) : (
                            <span className="text-gray-400 dark:text-gray-600">
                              -
                            </span>
                          )}
                        </td>
                        <td
                          className={`py-2 px-2 text-center font-mono ${
                            hop.loss > 0
                              ? "text-red-600 dark:text-red-400"
                              : "text-green-600 dark:text-green-400"
                          }`}
                        >
                          {hop.loss.toFixed(1)}%
                        </td>
                        <td className="py-2 px-2 text-center font-mono text-blue-600 dark:text-blue-400">
                          {hop.avg.toFixed(1)}
                        </td>
                        <td className="py-2 px-2 text-center font-mono text-green-600 dark:text-green-400">
                          {hop.best.toFixed(1)}
                        </td>
                        <td className="py-2 px-2 text-center font-mono text-yellow-600 dark:text-yellow-400">
                          {hop.worst.toFixed(1)}
                        </td>
                        <td className="py-2 px-2 text-center font-mono text-gray-600 dark:text-gray-400">
                          {hop.stddev.toFixed(1)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </td>
        </motion.tr>
      )}
    </AnimatePresence>
  );
};
