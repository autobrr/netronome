/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { Schedule, Server } from "@/types/types";
import {
  DisclosureButton,
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
} from "@headlessui/react";
import { ChevronUpDownIcon } from "@heroicons/react/20/solid";
import { Disclosure } from "@headlessui/react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";
import { motion } from "motion/react";

interface ScheduleManagerProps {
  servers: Server[];
  selectedServers: Server[];
  onServerSelect: (server: Server) => void;
  loading: boolean;
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
  const [interval, setInterval] = useState<string>("1h");
  const [enabled] = useState(true);
  const [loading, setLoading] = useState(false);
  const [, setError] = useState<string | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(true);

  useEffect(() => {
    fetchSchedules();
  }, []);

  const fetchSchedules = async () => {
    try {
      setIsInitialLoading(true);
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

  const handleCreateSchedule = async () => {
    if (selectedServers.length === 0) {
      setError("Please select at least one server");
      return;
    }

    setLoading(true);
    setError(null);

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

      await fetchSchedules();
      setLoading(false);
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to create schedule"
      );
      setLoading(false);
    }
  };

  const handleDeleteSchedule = async (id: number) => {
    try {
      const response = await fetch(`/api/schedules/${id}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      await fetchSchedules();
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to delete schedule"
      );
    }
  };

  const formatNextRun = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffMins = Math.round(diffMs / (60 * 1000));

    if (diffMins < 60) {
      return `in ${diffMins} minute${diffMins !== 1 ? "s" : ""}`;
    } else if (diffMins < 1440) {
      const hours = Math.floor(diffMins / 60);
      return `in ${hours} hour${hours !== 1 ? "s" : ""}`;
    } else {
      const days = Math.floor(diffMins / 1440);
      return `in ${days} day${days !== 1 ? "s" : ""}`;
    }
  };

  const getServerNames = (serverIds: string[] | undefined) => {
    const serversList = (serverIds || [])
      .map((id: string) => {
        const server = servers.find((s: Server) => s.id === id);
        return server ? `${server.sponsor} - ${server.name}` : null;
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
      <Disclosure defaultOpen={true}>
        {({ open }) => (
          <div className="flex flex-col h-full">
            <DisclosureButton
              className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
                open ? "rounded-t-xl" : "rounded-xl"
              } shadow-lg border-b-0 border-gray-900 text-left`}
            >
              <h6 className="text-white text-xl ml-1 py-1 font-semibold select-none">
                Schedule Manager
              </h6>
              <ChevronDownIcon
                className={`${
                  open ? "transform rotate-180" : ""
                } w-5 h-5 text-gray-400 transition-transform duration-200`}
              />
            </DisclosureButton>

            {open && (
              <motion.div
                className="bg-gray-850/95 p-4 rounded-b-xl shadow-lg border-t-0 border-gray-900 select-none pointer-events-none schedule-manager-animate h-full"
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
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
                <div className="flex flex-col gap-4 pb-4">
                  <div className="grid grid-cols-1 gap-4">
                    <div>
                      <Listbox value={interval} onChange={setInterval}>
                        <div className="relative">
                          <ListboxButton className="relative w-full cursor-pointer rounded-lg bg-gray-800/50 py-2 pl-3 pr-10 text-left border border-gray-900 focus:outline-none focus-visible:border-blue-500 focus-visible:ring-2 focus-visible:ring-blue-500">
                            <span className="block truncate text-gray-200">
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
                          <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-gray-800 py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
                            {intervalOptions.map((option) => (
                              <ListboxOption
                                key={option.value}
                                value={option.value}
                                className={({ active }) =>
                                  `relative cursor-pointer select-none py-2 pl-3 pr-9 ${
                                    active
                                      ? "bg-gray-700 text-white"
                                      : "text-gray-200"
                                  }`
                                }
                              >
                                {({ selected }) => (
                                  <span
                                    className={`block truncate ${
                                      selected ? "font-medium" : "font-normal"
                                    }`}
                                  >
                                    {option.label}
                                  </span>
                                )}
                              </ListboxOption>
                            ))}
                          </ListboxOptions>
                        </div>
                      </Listbox>

                      <div className="flex items-center justify-between mt-4">
                        <div className="relative inline-block group">
                          <button
                            className={`ml-1 px-3 py-2 rounded-lg transition-colors ${
                              loading || selectedServers.length === 0
                                ? "bg-gray-700 text-gray-400 cursor-not-allowed"
                                : "bg-blue-500 hover:bg-blue-600 text-white"
                            }`}
                            onClick={handleCreateSchedule}
                            disabled={loading || selectedServers.length === 0}
                          >
                            {loading
                              ? "Creating schedule..."
                              : "Create schedule"}
                          </button>
                          {selectedServers.length === 0 && (
                            <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-3 py-1 text-sm text-white bg-gray-800 rounded-md invisible group-hover:visible transition-all duration-200 whitespace-nowrap">
                              Pick a server first
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>

                  {schedules.length > 0 && (
                    <div className="mt-6 px-1">
                      <h6 className="text-white mb-4 text-lg font-semibold">
                        Active Schedules
                      </h6>

                      <div className="grid grid-cols-1 gap-4">
                        {schedules.map((schedule) => (
                          <motion.div
                            key={schedule.id}
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.3 }}
                            className="bg-gray-800/50 p-4 rounded-lg border border-gray-900"
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
                                  className="text-red-500 px-2 py-1 bg-red-800/50 border border-gray-900 rounded-lg hover:bg-red-900/70 hover:text-red-400 transition-colors"
                                >
                                  Delete
                                </button>
                              </div>
                              <p className="text-gray-400 text-sm">
                                <span className="font-medium">Servers:</span>{" "}
                                <span className="truncate">
                                  {getServerNames(schedule.serverIds)}
                                </span>
                              </p>
                              <p className="text-gray-400 text-sm">
                                <span className="font-medium">Next run:</span>{" "}
                                <span className="text-blue-400">
                                  {formatNextRun(schedule.nextRun)}
                                </span>
                              </p>
                            </div>
                          </motion.div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </motion.div>
            )}
          </div>
        )}
      </Disclosure>
    </div>
  );
}
