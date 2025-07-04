/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { Schedule, Server, SavedIperfServer } from "@/types/types";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getSchedules } from "@/api/speedtest";
import {
  DisclosureButton,
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
  Transition,
} from "@headlessui/react";
import { ChevronUpDownIcon } from "@heroicons/react/20/solid";
import { Disclosure } from "@headlessui/react";
import {
  ChevronDownIcon,
  XMarkIcon,
  ClockIcon,
  ArrowPathIcon,
} from "@heroicons/react/20/solid";
import { motion, AnimatePresence } from "motion/react";
import { getApiUrl } from "@/utils/baseUrl";
import { formatNextRun } from "@/utils/timeUtils";

interface ScheduleManagerProps {
  servers: Server[];
  selectedServers: Server[];
  onServerSelect: (server: Server) => void;
}

interface IntervalOption {
  value: string;
  label: string;
}

interface TimeOption {
  value: string;
  label: string;
}

const intervalOptions: IntervalOption[] = [
  { value: "5m", label: "Every 5 Minutes" },
  { value: "15m", label: "Every 15 Minutes" },
  { value: "30m", label: "Every 30 Minutes" },
  { value: "1h", label: "Every Hour" },
  { value: "6h", label: "Every 6 Hours" },
  { value: "12h", label: "Every 12 Hours" },
  { value: "24h", label: "Every Day" },
  { value: "7d", label: "Every Week" },
];

const timeOptions: TimeOption[] = [
  { value: "00:00", label: "12:00 AM" },
  { value: "01:00", label: "1:00 AM" },
  { value: "02:00", label: "2:00 AM" },
  { value: "03:00", label: "3:00 AM" },
  { value: "04:00", label: "4:00 AM" },
  { value: "05:00", label: "5:00 AM" },
  { value: "06:00", label: "6:00 AM" },
  { value: "07:00", label: "7:00 AM" },
  { value: "08:00", label: "8:00 AM" },
  { value: "09:00", label: "9:00 AM" },
  { value: "10:00", label: "10:00 AM" },
  { value: "11:00", label: "11:00 AM" },
  { value: "12:00", label: "12:00 PM" },
  { value: "13:00", label: "1:00 PM" },
  { value: "14:00", label: "2:00 PM" },
  { value: "15:00", label: "3:00 PM" },
  { value: "16:00", label: "4:00 PM" },
  { value: "17:00", label: "5:00 PM" },
  { value: "18:00", label: "6:00 PM" },
  { value: "19:00", label: "7:00 PM" },
  { value: "20:00", label: "8:00 PM" },
  { value: "21:00", label: "9:00 PM" },
  { value: "22:00", label: "10:00 PM" },
  { value: "23:00", label: "11:00 PM" },
];

const parseInterval = (intervalStr: string): number => {
  const value = parseInt(intervalStr);
  const unit = intervalStr.slice(-1);

  switch (unit) {
    case "m":
      return value * 60 * 1000;
    case "h":
      return value * 60 * 60 * 1000;
    case "d":
      return value * 24 * 60 * 60 * 1000;
    default:
      return value * 60 * 1000;
  }
};

const calculateNextRun = (
  intervalStr: string,
  scheduleType: "interval" | "exact",
  exactTime?: string
): string => {
  if (scheduleType === "exact" && exactTime) {
    const now = new Date();
    const times = exactTime.split(",");

    let closestTime: Date | null = null;
    let minDiff = Infinity;

    // Check each time to find the next upcoming one
    for (const timeStr of times) {
      const [hours, minutes] = timeStr.trim().split(":").map(Number);

      // Try today
      const todayRun = new Date(now);
      todayRun.setHours(hours, minutes, 0, 0);

      if (todayRun > now) {
        const diff = todayRun.getTime() - now.getTime();
        if (diff < minDiff) {
          minDiff = diff;
          closestTime = todayRun;
        }
      }

      // Try tomorrow
      const tomorrowRun = new Date(todayRun);
      tomorrowRun.setDate(tomorrowRun.getDate() + 1);
      const tomorrowDiff = tomorrowRun.getTime() - now.getTime();
      if (tomorrowDiff < minDiff) {
        minDiff = tomorrowDiff;
        closestTime = tomorrowRun;
      }
    }

    return closestTime ? closestTime.toISOString() : new Date().toISOString();
  } else {
    const milliseconds = parseInterval(intervalStr);
    return new Date(Date.now() + milliseconds).toISOString();
  }
};

