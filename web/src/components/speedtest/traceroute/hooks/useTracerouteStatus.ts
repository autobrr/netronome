/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { TracerouteUpdate, TracerouteResult } from "@/types/types";
import { getTracerouteStatus } from "@/api/speedtest";
import { TRACEROUTE_POLLING } from "../constants/tracerouteConstants";

interface UseTracerouteStatusProps {
  isRunning: boolean;
  tracerouteStatus: TracerouteUpdate | null;
  results: TracerouteResult | null;
  onStatusUpdate: (status: TracerouteUpdate | null) => void;
}

export const useTracerouteStatus = ({
  isRunning,
  tracerouteStatus,
  results,
  onStatusUpdate,
}: UseTracerouteStatusProps) => {
  // Poll for traceroute status
  const { data: statusData } = useQuery({
    queryKey: ["traceroute", "status"],
    queryFn: getTracerouteStatus,
    refetchInterval:
      isRunning || (tracerouteStatus && !tracerouteStatus.isComplete)
        ? TRACEROUTE_POLLING.interval
        : false,
    enabled:
      isRunning || Boolean(tracerouteStatus && !tracerouteStatus.isComplete),
  });

  // Update status when we get new data
  useEffect(() => {
    if (statusData && !results) {
      onStatusUpdate(statusData);
    } else if (results) {
      // Clear status when we have final results
      onStatusUpdate(null);
    }
  }, [statusData, results, onStatusUpdate]);

  return {
    statusData,
    isPolling:
      isRunning || Boolean(tracerouteStatus && !tracerouteStatus.isComplete),
  };
};
