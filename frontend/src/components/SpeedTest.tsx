import React, { useState, useEffect, useMemo } from "react";
import { Container } from "@mui/material";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import ScheduleManager from "./ScheduleManager";
import { Server } from "../types/types";
import { ArrowDownIcon, ArrowUpIcon } from "@heroicons/react/24/solid";
import { IoIosPulse, IoMdGitCompare } from "react-icons/io";
import { Switch } from "@headlessui/react";

interface SpeedTestResult {
  id: number;
  serverName: string;
  serverId: string;
  downloadSpeed: number;
  uploadSpeed: number;
  latency: string;
  packetLoss: number;
  jitter?: number;
  createdAt: string;
}

interface TestOptions {
  serverId?: string;
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  multiServer: boolean;
}

interface ServerListProps {
  servers: Server[];
  selectedServers: Server[];
  onSelect: (server: Server) => void;
}

interface SpeedUpdate {
  type: "download" | "upload" | "ping" | "complete";
  serverName: string;
  speed: number;
  progress: number;
  isComplete: boolean;
  latency?: string;
}

interface TestProgress {
  currentServer: string;
  currentTest: string;
  currentSpeed: number;
  currentLatency?: string;
  progress: number;
  isComplete: boolean;
  latency?: string;
  packetLoss?: number;
  type: "download" | "upload" | "ping";
  speed: number;
}

const ServerList: React.FC<ServerListProps> = ({
  servers,
  selectedServers,
  onSelect,
}) => {
  const [displayCount, setDisplayCount] = useState(9);

  // First, flatten the servers but keep track of their provider
  const flattenedServers = useMemo(() => {
    // Group servers by provider first
    const groups = servers.reduce((acc, server) => {
      const key = server.sponsor || "Unknown Provider";
      if (!acc[key]) {
        acc[key] = [];
      }
      acc[key].push(server);
      return acc;
    }, {} as Record<string, Server[]>);

    // Sort each provider's servers by distance
    Object.keys(groups).forEach((key) => {
      groups[key].sort((a, b) => a.distance - b.distance);
    });

    // Flatten while keeping providers together
    return Object.entries(groups)
      .sort(([, a], [, b]) => a[0].distance - b[0].distance)
      .flatMap(([provider, servers]) =>
        servers.map((server) => ({
          ...server,
          provider,
        }))
      );
  }, [servers]);

  const visibleServers = flattenedServers.slice(0, displayCount);
  const remainingCount = flattenedServers.length - visibleServers.length;

  // Create rows of servers
  const rows = [];
  for (let i = 0; i < visibleServers.length; i += 3) {
    rows.push(visibleServers.slice(i, i + 3));
  }

  return (
    <div className="space-y-4">
      {rows.map((rowServers, rowIndex) => (
        <div
          key={rowIndex}
          className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 mb-3"
        >
          {rowServers.map((server) => (
            <div
              key={server.id}
              onClick={() => onSelect(server)}
              className={`flex flex-col p-3 rounded-md cursor-pointer transition-colors
                ${rowIndex >= rows.length - 1 ? "animate-fade-in" : ""}
                ${
                  selectedServers.some((s) => s.id === server.id)
                    ? "bg-blue-500/10 border border-blue-500/30"
                    : "bg-gray-850/95 hover:bg-gray-850/70 border border-gray-900"
                }`}
            >
              <div className="text-sm text-blue-400 font-medium mb-2">
                {server.provider}
              </div>
              <div className="flex items-center justify-between">
                <span className="font-medium text-gray-200">{server.name}</span>
                <span className="text-xs bg-gray-700 rounded px-1.5 py-0.5 text-gray-300">
                  {server.distance.toFixed(0)}km
                </span>
              </div>
              <div className="text-xs text-gray-400 truncate mt-1">
                {server.host}
              </div>
              <div className="text-xs text-gray-500 mt-1">ID: {server.id}</div>
            </div>
          ))}
        </div>
      ))}

      {remainingCount > 0 && (
        <button
          onClick={() => setDisplayCount((prev) => prev + 9)}
          className="w-full py-2 text-sm text-gray-400 hover:text-gray-300 transition-colors"
        >
          Show More Servers ({remainingCount} remaining)
        </button>
      )}
    </div>
  );
};

