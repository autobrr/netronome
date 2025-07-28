/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { TracerouteResult, TracerouteUpdate } from "@/types/types";
import { runTraceroute } from "@/api/speedtest";
import { extractHostname } from "../utils/tracerouteUtils";
import { DEFAULT_TRACEROUTE_CONFIG } from "../constants/tracerouteConstants";
import { showToast } from "@/components/common/Toast";

interface UseTracerouteExecutionProps {
  onStatusUpdate?: (status: TracerouteUpdate | null) => void;
  onError?: (error: string | null) => void;
}

export const useTracerouteExecution = ({
  onStatusUpdate,
  onError,
}: UseTracerouteExecutionProps = {}) => {
  const queryClient = useQueryClient();

  const tracerouteMutation = useMutation({
    mutationFn: runTraceroute,
    onMutate: (targetHost: string) => {
      // Clear previous results and error state
      queryClient.setQueryData(["traceroute", "results"], null);
      onError?.(null);
      showToast("Traceroute started", "success", {
        description: `Tracing route to ${targetHost}`,
      });

      // Set initial status
      const initialStatus: TracerouteUpdate = {
        type: "traceroute",
        host: targetHost,
        progress: DEFAULT_TRACEROUTE_CONFIG.initialProgress,
        isComplete: false,
        currentHop: 0,
        totalHops: DEFAULT_TRACEROUTE_CONFIG.totalHops,
        isScheduled: false,
        hops: [],
        destination: targetHost,
        ip: "",
      };

      onStatusUpdate?.(initialStatus);
    },
    onSuccess: (data: TracerouteResult) => {
      queryClient.setQueryData(["traceroute", "results"], data);
      onStatusUpdate?.(null);
      onError?.(null);
      showToast("Traceroute completed", "success", {
        description: `Route to ${data.destination} traced successfully (${data.hops.length} hops)`,
      });
    },
    onError: (error: Error) => {
      console.error("Traceroute failed:", error);
      onStatusUpdate?.(null);
      const errorMessage =
        error.message ||
        "Traceroute failed. Please check the hostname and try again.";
      onError?.(errorMessage);
      showToast("Traceroute failed", "error", {
        description: errorMessage,
      });
    },
  });

  const runTracerouteWithHostname = (
    host: string,
    selectedServerHost?: string
  ) => {
    let targetHost = selectedServerHost ? selectedServerHost : host.trim();
    if (!targetHost) return;

    // Extract hostname for all server types
    targetHost = extractHostname(targetHost);

    // Clear previous state before starting
    queryClient.setQueryData(["traceroute", "results"], null);
    onStatusUpdate?.(null);
    tracerouteMutation.mutate(targetHost);
  };

  return {
    runTraceroute: runTracerouteWithHostname,
    isRunning: tracerouteMutation.isPending,
    error: tracerouteMutation.error,
    data: tracerouteMutation.data,
    reset: tracerouteMutation.reset,
  };
};
