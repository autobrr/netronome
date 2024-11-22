import { useState, useEffect } from "react";
import { Box, CircularProgress } from "@mui/material";
import { Schedule, Server } from "../types/types";
import {
  DisclosureButton,
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
  Popover,
  PopoverButton,
  PopoverPanel,
} from "@headlessui/react";
import { ChevronUpDownIcon } from "@heroicons/react/20/solid";
import { Disclosure } from "@headlessui/react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";
import { motion, AnimatePresence } from "motion/react";

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

export default function ScheduleManager({
  servers,
  selectedServers,
}: ScheduleManagerProps) {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [interval, setInterval] = useState("5m");
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
      interval,
      nextRun: new Date(Date.now() + parseInterval(interval)).toISOString(),
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

  const parseInterval = (interval: string): number => {
    const value = parseInt(interval);
    const unit = interval.slice(-1);
    switch (unit) {
      case "m":
        return value * 60 * 1000;
      case "h":
        return value * 60 * 60 * 1000;
      case "d":
        return value * 24 * 60 * 60 * 1000;
      default:
        return value * 60 * 60 * 1000;
    }
  };

  const formatNextRun = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffMins = Math.round(diffMs / 60000);

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

  const getServerNames = (serverIds: string[]) => {
    return serverIds
      .map((id: string) => {
        const server = servers.find((s: Server) => s.id === id);
        return server ? `${server.sponsor} - ${server.name}` : null;
      })
      .filter(Boolean)
      .join(", ");
  };

  if (isInitialLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ mt: 4 }}>
      <Disclosure defaultOpen={false}>
        {({ open }) => (
          <div className="flex flex-col">
            <DisclosureButton
              className={`flex justify-between items-center w-full p-4 bg-gray-850/95 ${
                open ? "rounded-t-xl" : "rounded-xl"
              } shadow-lg border-b-0 border-gray-900 text-left`}
            >
              <h6 className="text-white text-xl ml-1 py-1 font-semibold">
                Schedule Manager
              </h6>
              <ChevronDownIcon
                className={`${
                  open ? "transform rotate-180" : ""
                } w-5 h-5 text-gray-400 transition-transform duration-200`}
              />
            </DisclosureButton>

            <AnimatePresence>
              {open && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.2 }}
                  className="bg-gray-850/95 p-4 rounded-b-xl shadow-lg border-t-0 border-gray-900"
                >
                  <div className="flex flex-col gap-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
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
                          <div className="flex items-center justify-between pt-4 pl-1">
                            <Popover className="relative">
                              <PopoverButton
                                className="bg-blue-500 hover:bg-blue-600 text-white text-sm px-3 py-2 rounded"
                                onClick={handleCreateSchedule}
                              >
                                {loading
                                  ? "Creating schedule..."
                                  : "Create schedule"}
                              </PopoverButton>
                              <PopoverPanel className="absolute z-10 left-full ml-2 top-1/2 -translate-y-1/2 transform w-64">
                                <div className="overflow-hidden rounded-lg shadow-lg ring-1 ring-black ring-opacity-5">
                                  <div className="relative bg-gray-800 p-3">
                                    <p className="text-sm text-white">
                                      Please select one or more servers from the
                                      Server Selection section to create a
                                      schedule
                                    </p>
                                  </div>
                                </div>
                              </PopoverPanel>
                            </Popover>
                          </div>
                        </Listbox>
                      </div>
                    </div>

                    <div className="flex items-center justify-between mt-4"></div>

                    {schedules.length > 0 && (
                      <div className="p-6 rounded-xl">
                        <h6 className="text-white mb-4 text-lg font-semibold">
                          Active Schedules
                        </h6>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          {schedules.map((schedule) => (
                            <div
                              key={schedule.id}
                              className="bg-gray-800/50 p-4 rounded-lg border border-gray-900 flex flex-col"
                            >
                              <div className="flex flex-col">
                                <h6 className="text-white font-medium">
                                  <strong>Test Frequency:</strong> Every{" "}
                                  {schedule.interval}
                                </h6>
                                <p className="text-gray-300">
                                  <strong>Selected Servers:</strong>{" "}
                                  {getServerNames(schedule.serverIds)}
                                </p>
                                <p className="text-gray-300">
                                  <strong>Next Run:</strong>{" "}
                                  <span className="text-blue-400">
                                    {formatNextRun(schedule.nextRun)}
                                  </span>
                                </p>
                              </div>
                              <div className="flex justify-end mt-2">
                                <button
                                  onClick={() =>
                                    schedule.id &&
                                    handleDeleteSchedule(schedule.id)
                                  }
                                  className="border-red-500 text-red-500 bg-red-200/10 hover:bg-red-500/10 transition-colors ease-in-out ml-2 px-4 py-2 rounded"
                                >
                                  Delete
                                </button>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        )}
      </Disclosure>
    </Box>
  );
}