export default function ScheduleManager({
  servers,
  selectedServers,
}: ScheduleManagerProps) {
  const queryClient = useQueryClient();
  const [iperfServers, setIperfServers] = useState<SavedIperfServer[]>([]);
  const [interval, setInterval] = useState<string>("1h");
  const [scheduleType, setScheduleType] = useState<"interval" | "exact">(
    "interval"
  );
  const [exactTimes, setExactTimes] = useState<string[]>(["09:00"]);
  const [enabled] = useState(true);
  const [, setError] = useState<string | null>(null);
  const [updateTrigger, setUpdateTrigger] = useState(0);
  const [isOpen] = useState(() => {
    const saved = localStorage.getItem("schedule-manager-open");
    return saved === null ? true : saved === "true";
  });

  // Use TanStack Query for schedules with automatic refetching
  const {
    data: schedules = [],
    isLoading: isSchedulesLoading,
    error: schedulesError,
  } = useQuery({
    queryKey: ["schedules"],
    queryFn: () => {
      console.log("[ScheduleManager] Fetching schedules...");
      return getSchedules();
    },
    refetchInterval: 30000, // Refetch every 30 seconds
    staleTime: 10000, // Consider data stale after 10 seconds
  }) as { data: Schedule[]; isLoading: boolean; error: any };

  // Log when schedules data changes
  useEffect(() => {
    console.log("[ScheduleManager] Schedules data updated:", schedules);
    if (schedulesError) {
      console.error("[ScheduleManager] Schedules error:", schedulesError);
    }
  }, [schedules, schedulesError]);

  // Update "Next run in:" times every minute (synchronized)
  useEffect(() => {
    // Calculate delay to sync with minute boundary
    const now = new Date();
    const secondsUntilNextMinute = 60 - now.getSeconds();
    const initialDelay = secondsUntilNextMinute * 1000;

    // Start timer at the next minute boundary
    const initialTimer = window.setTimeout(() => {
      console.log("[ScheduleManager] Updating next run times... (synced)");
      setUpdateTrigger((prev) => prev + 1);

      // Set up regular interval after initial sync
      const timer = window.setInterval(() => {
        console.log("[ScheduleManager] Updating next run times... (synced)");
        setUpdateTrigger((prev) => prev + 1);
      }, 60000); // Update every minute

      // Store timer ID for cleanup
      (window as any)._scheduleManagerTimer = timer;
    }, initialDelay);

    return () => {
      window.clearTimeout(initialTimer);
      if ((window as any)._scheduleManagerTimer) {
        window.clearInterval((window as any)._scheduleManagerTimer);
        delete (window as any)._scheduleManagerTimer;
      }
    };
  }, []);

  useEffect(() => {
    fetchIperfServers();
  }, []);

  const fetchIperfServers = async () => {
    try {
      const response = await fetch("/api/iperf/servers");
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.message || `HTTP error! status: ${response.status}`
        );
      }
      const data = await response.json();
      setIperfServers(data || []);
    } catch (error) {
      console.error("Failed to fetch iperf servers:", error);
    }
  };

  const handleCreateSchedule = async () => {
    if (selectedServers.length === 0) {
      setError("Please select at least one server");
      return;
    }

    setError(null);

    const isIperfServer = selectedServers[0].isIperf;
    const isLibrespeedServer = selectedServers[0].isLibrespeed;

    const newSchedule: Schedule = {
      serverIds: selectedServers.map((s) => s.id),
      interval:
        scheduleType === "exact" ? `exact:${exactTimes.join(",")}` : interval,
      nextRun: calculateNextRun(interval, scheduleType, exactTimes.join(",")),
      enabled,
      options: {
        enableDownload: true,
        enableUpload: true,
        enablePacketLoss: true,
        serverIds: selectedServers.map((s) => s.id),
        useIperf: isIperfServer,
        useLibrespeed: isLibrespeedServer,
        serverHost: isIperfServer ? selectedServers[0].host : undefined,
      },
    };

    try {
      const response = await fetch(getApiUrl("/schedules"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(newSchedule),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.message || `HTTP error! status: ${response.status}`
        );
      }

      await response.json();
      // Invalidate and refetch schedules
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to create schedule"
      );
      // Invalidate and refetch schedules on error
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    }
  };

  const handleDeleteSchedule = async (id: number) => {
    // Optimistically update UI by invalidating the query
    queryClient.setQueryData(["schedules"], (old: Schedule[] | undefined) =>
      old ? old.filter((schedule) => schedule.id !== id) : []
    );

    try {
      const response = await fetch(getApiUrl(`/schedules/${id}`), {
        method: "DELETE",
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.message || `HTTP error! status: ${response.status}`
        );
      }
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to delete schedule"
      );
      // Invalidate and refetch schedules on error
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    }
  };

  const getServerNames = (serverIds: string[] | undefined) => {
    const serversList = (serverIds || [])
      .map((id: string) => {
        if (id.startsWith("iperf3-")) {
          const host = id.substring(7);
          const iperfServer = iperfServers.find(
            (s) =>
              s.host === host.split(":")[0] &&
              s.port === parseInt(host.split(":")[1])
          );
          return (
            <span
              key={id}
              className="inline-block group relative cursor-pointer"
            >
              <span>
                {iperfServer?.name || host} -{" "}
                <span className="text-purple-600 dark:text-purple-400 drop-shadow-[0_0_1px_rgba(168,85,247,0.8)]">
                  iperf3
                </span>
              </span>
              {iperfServer?.name && (
                <span
                  className="
                    absolute top-full left-1/2 transform -translate-x-1/2 mt-2
                    px-3 py-2 text-sm
                    text-gray-900 dark:text-gray-200 bg-gray-100/95 dark:bg-gray-800/95
                    rounded-lg shadow-lg
                    border border-gray-300/50 dark:border-gray-700/50
                    backdrop-blur-sm
                    opacity-0 scale-95 invisible 
                    group-hover:opacity-100 group-hover:scale-100 group-hover:visible
                    transition-all duration-200 ease-out
                    whitespace-nowrap
                    z-50
                    before:content-['']
                    before:absolute before:-top-1
                    before:left-1/2 before:-translate-x-1/2
                    before:w-2 before:h-2
                    before:rotate-45
                    before:bg-gray-100/95 dark:before:bg-gray-800/95
                    before:border-t before:border-l
                    before:border-gray-300/50 dark:before:border-gray-700/50
                  "
                >
                  {host}
                </span>
              )}
            </span>
          );
        }

        const server = servers.find((s: Server) => s.id === id);
        if (server) {
          if (server.isLibrespeed) {
            return (
              <span key={id}>
                {server.name} -{" "}
                <span className="text-blue-600 dark:text-blue-400 drop-shadow-[0_0_1px_rgba(96,165,250,0.8)]">
                  librespeed
                </span>
              </span>
            );
          }
          return (
            <span key={id}>
              {server.sponsor} - {server.name} -{" "}
              <span className="text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_1px_rgba(251,191,36,0.8)]">
                speedtest.net
              </span>
            </span>
          );
        }
        return null;
      })
      .filter(Boolean);

    if (serversList.length === 1) {
      return serversList[0];
    } else if (serversList.length > 1) {
      return `${serversList.length} servers`;
    }
    return "No servers";
  };

  if (isSchedulesLoading) {
    return (
      <div className="flex justify-center p-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    );
  }

  return (
    <div className="h-full">
      <Disclosure defaultOpen={isOpen}>
        {({ open }) => {
          useEffect(() => {
            localStorage.setItem("schedule-manager-open", open.toString());
          }, [open]);

          return (
            <div className="flex flex-col h-full">
              <DisclosureButton
                className={`flex justify-between items-center w-full px-4 py-2 bg-gray-50/95 dark:bg-gray-850/95 ${
                  open ? "rounded-t-xl" : "rounded-xl"
                } shadow-lg border-b-0 border-gray-200 dark:border-gray-900 text-left`}
              >
                <div className="flex flex-col">
                  <h2 className="text-gray-900 dark:text-white text-xl font-semibold p-1 select-none">
                    Schedule Manager
                  </h2>
                  <p className="text-gray-600 dark:text-gray-400 text-sm pl-1 pb-1">
                    Create and manage your schedules
                  </p>
                </div>
                <ChevronDownIcon
                  className={`${
                    open ? "transform rotate-180" : ""
                  } w-5 h-5 text-gray-600 dark:text-gray-400 transition-transform duration-200`}
                />
              </DisclosureButton>

              {open && (
                <div className="bg-gray-50/95 dark:bg-gray-850/95 px-4 pt-3 rounded-b-xl shadow-lg flex-1">
                  <div className="flex flex-col pl-1">
                    <div className="flex flex-col gap-4 pb-4">
                      <div className="grid grid-cols-1 gap-4">
                        <div>
                          {/* Schedule Type Toggle Buttons */}
                          <div className="mb-4">
                            <div className="grid grid-cols-2 gap-2 p-1 bg-gray-200/50 dark:bg-gray-800/30 rounded-lg">
                              <button
                                onClick={() => setScheduleType("interval")}
                                className={`flex items-center justify-center gap-2 px-4 py-2.5 rounded-md font-medium transition-all duration-200 ${
                                  scheduleType === "interval"
                                    ? "bg-gray-300 dark:bg-gray-700 text-gray-900 dark:text-gray-200 shadow-lg transform scale-105"
                                    : "text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 hover:bg-gray-300/50 dark:hover:bg-gray-800/50"
                                }`}
                              >
                                <ArrowPathIcon className="w-4 h-4" />
                                <span>Interval</span>
                              </button>
                              <button
                                onClick={() => setScheduleType("exact")}
                                className={`flex items-center justify-center gap-2 px-4 py-2.5 rounded-md font-medium transition-all duration-200 ${
                                  scheduleType === "exact"
                                    ? "bg-gray-300 dark:bg-gray-700 text-gray-900 dark:text-gray-200 shadow-lg transform scale-105"
                                    : "text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-300 hover:bg-gray-300/50 dark:hover:bg-gray-800/50"
                                }`}
                              >
                                <ClockIcon className="w-4 h-4" />
                                <span>Exact Time</span>
                              </button>
                            </div>
                          </div>

                          {/* Interval or Time Selector */}
                          {scheduleType === "interval" ? (
                            <Listbox value={interval} onChange={setInterval}>
                              <div className="relative">
                                <ListboxButton className="relative w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-lg text-left text-gray-700 dark:text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                                  <span className="block truncate">
                                    {
                                      intervalOptions.find(
                                        (opt) => opt.value === interval
                                      )?.label
                                    }
                                  </span>
                                  <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                                    <ChevronUpDownIcon
                                      className="h-5 w-5 text-gray-600 dark:text-gray-400"
                                      aria-hidden="true"
                                    />
                                  </span>
                                </ListboxButton>
                                <Transition
                                  enter="transition duration-100 ease-out"
                                  enterFrom="transform scale-95 opacity-0"
                                  enterTo="transform scale-100 opacity-100"
                                  leave="transition duration-75 ease-out"
                                  leaveFrom="transform scale-100 opacity-100"
                                  leaveTo="transform scale-95 opacity-0"
                                >
                                  <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-900 py-1 shadow-lg focus:outline-none">
                                    {intervalOptions.map((option) => (
                                      <ListboxOption
                                        key={option.value}
                                        value={option.value}
                                        className={({ focus }) =>
                                          `relative cursor-pointer select-none py-2 px-4 ${
                                            focus
                                              ? "bg-blue-500/10 text-blue-600 dark:text-blue-200"
                                              : "text-gray-700 dark:text-gray-300"
                                          }`
                                        }
                                      >
                                        {option.label}
                                      </ListboxOption>
                                    ))}
                                  </ListboxOptions>
                                </Transition>
                              </div>
                            </Listbox>
                          ) : (
                            <div className="space-y-3">
                              {/* Selected Times Display */}
                              {exactTimes.length > 0 && (
                                <div className="flex flex-wrap gap-2 p-3 bg-gray-200/50 dark:bg-gray-800/30 rounded-lg border border-gray-300 dark:border-gray-900">
                                  {exactTimes.sort().map((time) => (
                                    <div
                                      key={time}
                                      className="flex items-center gap-1 px-3 py-1.5 bg-blue-500/20 text-blue-600 dark:text-blue-400 rounded-md border border-blue-500/30"
                                    >
                                      <ClockIcon className="w-3.5 h-3.5" />
                                      <span className="text-sm font-medium">
                                        {timeOptions.find(
                                          (opt) => opt.value === time
                                        )?.label || time}
                                      </span>
                                      <button
                                        onClick={() =>
                                          setExactTimes(
                                            exactTimes.filter((t) => t !== time)
                                          )
                                        }
                                        className="ml-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
                                      >
                                        <XMarkIcon className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  ))}
                                </div>
                              )}

                              {/* Time Picker */}
                              <Listbox
                                value=""
                                onChange={(newTime: string) => {
                                  if (
                                    newTime &&
                                    !exactTimes.includes(newTime)
                                  ) {
                                    setExactTimes([...exactTimes, newTime]);
                                  }
                                }}
                              >
                                <div className="relative">
                                  <ListboxButton className="relative w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-lg text-left text-gray-700 dark:text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                                    <span className="block truncate">
                                      {exactTimes.length === 0
                                        ? "Select times..."
                                        : "Add another time..."}
                                    </span>
                                    <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                                      <ChevronUpDownIcon
                                        className="h-5 w-5 text-gray-400"
                                        aria-hidden="true"
                                      />
                                    </span>
                                  </ListboxButton>
                                  <Transition
                                    enter="transition duration-100 ease-out"
                                    enterFrom="transform scale-95 opacity-0"
                                    enterTo="transform scale-100 opacity-100"
                                    leave="transition duration-75 ease-out"
                                    leaveFrom="transform scale-100 opacity-100"
                                    leaveTo="transform scale-95 opacity-0"
                                  >
                                    <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-900 py-1 shadow-lg focus:outline-none">
                                      {timeOptions.map((option) => (
                                        <ListboxOption
                                          key={option.value}
                                          value={option.value}
                                          disabled={exactTimes.includes(
                                            option.value
                                          )}
                                          className={({ focus, disabled }) =>
                                            `relative cursor-pointer select-none py-2 px-4 flex items-center justify-between ${
                                              disabled
                                                ? "opacity-50 cursor-not-allowed text-gray-500 dark:text-gray-500"
                                                : focus
                                                  ? "bg-blue-500/10 text-blue-600 dark:text-blue-200"
                                                  : "text-gray-700 dark:text-gray-300"
                                            }`
                                          }
                                        >
                                          <span>{option.label}</span>
                                          {exactTimes.includes(
                                            option.value
                                          ) && (
                                            <span className="text-xs text-gray-500 dark:text-gray-500">
                                              Added
                                            </span>
                                          )}
                                        </ListboxOption>
                                      ))}
                                    </ListboxOptions>
                                  </Transition>
                                </div>
                              </Listbox>
                            </div>
                          )}

                          {/* Next Run Preview */}
                          {selectedServers.length > 0 &&
                            (scheduleType === "interval" ||
                              exactTimes.length > 0) && (
                              <div className="mt-4 p-3 bg-gray-200/50 dark:bg-gray-800/30 rounded-lg border border-gray-300 dark:border-gray-900">
                                <p className="text-sm text-gray-600 dark:text-gray-400">
                                  <span className="font-medium">Next run:</span>{" "}
                                  <span className="text-blue-600 dark:text-blue-400">
                                    {(() => {
                                      // Force re-calculation when updateTrigger changes
                                      updateTrigger; // This ensures the component re-renders
                                      const nextRun = new Date(
                                        calculateNextRun(
                                          interval,
                                          scheduleType,
                                          exactTimes.join(",")
                                        )
                                      );
                                      const now = new Date();
                                      const diffMs =
                                        nextRun.getTime() - now.getTime();
                                      const diffMins = Math.round(
                                        diffMs / 60000
                                      );

                                      if (diffMins < 60) {
                                        return `in ${diffMins} minute${
                                          diffMins !== 1 ? "s" : ""
                                        }`;
                                      } else if (diffMins < 1440) {
                                        const hours = Math.floor(diffMins / 60);
                                        return `in ${hours} hour${
                                          hours !== 1 ? "s" : ""
                                        }`;
                                      } else {
                                        const days = Math.floor(
                                          diffMins / 1440
                                        );
                                        return `in ${days} day${
                                          days !== 1 ? "s" : ""
                                        }`;
                                      }
                                    })()}
                                  </span>
                                  {scheduleType === "exact" && (
                                    <span className="text-gray-500 dark:text-gray-500 text-xs ml-2">
                                      (
                                      {new Date(
                                        calculateNextRun(
                                          interval,
                                          scheduleType,
                                          exactTimes.join(",")
                                        )
                                      ).toLocaleDateString()}
                                      )
                                    </span>
                                  )}
                                </p>
                              </div>
                            )}

                          {/* Create Schedule Button */}
                          <div className="mt-6">
                            <button
                              className={`w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg font-medium transition-all duration-200 ${
                                selectedServers.length === 0 ||
                                (scheduleType === "exact" &&
                                  exactTimes.length === 0)
                                  ? "bg-gray-300/50 dark:bg-gray-800/50 text-gray-500 dark:text-gray-500 cursor-not-allowed border border-gray-400 dark:border-gray-900"
                                  : "bg-blue-500 hover:bg-blue-600 text-white shadow-lg border border-blue-600 hover:border-blue-700 hover:shadow-xl"
                              }`}
                              onClick={handleCreateSchedule}
                              disabled={
                                selectedServers.length === 0 ||
                                (scheduleType === "exact" &&
                                  exactTimes.length === 0)
                              }
                            >
                              {selectedServers.length === 0 ? (
                                <>Select a server to create schedule</>
                              ) : scheduleType === "exact" &&
                                exactTimes.length === 0 ? (
                                <>Select at least one time</>
                              ) : (
                                <>
                                  {scheduleType === "interval" ? (
                                    <ArrowPathIcon className="w-5 h-5" />
                                  ) : (
                                    <ClockIcon className="w-5 h-5" />
                                  )}
                                  <span>
                                    Create{" "}
                                    {scheduleType === "interval"
                                      ? intervalOptions.find(
                                          (opt) => opt.value === interval
                                        )?.label
                                      : exactTimes.length === 1
                                        ? `Daily at ${
                                            timeOptions.find(
                                              (opt) =>
                                                opt.value === exactTimes[0]
                                            )?.label
                                          }`
                                        : `Daily at ${exactTimes.length} times`}
                                  </span>
                                </>
                              )}
                            </button>
                          </div>
                        </div>
                      </div>

                      <AnimatePresence mode="popLayout">
                        {schedules && schedules.length > 0 && (
                          <motion.div
                            className="mt-6 px-1 select-none pointer-events-none schedule-manager-animate"
                            initial={{ opacity: 0, y: -20 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: -20 }}
                            transition={{
                              duration: 0.5,
                              type: "spring",
                              stiffness: 300,
                              damping: 20,
                            }}
                            onAnimationComplete={() => {
                              const element = document.querySelector(
                                ".schedule-manager-animate"
                              );
                              if (element) {
                                element.classList.remove(
                                  "select-none",
                                  "pointer-events-none"
                                );
                              }
                            }}
                          >
                            <h6 className="text-gray-900 dark:text-white mb-4 text-lg font-semibold">
                              Active Schedules
                            </h6>

                            <div className="grid grid-cols-1 gap-4">
                              <AnimatePresence mode="popLayout">
                                {schedules?.map((schedule) => (
                                  <motion.div
                                    key={schedule.id}
                                    initial={{ opacity: 0, y: 20 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    exit={{ opacity: 0, y: -20 }}
                                    transition={{ duration: 0.3 }}
                                    className="bg-gray-200/50 dark:bg-gray-800/50 p-3 rounded-lg shadow-md border border-gray-300 dark:border-gray-900"
                                  >
                                    <div className="flex flex-col gap-2">
                                      <div className="flex items-center justify-between">
                                        <h6 className="text-gray-900 dark:text-white font-medium flex items-center gap-2">
                                          {schedule.interval.startsWith(
                                            "exact:"
                                          ) ? (
                                            <>
                                              <ClockIcon className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                                              <span>
                                                Daily at{" "}
                                                {(() => {
                                                  const times =
                                                    schedule.interval
                                                      .substring(6)
                                                      .split(",");
                                                  if (times.length === 1) {
                                                    return new Date(
                                                      `2000-01-01T${times[0]}:00`
                                                    ).toLocaleTimeString([], {
                                                      hour: "numeric",
                                                      minute: "2-digit",
                                                      hour12: true,
                                                    });
                                                  } else {
                                                    return `${times.length} times`;
                                                  }
                                                })()}
                                              </span>
                                            </>
                                          ) : (
                                            <>
                                              <ArrowPathIcon className="w-4 h-4 text-green-600 dark:text-green-400" />
                                              <span>
                                                Every {schedule.interval}
                                              </span>
                                            </>
                                          )}
                                        </h6>
                                        <button
                                          onClick={() =>
                                            schedule.id &&
                                            handleDeleteSchedule(schedule.id)
                                          }
                                          className="text-gray-500 dark:text-gray-400 p-1 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-md hover:bg-red-200/50 dark:hover:bg-red-900/50 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                                          title="Delete schedule"
                                        >
                                          <XMarkIcon className="h-4 w-4" />
                                        </button>
                                      </div>
                                      <p className="text-gray-600 dark:text-gray-400 text-sm">
                                        <span className="font-medium">
                                          Server:
                                        </span>{" "}
                                        <span className="truncate">
                                          {getServerNames(schedule.serverIds)}
                                        </span>
                                      </p>
                                      {schedule.interval.startsWith("exact:") &&
                                        schedule.interval
                                          .substring(6)
                                          .split(",").length > 1 && (
                                          <div className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                                            <span className="font-medium">
                                              Times:
                                            </span>{" "}
                                            <span className="text-blue-600 dark:text-blue-400">
                                              {schedule.interval
                                                .substring(6)
                                                .split(",")
                                                .map((time) =>
                                                  new Date(
                                                    `2000-01-01T${time}:00`
                                                  ).toLocaleTimeString([], {
                                                    hour: "numeric",
                                                    minute: "2-digit",
                                                    hour12: true,
                                                  })
                                                )
                                                .join(", ")}
                                            </span>
                                          </div>
                                        )}
                                      <p className="text-gray-600 dark:text-gray-400 text-xs pt-2">
                                        <span className="font-normal">
                                          Next run in:
                                        </span>{" "}
                                        <span className="font-medium text-blue-600 dark:text-blue-400">
                                          {formatNextRun(schedule.nextRun)}
                                        </span>
                                      </p>
                                    </div>
                                  </motion.div>
                                ))}
                              </AnimatePresence>
                            </div>
                          </motion.div>
                        )}
                      </AnimatePresence>
                    </div>
                  </div>
                </div>
              )}
            </div>
          );
        }}
      </Disclosure>
    </div>
  );
}
