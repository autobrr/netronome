/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { TracerouteHop } from "@/types/types";
import { CountryFlag } from "@/components/speedtest/packetloss/components/CountryFlag";
import { formatRTT, getAverageRTT } from "../utils/tracerouteUtils";
import { TABLE_COLUMNS } from "../constants/tracerouteConstants";

interface TracerouteTableProps {
  hops: TracerouteHop[];
  showAnimation?: boolean;
  filterTrailingTimeouts?: boolean;
}

export const TracerouteTable: React.FC<TracerouteTableProps> = ({
  hops,
  showAnimation = true,
  filterTrailingTimeouts = false,
}) => {
  // Filter trailing timeouts if requested (for final results)
  const displayHops = filterTrailingTimeouts
    ? filterTrailingTimeoutsFromHops(hops)
    : hops;

  return (
    <div className="hidden md:block">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-800">
            <th className="text-center py-3 px-2 text-gray-400 font-medium">{TABLE_COLUMNS.hop}</th>
            <th className="text-center py-3 px-2 text-gray-400 font-medium">{TABLE_COLUMNS.host}</th>
            <th className="text-center py-3 px-2 text-gray-400 font-medium">
              {TABLE_COLUMNS.provider}
            </th>
            <th className="text-center py-3 px-2 text-gray-400 font-medium">{TABLE_COLUMNS.rtt1}</th>
            <th className="text-center py-3 px-2 text-gray-400 font-medium">{TABLE_COLUMNS.rtt2}</th>
            <th className="text-center py-3 px-2 text-gray-400 font-medium">{TABLE_COLUMNS.rtt3}</th>
            <th className="text-center py-3 px-2 text-gray-400 font-medium">{TABLE_COLUMNS.average}</th>
          </tr>
        </thead>
        <tbody>
          {displayHops.map((hop) => (
            <TracerouteTableRow
              key={hop.number}
              hop={hop}
              showAnimation={showAnimation}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
};

interface TracerouteTableRowProps {
  hop: TracerouteHop;
  showAnimation: boolean;
}

const TracerouteTableRow: React.FC<TracerouteTableRowProps> = ({
  hop,
  showAnimation,
}) => {
  const RowComponent = showAnimation ? motion.tr : "tr";
  const rowProps = showAnimation
    ? {
        initial: { opacity: 0, y: 20 },
        animate: { opacity: 1, y: 0 },
        transition: { duration: 0.3 },
      }
    : {};

  return (
    <RowComponent {...rowProps} className="border-b border-gray-300/50 dark:border-gray-800/50 last:border-0 hover:bg-gray-200/30 dark:hover:bg-gray-800/30 transition-colors">
      <td className="py-3 px-2 text-gray-700 dark:text-gray-300 text-center">{hop.number}</td>
      <td
        className="py-3 px-2 text-gray-700 dark:text-gray-300 text-center"
        title={hop.timeout ? "Request timed out" : hop.host}
      >
        {hop.timeout ? (
          <span className="text-gray-500 dark:text-gray-500">Timeout</span>
        ) : (
          hop.host
        )}
      </td>
      <td className="py-3 px-2 min-w-[200px] max-w-[200px] text-center">
        {hop.as ? (
          <div className="flex items-center justify-center gap-2">
            <CountryFlag
              countryCode={hop.countryCode}
              className="w-4 h-3 flex-shrink-0"
            />
            <span
              className="text-blue-600 dark:text-blue-400 text-xs truncate"
              title={hop.as}
            >
              {hop.as}
            </span>
          </div>
        ) : (
          <span className="text-gray-500 dark:text-gray-500">—</span>
        )}
      </td>
      <td className="py-3 px-2 text-center font-mono">
        {hop.timeout ? (
          <span className="text-gray-500">*</span>
        ) : (
          <span className="text-emerald-600 dark:text-emerald-400">
            {formatRTT(hop.rtt1)}
          </span>
        )}
      </td>
      <td className="py-3 px-2 text-center font-mono">
        {hop.timeout ? (
          <span className="text-gray-500">*</span>
        ) : (
          <span className="text-yellow-600 dark:text-yellow-400">
            {formatRTT(hop.rtt2)}
          </span>
        )}
      </td>
      <td className="py-3 px-2 text-center font-mono">
        {hop.timeout ? (
          <span className="text-gray-500">*</span>
        ) : (
          <span className="text-orange-600 dark:text-orange-400">
            {formatRTT(hop.rtt3)}
          </span>
        )}
      </td>
      <td className="py-3 px-2 text-center font-mono">
        {hop.timeout ? (
          <span className="text-gray-500">*</span>
        ) : (
          <span className="text-gray-700 dark:text-gray-300">
            {formatRTT(getAverageRTT(hop))}
          </span>
        )}
      </td>
    </RowComponent>
  );
};

/**
 * Filter trailing timeouts from hops array
 * This function should be imported from clipboard utils if available
 */
const filterTrailingTimeoutsFromHops = (
  hops: TracerouteHop[],
): TracerouteHop[] => {
  // Find the last non-timeout hop
  let lastValidIndex = hops.length - 1;
  while (lastValidIndex >= 0 && hops[lastValidIndex].timeout) {
    lastValidIndex--;
  }

  // Return hops up to and including the last valid hop
  return hops.slice(0, lastValidIndex + 1);
};
