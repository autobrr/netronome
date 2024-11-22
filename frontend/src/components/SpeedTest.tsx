import React, { useState, useEffect } from "react";
import { Container } from "@mui/material";
import { IoIosPulse, IoMdGitCompare } from "react-icons/io";
import { FaArrowDown, FaArrowUp } from "react-icons/fa";
import { ServerList } from "./speedtest/ServerList";
import { TestProgress } from "./speedtest/TestProgress";
import { SpeedHistoryChart } from "./speedtest/SpeedHistoryChart";
import ScheduleManager from "./ScheduleManager";
import {
  Server,
  SpeedTestResult,
  TestProgress as TestProgressType,
  TimeRange,
} from "../types/types";
import { Disclosure, DisclosureButton } from "@headlessui/react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";

interface TestOptions {
  serverId?: string;
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  multiServer: boolean;
}

interface SpeedUpdate {
  isComplete: boolean;
  type: "download" | "upload" | "ping" | "complete";
  speed: number;
  progress: number;
  serverName: string;
  latency?: number;
}

export default function SpeedTest() {
  const [loading, setLoading] = useState(false);
  const [servers, setServers] = useState<Server[]>([]);
  const [history, setHistory] = useState<SpeedTestResult[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [options, setOptions] = useState<TestOptions>({
    enableDownload: true,
    enableUpload: true,
    enablePacketLoss: true,
    multiServer: false,
  });
  const [selectedServers, setSelectedServers] = useState<Server[]>([]);
  const [progress, setProgress] = useState<TestProgressType | null>(null);
  const [testStatus, setTestStatus] = useState<"idle" | "running" | "complete">(
    "idle"
  );
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    const saved = localStorage.getItem("speedtest-time-range");
    return (saved as TimeRange) || "1w";
  });

  const fetchServers = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/servers");
      if (!response.ok)
        throw new Error(`HTTP error! status: ${response.status}`);
      const text = await response.text();
      try {
        const data = JSON.parse(text);
        setServers(data);
      } catch (parseError) {
        throw new Error(
          `Invalid JSON response (${
            (parseError as Error).message
          }): ${text.substring(0, 100)}...`
        );
      }
    } catch (error) {
      setError(
        error instanceof Error
          ? `Server Error: ${error.message}`
          : "Failed to fetch servers"
      );
    } finally {
      setLoading(false);
    }
  };

  const fetchHistory = async () => {
    try {
      const response = await fetch("/api/speedtest/history");
      if (!response.ok)
        throw new Error(`HTTP error! status: ${response.status}`);
      const data = await response.json();
      setHistory(data);
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to fetch history"
      );
    }
  };

  const handleServerSelect = (server: Server) => {
    setSelectedServers((prev) => {
      const isSelected = prev.some((s) => s.id === server.id);
      if (!options.multiServer) {
        return isSelected ? [] : [server];
      }
      return isSelected
        ? prev.filter((s) => s.id !== server.id)
        : [...prev, server];
    });
  };

  useEffect(() => {
    fetchServers();
    fetchHistory();
  }, []);

  useEffect(() => {
    if (!loading) return;

    const pollInterval = setInterval(async () => {
      try {
        const response = await fetch("/api/speedtest/status");
        if (!response.ok) return;

        const update: SpeedUpdate = await response.json();
        if (update.isComplete || update.type === "complete") {
          clearInterval(pollInterval);
          setTestStatus("complete");
          setProgress(null);
          setLoading(false);
        } else if (update.speed > 0) {
          setTestStatus("running");
          setProgress({
            currentServer: update.serverName,
            currentTest: update.type,
            currentSpeed: update.speed,
            progress: update.progress,
            isComplete: update.isComplete,
            type: update.type,
            speed: update.speed,
            latency: update.latency,
          });
        }
      } catch (error) {
        console.error("Failed to fetch status:", error);
      }
    }, 1000);

    return () => clearInterval(pollInterval);
  }, [loading]);

  return (
    <div className="min-h-screen">
      <Container maxWidth="lg" className="pb-8">
        {/* Header */}
        <div className="flex justify-between items-center -ml-2 mb-8 pt-8">
          <div className="flex items-center gap-3">
            <img
              src="/src/assets/logo2.png"
              alt="NetMetronome Logo"
              className="h-12 w-12 select-none pointer-events-none"
              draggable="false"
            />
            <h1 className="text-3xl font-bold text-white select-none">
              NetMetronome
            </h1>
          </div>
        </div>

        {/* Test Progress */}
        {testStatus === "running" && progress && (
          <TestProgress progress={progress} />
        )}

        {/* Error Messages */}
        {error && (
          <div className="bg-red-900/50 border border-red-700 text-red-200 px-4 py-3 rounded-xl mb-4">
            {error}
          </div>
        )}

        {/* Latest Results */}
        {history.length > 0 && (
          <div className="mb-6">
            <h2 className="text-white text-xl font-semibold">Latest Run</h2>
            <div className="text-gray-400 text-sm mb-4">
              Last test run:{" "}
              {new Date(history[0].createdAt).toLocaleString(undefined, {
                dateStyle: "short",
                timeStyle: "short",
              })}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 cursor-default">
              <MetricCard
                icon={<IoIosPulse className="w-5 h-5 text-blue-400" />}
                title="Latency"
                value={parseFloat(history[0].latency).toFixed(2)}
                unit="ms"
              />
              <MetricCard
                icon={<FaArrowDown className="w-5 h-5 text-emerald-400" />}
                title="Download"
                value={history[0].downloadSpeed.toFixed(2)}
                unit="Mbps"
              />
              <MetricCard
                icon={<FaArrowUp className="w-5 h-5 text-purple-400" />}
                title="Upload"
                value={history[0].uploadSpeed.toFixed(2)}
                unit="Mbps"
              />
              <MetricCard
                icon={<IoMdGitCompare className="w-5 h-5 text-blue-400" />}
                title="Jitter"
                value={history[0].jitter?.toFixed(2) ?? "N/A"}
                unit="ms"
              />
            </div>
          </div>
        )}

        {/* Speed History Chart */}
        {history.length > 0 && (
          <SpeedHistoryChart
            history={history}
            timeRange={timeRange}
            onTimeRangeChange={setTimeRange}
          />
        )}

        {/* Server Selection */}
        <Disclosure defaultOpen={false}>
          {({ open }) => (
            <div className="flex flex-col mb-6">
              <DisclosureButton
                className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
                  open ? "rounded-t-xl border-b-0" : "rounded-xl"
                } shadow-lg border-b-0 border-gray-900 text-left`}
              >
                <div className="flex flex-col">
                  <h2 className="text-white text-xl font-semibold p-1 select-none">
                    Server Selection
                  </h2>
                </div>
                <div className="flex items-center gap-2">
                  {selectedServers.length > 0 && (
                    <span className="text-gray-400">
                      {selectedServers.length} server
                      {selectedServers.length !== 1 ? "s" : ""} selected
                    </span>
                  )}
                  <ChevronDownIcon
                    className={`${
                      open ? "transform rotate-180" : ""
                    } w-5 h-5 text-gray-400 transition-transform duration-200`}
                  />
                </div>
              </DisclosureButton>

              {open && (
                <div className="bg-gray-850/95 px-4 rounded-b-xl shadow-lg">
                  <div className="flex flex-col">
                    <p className="text-gray-400 text-sm pl-1 select-none pointer-events-none">
                      Select one or more servers to test
                    </p>
                  </div>
                  <ServerList
                    servers={servers}
                    selectedServers={selectedServers}
                    onSelect={handleServerSelect}
                    multiSelect={options.multiServer}
                    onMultiSelectChange={(enabled) =>
                      setOptions((prev) => ({ ...prev, multiServer: enabled }))
                    }
                  />
                </div>
              )}
            </div>
          )}
        </Disclosure>

        {/* Schedule Manager */}
        <ScheduleManager
          servers={servers}
          selectedServers={selectedServers}
          onServerSelect={handleServerSelect}
          loading={loading}
        />
      </Container>
    </div>
  );
}

// Helper Components
const MetricCard: React.FC<{
  icon: React.ReactNode;
  title: string;
  value: string;
  unit: string;
}> = ({ icon, title, value, unit }) => (
  <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900">
    <div className="flex items-center gap-3 mb-4">
      {icon}
      <h3 className="text-gray-400 font-medium">{title}</h3>
    </div>
    <div className="text-white text-3xl font-bold">
      {value}
      <span className="text-xl font-normal text-gray-400 ml-1">{unit}</span>
    </div>
  </div>
);
