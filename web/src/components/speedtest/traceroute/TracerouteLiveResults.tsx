/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { TracerouteUpdate } from "@/types/types";
import { TracerouteTable } from "./components/TracerouteTable";
import { TracerouteMobileCards } from "./components/TracerouteMobileCards";
import { STYLES } from "./constants/tracerouteConstants";

interface TracerouteLiveResultsProps {
  tracerouteStatus: TracerouteUpdate;
}

export const TracerouteLiveResults: React.FC<TracerouteLiveResultsProps> = ({
  tracerouteStatus,
}) => {
  // Don't show if complete or no hops yet
  if (tracerouteStatus.isComplete || !tracerouteStatus.hops?.length) {
    return null;
  }

  return (
    <div className={`mb-6 ${STYLES.card}`}>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
          Traceroute Results (In Progress)
        </h2>
      </div>

      <div className="mb-4">
        <div className="text-sm text-gray-600 dark:text-gray-400">
          <span className="font-medium">Destination:</span>{" "}
          {tracerouteStatus.host}
          {tracerouteStatus.ip && <span> ({tracerouteStatus.ip})</span>}
          <br />
          <span className="font-medium">Hops discovered:</span>{" "}
          {tracerouteStatus.hops.length}
          {tracerouteStatus.terminatedEarly && (
            <>
              <br />
              <span className="font-medium text-amber-600 dark:text-amber-400">
                Status:
              </span>{" "}
              <span className="text-amber-600 dark:text-amber-400">
                Early termination (reached destination or too many timeouts)
              </span>
            </>
          )}
        </div>
      </div>

      {/* Desktop Table View */}
      <TracerouteTable
        hops={tracerouteStatus.hops}
        showAnimation={true}
        filterTrailingTimeouts={false}
      />

      {/* Mobile Card View */}
      <TracerouteMobileCards
        hops={tracerouteStatus.hops}
        showAnimation={true}
        filterTrailingTimeouts={false}
      />
    </div>
  );
};