const TestProgress: React.FC<{ progress: TestProgress }> = ({ progress }) => {
  return (
    <div className="bg-gray-850/95 p-4 rounded-xl shadow-lg mb-6 border border-gray-900">
      <h2 className="text-xl font-semibold mb-2 text-white">
        Test in Progress
      </h2>
      <div className="space-y-2 text-gray-300">
        <div>Testing Server: {progress.currentServer}</div>
        {progress.currentTest && (
          <div className="flex items-center gap-4">
            <div className="capitalize">{progress.currentTest} Test:</div>
            {progress.currentSpeed > 0 && (
              <div className="group relative text-xl font-bold">
                {progress.currentSpeed.toFixed(2)} Mbps
                {/* Add tooltip */}
                {progress.packetLoss !== undefined && (
                  <div className="absolute hidden group-hover:block bottom-full left-1/2 -translate-x-1/2 p-2 bg-gray-900 rounded-lg text-sm whitespace-nowrap">
                    Packet Loss: {progress.packetLoss.toFixed(2)}%
                  </div>
                )}
              </div>
            )}
            <div className="animate-pulse text-blue-500">Running...</div>
          </div>
        )}
      </div>
    </div>
  );
};

type TimeRange = "1d" | "3d" | "1w" | "1m" | "all";

