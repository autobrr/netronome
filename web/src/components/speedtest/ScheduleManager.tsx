/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { Schedule, Server, SavedIperfServer } from "@/types/types";
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
import { ChevronDownIcon, XMarkIcon } from "@heroicons/react/20/solid";
import { motion, AnimatePresence } from "motion/react";

interface ScheduleManagerProps {
  servers: Server[];
  selectedServers: Server[];
  onServerSelect: (server: Server) => void;
}

interface IntervalOption {
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

const calculateNextRun = (intervalStr: string): string => {
  const milliseconds = parseInterval(intervalStr);
  return new Date(Date.now() + milliseconds).toISOString();
};

export default function ScheduleManager({
  servers,
  selectedServers,
}: ScheduleManagerProps) {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [iperfServers, setIperfServers] = useState<SavedIperfServer[]>([]);
  const [interval, setInterval] = useState<string>("1h");
  const [enabled] = useState(true);
  const [, setError] = useState<string | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(true);
  const [isOpen] = useState(() => {
    const saved = localStorage.getItem("schedule-manager-open");
    return saved === null ? true : saved === "true";
  });

  useEffect(() => {
    fetchSchedules();
    fetchIperfServers();
  }, []);

  const fetchSchedules = async () => {
    try {
      if (schedules.length === 0) {
        setIsInitialLoading(true);
      }
      const response = await fetch("/api/schedules");
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setSchedules(data || []);
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to fetch schedules"
      );
      setSchedules([]);
    } finally {
      setIsInitialLoading(false);
    }
  };

