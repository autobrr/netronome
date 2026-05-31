/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import { InfiniteData, useQueryClient } from "@tanstack/react-query";
import {
  PacketLossMonitor,
  PacketLossResult,
  PaginatedResponse,
} from "@/types/types";
import {
  getPacketLossHistory,
  getPacketLossMonitorStatus,
} from "@/api/packetloss";
import { PACKET_LOSS_HISTORY_PAGE_SIZE } from "../constants/packetLossConstants";
import { MonitorStatus } from "../types/monitorStatus";

const mergeLatestHistoryPage = (
  previous: InfiniteData<PaginatedResponse<PacketLossResult>>,
  latestPage: PaginatedResponse<PacketLossResult>,
): InfiniteData<PaginatedResponse<PacketLossResult>> => {
  const loadedCount = previous.pages.reduce(
    (total, page) => total + page.data.length,
    0,
  );
  const latestIDs = new Set(latestPage.data.map((result) => result.id));
  const remainingResults = previous.pages
    .flatMap((page) => page.data)
    .filter((result) => !latestIDs.has(result.id));
  const mergedResults = [...latestPage.data, ...remainingResults].slice(
    0,
    Math.min(loadedCount, latestPage.total || loadedCount),
  );

  return {
    ...previous,
    pages: previous.pages.map((page, index) => {
      const start = index * PACKET_LOSS_HISTORY_PAGE_SIZE;
      const end = start + PACKET_LOSS_HISTORY_PAGE_SIZE;

      return {
        ...page,
        data: mergedResults.slice(start, end),
        page: index + 1,
        limit: latestPage.limit,
        total: latestPage.total,
      };
    }),
  };
};

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

        const historyQueryKey = [
          "packetloss",
          "history",
          completedTest.monitorId,
        ] as const;

        // Refetch monitors list
        await queryClient.refetchQueries({
          queryKey: ["packetloss", "monitors"],
        });

        if (completedTest.monitorId === selectedMonitorId) {
          const cachedHistory = queryClient.getQueryData<
            InfiniteData<PaginatedResponse<PacketLossResult>>
          >(historyQueryKey);

          if (cachedHistory) {
            try {
              const latestHistory = await getPacketLossHistory(
                completedTest.monitorId,
                1,
                PACKET_LOSS_HISTORY_PAGE_SIZE,
              );

              queryClient.setQueryData<
                InfiniteData<PaginatedResponse<PacketLossResult>>
              >(historyQueryKey, (previous) =>
                previous
                  ? mergeLatestHistoryPage(previous, latestHistory)
                  : previous,
              );
            } catch (error) {
              console.error("Failed to refresh latest history page:", error);
              await queryClient.invalidateQueries({
                queryKey: historyQueryKey,
              });
            }
          } else {
            await queryClient.invalidateQueries({
              queryKey: historyQueryKey,
            });
          }
        } else {
          await queryClient.invalidateQueries({
            queryKey: historyQueryKey,
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