const SpeedHistoryChart: React.FC<{
  history: SpeedTestResult[];
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
}> = ({ history, timeRange, onTimeRangeChange }) => {
  const filterDataByTimeRange = (data: SpeedTestResult[]) => {
    const now = new Date();
    const ranges = {
      "1d": 1,
      "3d": 3,
      "1w": 7,
      "1m": 30,
      all: Infinity,
    };
    const daysAgo = ranges[timeRange];

    return data.filter((result) => {
      const resultDate = new Date(result.createdAt);
      const diffTime = Math.abs(now.getTime() - resultDate.getTime());
      const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
      return diffDays <= daysAgo;
    });
  };

  const filteredData = filterDataByTimeRange(history);
  const chartData = [...filteredData].reverse().map((result) => ({
    timestamp: new Date(result.createdAt).toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
    }),
    download: result.downloadSpeed,
    upload: result.uploadSpeed,
    latency: parseFloat(result.latency.replace("ms", "")),
    jitter: result.jitter,
    server: result.serverName,
  }));

  return (
    <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg mb-6 border border-gray-900">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-bold text-white">Speed History</h2>
        <div className="flex gap-2">
          {(["1d", "3d", "1w", "1m", "all"] as TimeRange[]).map((range) => (
            <button
              key={range}
              onClick={() => onTimeRangeChange(range)}
              className={`px-3 py-1 rounded-md text-sm ${
                timeRange === range
                  ? "bg-blue-500 text-white"
                  : "bg-gray-700 text-gray-300 hover:bg-gray-600"
              }`}
            >
              {range === "all" ? "All" : range.toUpperCase()}
            </button>
          ))}
        </div>
      </div>
      <div className="h-[400px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            data={chartData}
            margin={{ top: 5, right: 30, left: 20, bottom: 25 }}
            className="[&_.recharts-cartesian-grid-horizontal]:stroke-gray-700/50 
                      [&_.recharts-cartesian-axis-line]:stroke-gray-700
                      [&_.recharts-legend-item-text]:!text-gray-300
                      [&_.recharts-text]:!fill-gray-400
                      [&_.recharts-tooltip]:!bg-gray-850/95
                      [&_.recharts-tooltip]:!border-gray-900
                      [&_.recharts-tooltip]:!rounded-lg
                      [&_.recharts-tooltip]:!shadow-xl
                      [&_.recharts-tooltip]:!backdrop-blur-sm"
          >
            <CartesianGrid
              strokeDasharray="3 3"
              horizontal={true}
              vertical={false}
            />
            <XAxis dataKey="timestamp" height={60} tickMargin={10} />
            <YAxis
              yAxisId="speed"
              label={{
                value: "Speed (Mbps)",
                position: "insideLeft",
                angle: -90,
                offset: 0,
                className: "fill-gray-400",
              }}
              domain={[0, "auto"]}
              allowDataOverflow={false}
            />
            <YAxis
              yAxisId="latency"
              orientation="right"
              label={{
                value: "ms",
                position: "insideRight",
                angle: -90,
                offset: 0,
                className: "fill-gray-400",
              }}
              domain={[0, "auto"]}
              allowDataOverflow={false}
            />
            <Tooltip
              wrapperClassName="!bg-gray-800/90 !border-gray-900 !rounded-lg !shadow-xl !backdrop-blur-sm"
              contentStyle={{
                backgroundColor: "transparent",
                border: "none",
                borderRadius: "0.5rem",
                padding: "1rem",
              }}
              itemStyle={{ color: "#E5E7EB" }}
              labelStyle={{ color: "#9CA3AF", fontWeight: "bold" }}
            />
            <Legend
              verticalAlign="top"
              height={36}
              iconType="circle"
              wrapperStyle={{
                paddingTop: "10px",
                paddingBottom: "10px",
              }}
            />
            <Line
              yAxisId="speed"
              type="monotone"
              dataKey="download"
              name="Download"
              stroke="#3B82F6"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              className="!stroke-blue-500"
            />
            <Line
              yAxisId="speed"
              type="monotone"
              dataKey="upload"
              name="Upload"
              stroke="#10B981"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              className="!stroke-emerald-500"
            />
            <Line
              yAxisId="latency"
              type="monotone"
              dataKey="latency"
              name="Latency"
              stroke="#F59E0B"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              className="!stroke-amber-500"
              strokeDasharray="3 3"
            />
            <Line
              yAxisId="latency"
              type="monotone"
              dataKey="jitter"
              name="Jitter"
              stroke="#9333EA"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6 }}
              className="!stroke-purple-500"
              strokeDasharray="5 5"
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default function SpeedTest() {
  const [loading, setLoading] = useState(false);
  const [servers, setServers] = useState<Server[]>([]);
  const [history, setHistory] = useState<SpeedTestResult[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [options, setOptions] = useState<TestOptions>({
    enableDownload: true,
    enableUpload: true,
    enablePacketLoss: true,
    multiServer: false,
  });
  const [selectedServers, setSelectedServers] = useState<Server[]>([]);
  const [progress, setProgress] = useState<TestProgress | null>(null);
  const [testStatus, setTestStatus] = useState<"idle" | "running" | "complete">(
    "idle"
  );
  const [timeRange, setTimeRange] = useState<TimeRange>("1w");
  const [isConfigExpanded, setIsConfigExpanded] = useState(false);

  const fetchServers = async () => {
    try {
      const response = await fetch("/api/servers");
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
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
    }
  };

  const fetchHistory = async () => {
    try {
      console.log("Fetching history...");
      const response = await fetch("/api/speedtest/history");
      console.log("Response status:", response.status);
      console.log("Response headers:", Object.fromEntries(response.headers));

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const text = await response.text();
      console.log("Raw history response:", text);

      try {
        const data = JSON.parse(text);
        console.log("Parsed history data:", data);
        if (Array.isArray(data) && data.length === 0) {
          console.log("Warning: Received empty array from API");
        }
        setHistory(data);
      } catch (parseError) {
        console.error("Parse error:", parseError);
        throw parseError;
      }
    } catch (error) {
      console.error("Error fetching history:", {
        error,
        type: error instanceof Error ? error.constructor.name : typeof error,
        message: error instanceof Error ? error.message : String(error),
      });
      setError(
        error instanceof Error ? error.message : "Failed to fetch history"
      );
    }
  };

  const runTest = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch("/api/speedtest", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          ...options,
          serverIds: selectedServers.map((s) => s.id),
        }),
      });
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const result = await response.json();
      if (result.error) {
        throw new Error(result.error);
      }
      await fetchHistory();
      setProgress(null);
      setTestStatus("complete");
      setLoading(false);
    } catch (error) {
      setError(
        error instanceof Error ? error.message : "Failed to run speed test"
      );
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServers();
    fetchHistory();
  }, []);

  const filteredServers = servers.filter(
    (server) =>
      server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.country.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (server.sponsor || "").toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleServerSelect = (server: Server) => {
    if (!options.multiServer) {
      setSelectedServers((prev) =>
        prev.some((s) => s.id === server.id) ? [] : [server]
      );
    } else {
      setSelectedServers((prev) => {
        const isSelected = prev.some((s) => s.id === server.id);
        if (isSelected) {
          return prev.filter((s) => s.id !== server.id);
        } else {
          return [...prev, server];
        }
      });
    }
  };

  useEffect(() => {
    if (!loading) return;

    const pollInterval = setInterval(async () => {
      try {
        const response = await fetch("/api/speedtest/status");
        if (!response.ok) return;

        const update: SpeedUpdate = await response.json();
        console.log("Received status update:", update);

        if (update.isComplete || update.type === "complete") {
          console.log("Test complete, stopping polling");
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
        {/* Header Section */}
        <div className="flex justify-between items-center mb-8 pt-8">
          <h1 className="text-3xl font-bold text-white">Speedtrackerr</h1>
        </div>

        {/* Active Test Progress - Show only when test is running */}
        {testStatus === "running" && progress && (
          <TestProgress progress={progress} />
        )}

        {/* Error Messages */}
        {error && (
          <div className="bg-red-900/50 border border-red-700 text-red-200 px-4 py-3 rounded-xl mb-4">
            {error}
          </div>
        )}

        {/* Latest Results - Show only if there's history */}
        <div className="mb-6">
          <h2 className="text-white text-xl font-semibold">Latest Results</h2>
          {history.length > 0 && (
            <div className="text-gray-400 text-sm mb-4">
              Last test run:{" "}
              {new Date(history[0].createdAt).toLocaleString(undefined, {
                dateStyle: "short",
                timeStyle: "short",
              })}
            </div>
          )}
          {history.length > 0 && (
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 cursor-default">
              {/* Latency Card */}
              <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900">
                <div className="flex items-center gap-3 mb-4">
                  <IoIosPulse className="w-5 h-5 text-blue-400" />
                  <h3 className="text-gray-400 font-medium">Latency</h3>
                </div>
                <div className="text-white text-3xl font-bold">
                  {parseFloat(history[0].latency).toFixed(2)}
                  <span className="text-xl font-normal text-gray-400 ml-1">
                    ms
                  </span>
                </div>
              </div>

              {/* Download Card */}
              <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900">
                <div className="flex items-center gap-3 mb-4">
                  <ArrowDownIcon className="w-5 h-5 text-emerald-400" />
                  <h3 className="text-gray-400 font-medium">Download</h3>
                </div>
                <div className="text-white text-3xl font-bold">
                  {history[0].downloadSpeed.toFixed(2)}
                  <span className="text-xl font-normal text-gray-400 ml-1">
                    Mbps
                  </span>
                </div>
              </div>

              {/* Upload Card */}
              <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900">
                <div className="flex items-center gap-3 mb-4">
                  <ArrowUpIcon className="w-5 h-5 text-purple-400" />
                  <h3 className="text-gray-400 font-medium">Upload</h3>
                </div>
                <div className="text-white text-3xl font-bold">
                  {history[0].uploadSpeed.toFixed(2)}
                  <span className="text-xl font-normal text-gray-400 ml-1">
                    Mbps
                  </span>
                </div>
              </div>

              {/* Jitter Card */}
              <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900">
                <div className="flex items-center gap-3 mb-4">
                  <IoMdGitCompare className="w-5 h-5 text-purple-400" />
                  <h3 className="text-gray-400 font-medium">Jitter</h3>
                </div>
                <div className="text-white text-3xl font-bold">
                  {history[0].jitter?.toFixed(2) ?? "N/A"}
                  <span className="text-xl font-normal text-gray-400 ml-1">
                    ms
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Historical Data - Show only if there's history */}
        {history.length > 0 && (
          <div className="space-y-8">
            <SpeedHistoryChart
              history={history}
              timeRange={timeRange}
              onTimeRangeChange={setTimeRange}
            />
            {/* ... rest of the history table ... */}
          </div>
        )}

        {/* Server Selection - Primary Action Area */}
        <div className="bg-gray-850/95 p-4 rounded-xl shadow-lg mb-6 border border-gray-900">
          <div
            className="flex justify-between items-center cursor-pointer"
            onClick={() => setIsConfigExpanded(!isConfigExpanded)}
          >
            <div className="flex flex-col">
              <h2 className="text-white text-xl font-semibold p-1">
                Server Selection
              </h2>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-gray-400">
                {selectedServers.length} server
                {selectedServers.length !== 1 ? "s" : ""} selected
              </span>
              <svg
                className={`w-5 h-5 text-gray-400 transform transition-transform ${
                  isConfigExpanded ? "rotate-180" : ""
                }`}
                fill="none"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path d="M19 9l-7 7-7-7"></path>
              </svg>
            </div>
          </div>

          <div
            className={`space-y-4 transition-all duration-200 ${
              isConfigExpanded ? "opacity-100" : "opacity-0 h-0 overflow-hidden"
            }`}
          >
            <p className="text-gray-400 text-sm p-1">
              Select a server for either a manual run or to set up a scheduled
              test
            </p>

            {/* Multi-Server Toggle */}
            <div className="flex items-center justify-end gap-2 pt-4">
              <label className="flex items-center gap-2 text-gray-300 text-sm">
                <Switch
                  checked={options.multiServer}
                  onChange={(checked) =>
                    setOptions((prev) => ({
                      ...prev,
                      multiServer: checked,
                    }))
                  }
                  className={`${
                    options.multiServer ? "bg-blue-600" : "bg-gray-700"
                  } relative inline-flex items-center h-6 rounded-full w-11`}
                >
                  <span className="sr-only">Multi-Server Mode</span>
                  <span
                    className={`${
                      options.multiServer ? "translate-x-6" : "translate-x-1"
                    } inline-block w-4 h-4 transform bg-white rounded-full transition`}
                  />
                </Switch>
                Multi-Server Mode
              </label>
            </div>

            {/* Server Search */}
            <div className="flex gap-3 items-center">
              <input
                type="text"
                placeholder="Search test servers..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="flex-1 text-sm border border-gray-900 bg-gray-800/50 text-white rounded-lg px-3 py-2 placeholder-gray-500"
              />
            </div>

            {/* Server List */}
            <ServerList
              servers={filteredServers}
              selectedServers={selectedServers}
              onSelect={handleServerSelect}
            />

            {/* Test Options */}
            <div className="pt-4 border-t border-gray-900">
              <div className="flex justify-between items-center">
                <div className="flex gap-6 text-gray-300">
                  <label className="flex items-center gap-2 text-sm">
                    <Switch
                      checked={options.enableDownload}
                      onChange={(checked) =>
                        setOptions((prev) => ({
                          ...prev,
                          enableDownload: checked,
                        }))
                      }
                      className={`${
                        options.enableDownload ? "bg-blue-600" : "bg-gray-700"
                      } relative inline-flex h-6 w-11 items-center rounded-full transition-colors`}
                    >
                      <span className="sr-only">Enable Download Test</span>
                      <span
                        className={`${
                          options.enableDownload
                            ? "translate-x-6"
                            : "translate-x-1"
                        } inline-block h-4 w-4 transform rounded-full bg-white transition`}
                      />
                    </Switch>
                    Download Test
                  </label>

                  <label className="flex items-center gap-2 text-sm">
                    <Switch
                      checked={options.enableUpload}
                      onChange={(checked) =>
                        setOptions((prev) => ({
                          ...prev,
                          enableUpload: checked,
                        }))
                      }
                      className={`${
                        options.enableUpload ? "bg-blue-600" : "bg-gray-700"
                      } relative inline-flex h-6 w-11 items-center rounded-full transition-colors`}
                    >
                      <span className="sr-only">Enable Upload Test</span>
                      <span
                        className={`${
                          options.enableUpload
                            ? "translate-x-6"
                            : "translate-x-1"
                        } inline-block h-4 w-4 transform rounded-full bg-white transition`}
                      />
                    </Switch>
                    Upload Test
                  </label>

                  <label className="flex items-center gap-2 text-sm">
                    <Switch
                      checked={options.enablePacketLoss}
                      onChange={(checked) =>
                        setOptions((prev) => ({
                          ...prev,
                          enablePacketLoss: checked,
                        }))
                      }
                      className={`${
                        options.enablePacketLoss ? "bg-blue-600" : "bg-gray-700"
                      } relative inline-flex h-6 w-11 items-center rounded-full transition-colors`}
                    >
                      <span className="sr-only">
                        Enable Packet Loss Analysis
                      </span>
                      <span
                        className={`${
                          options.enablePacketLoss
                            ? "translate-x-6"
                            : "translate-x-1"
                        } inline-block h-4 w-4 transform rounded-full bg-white transition`}
                      />
                    </Switch>
                    Packet Loss Analysis
                  </label>
                </div>

                <button
                  onClick={runTest}
                  disabled={loading || selectedServers.length === 0}
                  className="bg-blue-500 hover:bg-blue-600 text-white px-6 py-2 rounded-lg disabled:opacity-50"
                >
                  {loading
                    ? "Running Test..."
                    : selectedServers.length === 0
                    ? "Select a server"
                    : "Run Test"}
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Add ScheduleManager after the server selection */}
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
