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

interface SpeedTestResult {
  id: number;
  serverName: string;
  serverId: string;
  downloadSpeed: number;
  uploadSpeed: number;
  latency: string;
  packetLoss: number;
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
      <h2 className="text-lg font-semibold mb-2 text-white">
        Test in Progress
      </h2>
      <div className="space-y-2 text-gray-300">
        <div>Testing Server: {progress.currentServer}</div>
        {progress.currentTest && (
          <div className="flex items-center gap-4">
            <div className="capitalize">{progress.currentTest} Test:</div>
            {progress.currentSpeed > 0 && (
              <div className="text-xl font-bold">
                {progress.currentSpeed.toFixed(2)} Mbps
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
      minute: "2-digit",
    }),
    download: result.downloadSpeed,
    upload: result.uploadSpeed,
    server: result.serverName,
  }));

  return (
    <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg mb-6 border border-gray-900">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-white">Speed History</h2>
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
              label={{
                value: "Speed (Mbps)",
                position: "insideLeft",
                angle: -90,
                offset: 0,
                className: "fill-gray-400",
              }}
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
              type="monotone"
              dataKey="download"
              stroke="#3B82F6"
              name="Download"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              className="!stroke-blue-500"
            />
            <Line
              type="monotone"
              dataKey="upload"
              stroke="#10B981"
              name="Upload"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              className="!stroke-emerald-500"
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
      setSelectedServers([server]);
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
          <div className="flex gap-4">
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
        {history.length > 0 && (
          <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg border border-gray-900 mb-6">
            <h2 className="text-xl font-semibold mb-4 text-white">
              Latest Results
            </h2>
            <div className="grid grid-cols-3 gap-4">
              <div className="text-center">
                <p className="text-gray-400">Latency</p>
                <p className="text-2xl font-bold text-white">
                  {history[0].latency}
                </p>
              </div>
              <div className="text-center">
                <p className="text-gray-400">Download</p>
                <p className="text-2xl font-bold text-white">
                  {history[0].downloadSpeed.toFixed(2)} Mbps
                </p>
              </div>
              <div className="text-center">
                <p className="text-gray-400">Upload</p>
                <p className="text-2xl font-bold text-white">
                  {history[0].uploadSpeed.toFixed(2)} Mbps
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Server Selection - Primary Action Area */}
        <div className="bg-gray-850/95 p-4 rounded-xl shadow-lg mb-6 border border-gray-900">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-semibold text-white">
              Test Configuration
            </h2>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={options.multiServer}
                onChange={(e) =>
                  setOptions((prev) => ({
                    ...prev,
                    multiServer: e.target.checked,
                  }))
                }
                className="form-checkbox bg-gray-700 border-gray-900"
              />
              <label className="text-gray-300">Multi-Server Mode</label>
            </div>
          </div>

          {/* Server Search */}
          <div className="flex gap-3 items-center mb-3">
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
          <div className="mt-4 pt-4 border-t border-gray-900">
            <div className="flex gap-6 text-gray-300">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.enableDownload}
                  onChange={(e) =>
                    setOptions((prev) => ({
                      ...prev,
                      enableDownload: e.target.checked,
                    }))
                  }
                  className="form-checkbox bg-gray-700 border-gray-900"
                />
                Download Test
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.enableUpload}
                  onChange={(e) =>
                    setOptions((prev) => ({
                      ...prev,
                      enableUpload: e.target.checked,
                    }))
                  }
                  className="form-checkbox bg-gray-700 border-gray-900"
                />
                Upload Test
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.enablePacketLoss}
                  onChange={(e) =>
                    setOptions((prev) => ({
                      ...prev,
                      enablePacketLoss: e.target.checked,
                    }))
                  }
                  className="form-checkbox bg-gray-700 border-gray-900"
                />
                Packet Loss Analysis
              </label>
            </div>
          </div>
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
