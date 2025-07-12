/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useCallback } from "react";

export interface CrossTabData {
  traceroute?: {
    host?: string;
    fromPacketLoss?: boolean;
  };
  packetloss?: {
    host?: string;
    name?: string;
    fromTraceroute?: boolean;
  };
}

export interface CrossTabNavigationHook {
  crossTabData: CrossTabData;
  navigateToTraceroute: (data?: {
    host?: string;
    fromPacketLoss?: boolean;
  }) => void;
  navigateToPacketLoss: (data?: {
    host?: string;
    name?: string;
    fromTraceroute?: boolean;
  }) => void;
  clearCrossTabData: () => void;
  consumeTracerouteData: () => CrossTabData["traceroute"] | undefined;
  consumePacketLossData: () => CrossTabData["packetloss"] | undefined;
}

// Create a custom hook for cross-tab navigation
export const useCrossTabNavigation = (
  activeTab: string,
  onTabChange: (tabId: string) => void,
): CrossTabNavigationHook => {
  const [crossTabData, setCrossTabData] = useState<CrossTabData>({});

  const navigateToTraceroute = useCallback(
    (data?: { host?: string; fromPacketLoss?: boolean }) => {
      setCrossTabData((prev) => ({
        ...prev,
        traceroute: data || {},
      }));
      onTabChange("traceroute");
    },
    [onTabChange],
  );

  const navigateToPacketLoss = useCallback(
    (data?: { host?: string; name?: string; fromTraceroute?: boolean }) => {
      setCrossTabData((prev) => ({
        ...prev,
        packetloss: data || {},
      }));
      onTabChange("packetloss");
    },
    [onTabChange],
  );

  const clearCrossTabData = useCallback(() => {
    setCrossTabData({});
  }, []);

  // Consume data for traceroute tab (clears the data after consumption)
  const consumeTracerouteData = useCallback(() => {
    const data = crossTabData.traceroute;
    if (data && activeTab === "traceroute") {
      setCrossTabData((prev) => ({
        ...prev,
        traceroute: undefined,
      }));
      return data;
    }
    return undefined;
  }, [crossTabData.traceroute, activeTab]);

  // Consume data for packet loss tab (clears the data after consumption)
  const consumePacketLossData = useCallback(() => {
    const data = crossTabData.packetloss;
    if (data && activeTab === "packetloss") {
      setCrossTabData((prev) => ({
        ...prev,
        packetloss: undefined,
      }));
      return data;
    }
    return undefined;
  }, [crossTabData.packetloss, activeTab]);

  return {
    crossTabData,
    navigateToTraceroute,
    navigateToPacketLoss,
    clearCrossTabData,
    consumeTracerouteData,
    consumePacketLossData,
  };
};
