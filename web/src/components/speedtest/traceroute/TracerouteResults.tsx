/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion } from "motion/react";
import { TracerouteResult } from "@/types/types";
import { SignalIcon, ClipboardDocumentIcon } from "@heroicons/react/24/outline";
import { TracerouteTable } from "./components/TracerouteTable";
import { TracerouteMobileCards } from "./components/TracerouteMobileCards";
import {
  formatTracerouteForClipboard,
  copyToClipboard,
} from "@/utils/clipboard";
import { COPY_SUCCESS_DURATION } from "./constants/tracerouteConstants";

interface TracerouteResultsProps {
  results: TracerouteResult;
  onCreateMonitor?: () => void;
}

export const TracerouteResults: React.FC<TracerouteResultsProps> = ({
  results,
  onCreateMonitor,
}) => {
  const [copySuccess, setCopySuccess] = useState(false);

  const copyTracerouteResults = async () => {
    const formattedOutput = formatTracerouteForClipboard(results);
    const success = await copyToClipboard(formattedOutput);

    if (success) {
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), COPY_SUCCESS_DURATION);
    }
  };

  return (
    <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
          Traceroute Results
        </h2>
        <div className="flex items-center gap-2">
          {onCreateMonitor && (
            <motion.button
              onClick={onCreateMonitor}
              className="flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-xs bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/30 hover:bg-purple-500/20"
              title="Create packet loss monitor for this host"
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
            >
              <SignalIcon className="w-3 h-3" />
              <span>Monitor This Host</span>
            </motion.button>
          )}
          <motion.button
            onClick={copyTracerouteResults}
            className={`flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-xs ${
              copySuccess
                ? "bg-emerald-600 text-white"
                : "bg-gray-300 dark:bg-gray-700 hover:bg-gray-400 dark:hover:bg-gray-600 text-gray-900 dark:text-white"
            }`}
            title="Copy traceroute results to clipboard"
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
          >
            <motion.div
              animate={{
                rotate: copySuccess ? 360 : 0,
                scale: copySuccess ? 1.1 : 1,
              }}
              transition={{ duration: 0.3 }}
            >
              <ClipboardDocumentIcon className="w-3 h-3" />
            </motion.div>
            <motion.span
              key={copySuccess ? "copied" : "copy"}
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 10 }}
              transition={{ duration: 0.2 }}
            >
              {copySuccess ? "Copied!" : "Copy"}
            </motion.span>
          </motion.button>
        </div>
      </div>

      <p className="text-gray-600 dark:text-gray-400 text-sm mb-6">
        Route to {results.destination}
        {results.ip && results.ip !== results.destination && (
          <span className="text-gray-500 dark:text-gray-500 ml-1">
            ({results.ip})
          </span>
        )}
        {" • "}
        {results.totalHops} hops •{" "}
        {results.complete ? "Complete" : "Incomplete"}
      </p>

      {/* Desktop Table View */}
      <TracerouteTable
        hops={results.hops}
        showAnimation={false}
        filterTrailingTimeouts={true}
      />

      {/* Mobile Card View */}
      <TracerouteMobileCards
        hops={results.hops}
        showAnimation={false}
        filterTrailingTimeouts={true}
      />
    </div>
  );
};
