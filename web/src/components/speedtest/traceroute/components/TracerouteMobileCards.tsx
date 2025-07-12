/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { TracerouteHop } from "@/types/types";
import { CountryFlag } from "@/components/speedtest/packetloss/components/CountryFlag";
import { formatRTT, getAverageRTT } from "../utils/tracerouteUtils";
import { STYLES } from "../constants/tracerouteConstants";

interface TracerouteMobileCardsProps {
  hops: TracerouteHop[];
  showAnimation?: boolean;
  filterTrailingTimeouts?: boolean;
}

export const TracerouteMobileCards: React.FC<TracerouteMobileCardsProps> = ({
  hops,
  showAnimation = true,
  filterTrailingTimeouts = false,
}) => {
  // Filter trailing timeouts if requested (for final results)
  const displayHops = filterTrailingTimeouts
    ? filterTrailingTimeoutsFromHops(hops)
    : hops;

  return (
    <div className="md:hidden space-y-3">
      {displayHops.map((hop) => (
        <TracerouteMobileCard
          key={hop.number}
          hop={hop}
          showAnimation={showAnimation}
        />
      ))}
    </div>
  );
};

interface TracerouteMobileCardProps {
  hop: TracerouteHop;
  showAnimation: boolean;
}

const TracerouteMobileCard: React.FC<TracerouteMobileCardProps> = ({
  hop,
  showAnimation,
}) => {
  const CardComponent = showAnimation ? motion.div : "div";
  const cardProps = showAnimation
    ? {
        initial: { opacity: 0, y: 20 },
        animate: { opacity: 1, y: 0 },
        transition: { duration: 0.3 },
      }
    : {};

  return (
    <CardComponent {...cardProps} className={STYLES.mobileCard}>
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="font-mono text-blue-600 dark:text-blue-400 font-semibold">
            Hop {hop.number}
          </span>
          {hop.countryCode && (
            <div className="flex items-center gap-1">
              <CountryFlag countryCode={hop.countryCode} className="w-4 h-3" />
              <span className="text-gray-600 dark:text-gray-400 text-xs">
                {hop.location}
              </span>
            </div>
          )}
        </div>
      </div>

      <div className="text-sm text-gray-900 dark:text-gray-100 font-mono mb-2 break-all">
        {hop.timeout ? "Timeout" : hop.host}
      </div>

      <div className="flex flex-wrap gap-2 text-xs">
        <span className="text-gray-600 dark:text-gray-400">
          RTT:{" "}
          <span
            className={`font-mono ${
              hop.timeout
                ? "text-gray-500"
                : "text-emerald-600 dark:text-emerald-400"
            }`}
          >
            {formatRTT(hop.rtt1)}
          </span>{" "}
          <span
            className={`font-mono ${
              hop.timeout
                ? "text-gray-500"
                : "text-yellow-600 dark:text-yellow-400"
            }`}
          >
            {formatRTT(hop.rtt2)}
          </span>{" "}
          <span
            className={`font-mono ${
              hop.timeout
                ? "text-gray-500"
                : "text-orange-600 dark:text-orange-400"
            }`}
          >
            {formatRTT(hop.rtt3)}
          </span>
        </span>

        <span className="text-gray-600 dark:text-gray-400">
          Avg:{" "}
          <span
            className={`font-mono font-medium ${
              hop.timeout ? "text-gray-500" : "text-gray-900 dark:text-gray-100"
            }`}
          >
            {hop.timeout ? "*" : formatRTT(getAverageRTT(hop))}
          </span>
        </span>

        {hop.as && (
          <span className="text-gray-600 dark:text-gray-400">
            ASN:{" "}
            <span className="font-mono text-gray-900 dark:text-gray-100">
              {hop.as}
            </span>
          </span>
        )}
      </div>
    </CardComponent>
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
