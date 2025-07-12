/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { PacketLossMonitor } from "@/types/types";
import {
  getPacketLossMonitorStatus,
  getPacketLossHistory,
} from "@/api/packetloss";
import { MonitorStatus } from "../types/monitorStatus";

export const usePacketLossMonitorStatus = (
  monitors: PacketLossMonitor[],
  selectedMonitorId?: number,
) => {
  const queryClient = useQueryClient();
  const [monitorStatuses, setMonitorStatuses] = useState<
    Map<number, MonitorStatus>
  >(new Map());

  useEffect(() => {
    const enabledMonitors = monitors.filter((m) => m.enabled);

    if (enabledMonitors.length === 0) {
      return;
    }

    const pollInterval = setInterval(async () => {
      const statusPromises = enabledMonitors.map(async (monitor) => {
        try {
          const status = await getPacketLossMonitorStatus(monitor.id);
          return { monitorId: monitor.id, status };
        } catch (error) {
          console.error(
            `Failed to get status for monitor ${monitor.id}:`,
            error,
          );
          return null;
        }
      });

      const results = await Promise.all(statusPromises);

      // Check for completed tests - backend now properly sets isComplete
      const completedTests = results.filter(
        (result) => result && result.status.isComplete,
      );

      // Update state first
      setMonitorStatuses((prev) => {
        const newStatuses = new Map(prev);
        results.forEach((result) => {
          if (result) {
            newStatuses.set(result.monitorId, result.status);
          }
        });
        return newStatuses;
      });

      // Then handle refetches for completed tests
      for (const completedTest of completedTests) {
        if (!completedTest) continue;

        // Refetch monitors list
        await queryClient.refetchQueries({
          queryKey: ["packetloss", "monitors"],
        });

        // If this is the selected monitor, fetch fresh data and update cache directly
        if (completedTest.monitorId === selectedMonitorId) {
          try {
            // Fetch fresh history from the backend
            const freshHistory = await getPacketLossHistory(
              completedTest.monitorId,
              100,
            );

            // Directly update the cache with fresh data (like TracerouteTab does)
            queryClient.setQueryData(
              ["packetloss", "history", completedTest.monitorId],
              freshHistory,
            );

            // Force React Query to notify all subscribers of the change
            queryClient.invalidateQueries({
              queryKey: ["packetloss", "history", completedTest.monitorId],
              exact: true,
            });
          } catch (error) {
            console.error("Failed to fetch updated history:", error);
          }
        } else {
          // For other monitors, just invalidate to mark as stale
          await queryClient.invalidateQueries({
            queryKey: ["packetloss", "history", completedTest.monitorId],
          });
        }

        // Clear completion status after delay
        setTimeout(() => {
          setMonitorStatuses((prev) => {
            const newMap = new Map(prev);
            const currentStatus = newMap.get(completedTest.monitorId);
            if (currentStatus && currentStatus.isComplete) {
              newMap.delete(completedTest.monitorId);
            }
            return newMap;
          });
        }, 3000);
      }
    }, 2000); // Poll every 2 seconds

    return () => clearInterval(pollInterval);
  }, [monitors, queryClient, selectedMonitorId]);

  return monitorStatuses;
};