  const fetchIperfServers = async () => {
    try {
      const response = await fetch("/api/iperf/servers");
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
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

    const newSchedule: Schedule = {
      serverIds: selectedServers.map((s) => s.id),
      interval: interval,
      nextRun: calculateNextRun(interval),
      enabled,
      options: {
        enableDownload: true,
        enableUpload: true,
        enablePacketLoss: true,
        serverIds: selectedServers.map((s) => s.id),
        useIperf: isIperfServer,
        serverHost: isIperfServer ? selectedServers[0].host : undefined,
      },
    };

    try {
      const response = await fetch("/api/schedules", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(newSchedule),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setSchedules((prev) => [...prev, data]);
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to create schedule"
      );
      // Fetch only on error to restore state
      fetchSchedules();
    }
  };

  const handleDeleteSchedule = async (id: number) => {
    setSchedules((prev) => prev.filter((schedule) => schedule.id !== id));

    try {
      const response = await fetch(`/api/schedules/${id}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to delete schedule"
      );
      // Fetch only on error to restore state
      fetchSchedules();
    }
  };

  const formatNextRun = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffMins = Math.round(diffMs / (60 * 1000));

    if (diffMins < 60) {
      return `${diffMins} minute${diffMins !== 1 ? "s" : ""}`;
    } else if (diffMins < 1440) {
      const hours = Math.floor(diffMins / 60);
      return `${hours} hour${hours !== 1 ? "s" : ""}`;
    } else {
      const days = Math.floor(diffMins / 1440);
      return `${days} day${days !== 1 ? "s" : ""}`;
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
                <span className="text-purple-400 drop-shadow-[0_0_1px_rgba(168,85,247,0.8)]">
                  iperf3
                </span>
              </span>
              {iperfServer?.name && (
                <span
                  className="
                    absolute top-full left-1/2 transform -translate-x-1/2 mt-2
                    px-3 py-2 text-sm
                    text-gray-200 bg-gray-800/95
                    rounded-lg shadow-lg
                    border border-gray-700/50
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
                    before:bg-gray-800/95
                    before:border-t before:border-l
                    before:border-gray-700/50
                  "
                >
                  {host}
                </span>
              )}
            </span>
          );
        }

        const server = servers.find((s: Server) => s.id === id);
        return server ? (
          <span key={id}>
            {server.sponsor} - {server.name} -{" "}
            <span className="text-emerald-400 drop-shadow-[0_0_1px_rgba(251,191,36,0.8)]">
              speedtest.net
            </span>
          </span>
        ) : null;
      })
      .filter(Boolean);

    if (serversList.length === 1) {
      return serversList[0];
    } else if (serversList.length > 1) {
      return `${serversList.length} servers`;
    }
    return "No servers";
  };

  if (isInitialLoading) {
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
                className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
                  open ? "rounded-t-xl" : "rounded-xl"
                } shadow-lg border-b-0 border-gray-900 text-left`}
              >
                <div className="flex flex-col">
                  <h2 className="text-white text-xl font-semibold p-1 select-none">
                    Schedule Manager
                  </h2>
                  <p className="text-gray-400 text-sm pl-1 pb-1">
                    Create and manage your schedules
                  </p>
                </div>
                <ChevronDownIcon
                  className={`${
                    open ? "transform rotate-180" : ""
                  } w-5 h-5 text-gray-400 transition-transform duration-200`}
                />
              </DisclosureButton>

              {open && (
                <div className="bg-gray-850/95 px-4 pt-3 rounded-b-xl shadow-lg flex-1">
                  <div className="flex flex-col pl-1">
                    <div className="flex flex-col gap-4 pb-4">
                      <div className="grid grid-cols-1 gap-4">
                        <div>
                          <Listbox value={interval} onChange={setInterval}>
                            <div className="relative">
                              <ListboxButton className="relative w-full px-4 py-2 bg-gray-800/50 border border-gray-900 rounded-lg text-left text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                                <span className="block truncate">
                                  {
                                    intervalOptions.find(
                                      (opt) => opt.value === interval
                                    )?.label
                                  }
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
                                <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-gray-800 border border-gray-900 py-1 shadow-lg focus:outline-none">
                                  {intervalOptions.map((option) => (
                                    <ListboxOption
                                      key={option.value}
                                      value={option.value}
                                      className={({ focus }) =>
                                        `relative cursor-pointer select-none py-2 px-4 ${
                                          focus
                                            ? "bg-blue-500/10 text-blue-200"
                                            : "text-gray-300"
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

                          <div className="flex items-center justify-between mt-4">
                            <div className="relative inline-block group">
                              <button
                                className={`ml-1 px-3 py-2 rounded-lg shadow-md transition-colors border ${
                                  selectedServers.length === 0
                                    ? "bg-gray-700 text-gray-400 cursor-not-allowed border-gray-900"
                                    : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                                }`}
                                onClick={handleCreateSchedule}
                                disabled={selectedServers.length === 0}
                              >
                                Create schedule
                              </button>
                              {selectedServers.length === 0 && (
                                <div className="absolute bottom-full border border-gray-900 left-1/2 transform -translate-x-1/2 mb-2 px-3 py-1 text-sm text-white bg-gray-800 rounded-md invisible group-hover:visible transition-all duration-200 whitespace-nowrap">
                                  Pick a server first
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>

                      <AnimatePresence mode="popLayout">
                        {schedules.length > 0 && (
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
                            <h6 className="text-white mb-4 text-lg font-semibold">
                              Active Schedules
                            </h6>

                            <div className="grid grid-cols-1 gap-4">
                              <AnimatePresence mode="popLayout">
                                {schedules.map((schedule) => (
                                  <motion.div
                                    key={schedule.id}
                                    initial={{ opacity: 0, y: 20 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    exit={{ opacity: 0, y: -20 }}
                                    transition={{ duration: 0.3 }}
                                    className="bg-gray-800/50 p-3 rounded-lg shadow-md border border-gray-900"
                                  >
                                    <div className="flex flex-col gap-2">
                                      <div className="flex items-center justify-between">
                                        <h6 className="text-white font-medium">
                                          Every {schedule.interval}
                                        </h6>
                                        <button
                                          onClick={() =>
                                            schedule.id &&
                                            handleDeleteSchedule(schedule.id)
                                          }
                                          className="text-red-500 p-1 bg-red-900/50 border border-gray-900 rounded-md hover:bg-red-900/70 hover:text-red-400 transition-colors"
                                          title="Delete schedule"
                                        >
                                          <XMarkIcon className="h-4 w-4" />
                                        </button>
                                      </div>
                                      <p className="text-gray-400 text-sm">
                                        <span className="font-medium">
                                          Server:
                                        </span>{" "}
                                        <span className="truncate">
                                          {getServerNames(schedule.serverIds)}
                                        </span>
                                      </p>
                                      <p className="text-gray-400 text-xs pt-2">
                                        <span className="font-normal">
                                          Next run in:
                                        </span>{" "}
                                        <span className="font-medium text-blue-400">
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
