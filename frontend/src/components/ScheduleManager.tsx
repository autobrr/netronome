import { useState, useEffect } from "react";
import {
  Box,
  Button,
  Switch,
  Typography,
  FormControlLabel,
  CircularProgress,
  Tooltip,
} from "@mui/material";
import { Schedule, Server } from "../types/types";
import { Listbox } from "@headlessui/react";
import { ChevronUpDownIcon } from "@heroicons/react/20/solid";

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
  selectedServers,
  loading: parentLoading,
}: ScheduleManagerProps) {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [interval, setInterval] = useState("5m");
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
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

  const handleEnabledChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setEnabled(event.target.checked);
  };

  const isButtonDisabled =
    loading || parentLoading || selectedServers.length === 0;

  if (isInitialLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ mt: 4 }}>
      <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg mb-6 border border-gray-700">
        <Typography variant="h6" gutterBottom className="text-white mb-4">
          Schedule Speed Tests
        </Typography>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <div>
              <Listbox value={interval} onChange={setInterval}>
                <div className="relative">
                  <Listbox.Button className="relative w-full cursor-pointer rounded-lg bg-gray-800/50 py-2 pl-3 pr-10 text-left border border-gray-700 focus:outline-none focus-visible:border-blue-500 focus-visible:ring-2 focus-visible:ring-blue-500">
                    <span className="block truncate text-gray-200">
                      {
                        intervalOptions.find((opt) => opt.value === interval)
                          ?.label
                      }
                    </span>
                    <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                      <ChevronUpDownIcon
                        className="h-5 w-5 text-gray-400"
                        aria-hidden="true"
                      />
                    </span>
                  </Listbox.Button>
                  <Listbox.Options className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-gray-800 py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
                    {intervalOptions.map((option) => (
                      <Listbox.Option
                        key={option.value}
                        value={option.value}
                        className={({ active }) =>
                          `relative cursor-pointer select-none py-2 pl-3 pr-9 ${
                            active ? "bg-gray-700 text-white" : "text-gray-200"
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
                      </Listbox.Option>
                    ))}
                  </Listbox.Options>
                </div>
              </Listbox>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <FormControlLabel
              control={
                <Switch
                  checked={enabled}
                  onChange={handleEnabledChange}
                  className="text-blue-500"
                />
              }
              label="Enable Schedule"
              className="text-gray-300"
            />
            <Button
              variant="contained"
              onClick={handleCreateSchedule}
              disabled={isButtonDisabled}
              className="bg-blue-500 hover:bg-blue-600 text-white px-6 py-2 rounded-lg disabled:opacity-50"
            >
              {loading ? "Creating Schedule..." : "Create Schedule"}
            </Button>
          </div>
        </div>

        {error && (
          <Typography color="error" sx={{ mt: 2 }}>
            {error}
          </Typography>
        )}
      </div>

      {schedules.length > 0 && (
        <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-700">
          <Typography variant="h6" gutterBottom className="text-white mb-4">
            Active Schedules
          </Typography>

          <div className="space-y-4">
            {schedules.map((schedule) => (
              <div
                key={schedule.id}
                className="bg-gray-800/50 p-4 rounded-lg border border-gray-700"
              >
                <div className="flex flex-col sm:flex-row items-center gap-4">
                  <div className="w-full sm:w-1/2">
                    <div className="space-y-2">
                      <Typography
                        variant="subtitle1"
                        className="text-white font-medium"
                      >
                        Test Frequency: Every {schedule.interval}
                      </Typography>
                      <Typography variant="body2" className="text-gray-300">
                        {schedule.serverIds.length} Server
                        {schedule.serverIds.length !== 1 ? "s" : ""} Selected
                      </Typography>
                    </div>
                  </div>
                  <div className="w-full sm:w-1/2">
                    <div className="flex justify-between items-center">
                      <Tooltip
                        title={new Date(schedule.nextRun).toLocaleString()}
                        placement="top"
                      >
                        <Typography variant="body2" className="text-gray-300">
                          Next Run: {formatNextRun(schedule.nextRun)}
                        </Typography>
                      </Tooltip>
                      <Button
                        variant="outlined"
                        onClick={() =>
                          schedule.id && handleDeleteSchedule(schedule.id)
                        }
                        className="border-red-500 text-red-500 hover:bg-red-500/10"
                        size="small"
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </Box>
  );
}
