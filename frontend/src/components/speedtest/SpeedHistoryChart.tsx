import React, { useState } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { SpeedTestResult, TimeRange } from "../../types/types";

interface SpeedHistoryChartProps {
  history: SpeedTestResult[];
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
}

interface VisibleMetrics {
  download: boolean;
  upload: boolean;
  latency: boolean;
  jitter: boolean;
}

const timeRangeOptions: { value: TimeRange; label: string }[] = [
  { value: "1d", label: "24 Hours" },
  { value: "3d", label: "3 Days" },
  { value: "1w", label: "1 Week" },
  { value: "1m", label: "1 Month" },
  { value: "all", label: "All Time" },
];

export const SpeedHistoryChart: React.FC<SpeedHistoryChartProps> = ({
  history,
  timeRange,
  onTimeRangeChange,
}) => {
  const [visibleMetrics, setVisibleMetrics] = useState<VisibleMetrics>(() => {
    const saved = localStorage.getItem("speedtest-visible-metrics");
    return saved
      ? JSON.parse(saved)
      : {
          download: true,
          upload: true,
          latency: true,
          jitter: true,
        };
  });

  const handleMetricToggle = (key: keyof VisibleMetrics) => {
    setVisibleMetrics((prev: VisibleMetrics) => {
      const newState = {
        ...prev,
        [key]: !prev[key],
      };
      localStorage.setItem(
        "speedtest-visible-metrics",
        JSON.stringify(newState)
      );
      return newState;
    });
  };

  const getFilteredData = () => {
    const now = new Date();
    const timeRangeInMs: { [key in TimeRange]: number } = {
      "1d": 24 * 60 * 60 * 1000,
      "3d": 3 * 24 * 60 * 60 * 1000,
      "1w": 7 * 24 * 60 * 60 * 1000,
      "1m": 30 * 24 * 60 * 60 * 1000,
      all: 0,
    };

    return history
      .filter((item) => {
        if (timeRange === "all") return true;
        const cutoffTime = new Date(now.getTime() - timeRangeInMs[timeRange]);
        return new Date(item.createdAt) > cutoffTime;
      })
      .map((item) => ({
        timestamp: new Date(item.createdAt).toLocaleString(undefined, {
          month: "short",
          day: "numeric",
          hour: "numeric",
          minute: "numeric",
        }),
        download: item.downloadSpeed,
        upload: item.uploadSpeed,
        latency: parseFloat(item.latency.replace("ms", "")),
        jitter: item.jitter,
      }));
  };

  const filteredData = getFilteredData();

  const handleTimeRangeChange = (range: TimeRange) => {
    localStorage.setItem("speedtest-time-range", range);
    onTimeRangeChange(range);
  };

  return (
    <div className="bg-gray-850/95 p-6 rounded-xl shadow-lg mb-6 border border-gray-900">
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-4">
          <h2 className="text-white text-xl font-semibold">Speed History</h2>
          <div className="flex gap-2">
            {[
              { key: "download", color: "#3B82F6", label: "Download" },
              { key: "upload", color: "#10B981", label: "Upload" },
              { key: "latency", color: "#F59E0B", label: "Latency" },
              { key: "jitter", color: "#9333EA", label: "Jitter" },
            ].map(({ key, label, color }) => {
              const isActive =
                visibleMetrics[key as keyof typeof visibleMetrics];
              return (
                <button
                  key={key}
                  onClick={() =>
                    handleMetricToggle(key as keyof typeof visibleMetrics)
                  }
                  className={`
                    px-3 py-1.5 
                    rounded-md 
                    text-sm 
                    font-medium
                    flex items-center gap-2 
                    transition-all duration-150 ease-in-out
                    focus:outline-none
                    focus:ring-0
                    ${
                      isActive
                        ? `bg-opacity-20 border hover:bg-opacity-30`
                        : "bg-gray-800 text-gray-400 border border-gray-700 hover:bg-gray-750 hover:border-gray-600"
                    }
                  `}
                  style={{
                    backgroundColor: isActive ? `${color}20` : undefined,
                    borderColor: isActive ? color : undefined,
                    color: isActive ? color : undefined,
                  }}
                >
                  <div
                    className="w-2 h-2 rounded-full"
                    style={{ backgroundColor: isActive ? color : "#6B7280" }}
                  />
                  {label}
                </button>
              );
            })}
          </div>
        </div>
        <div className="flex gap-2">
          {timeRangeOptions.map((option) => (
            <button
              key={option.value}
              onClick={() => handleTimeRangeChange(option.value)}
              className={`px-3 py-1 rounded-lg text-sm transition-colors ${
                timeRange === option.value
                  ? "bg-blue-500 text-white"
                  : "bg-gray-800 text-gray-400 hover:bg-gray-700"
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="h-[400px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart
            data={filteredData}
            margin={{ top: 5, right: 30, left: 20, bottom: 25 }}
          >
            <defs>
              <linearGradient id="downloadGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#3B82F6" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#3B82F6" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="uploadGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#10B981" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#10B981" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#F59E0B" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#F59E0B" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="jitterGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#9333EA" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#9333EA" stopOpacity={0} />
              </linearGradient>
            </defs>
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
              contentStyle={{
                backgroundColor: "#1F2937",
                border: "1px solid #374151",
                borderRadius: "0.5rem",
              }}
              labelStyle={{ color: "#9CA3AF" }}
              itemStyle={{ color: "#E5E7EB" }}
            />
            {visibleMetrics.download && (
              <Area
                yAxisId="speed"
                type="monotone"
                dataKey="download"
                name="Download"
                stroke="#3B82F6"
                strokeWidth={3}
                dot={false}
                activeDot={{ r: 6 }}
                fill="url(#downloadGradient)"
                className="!stroke-blue-500"
              />
            )}
            {visibleMetrics.upload && (
              <Area
                yAxisId="speed"
                type="monotone"
                dataKey="upload"
                name="Upload"
                stroke="#10B981"
                strokeWidth={3}
                dot={false}
                activeDot={{ r: 6 }}
                fill="url(#uploadGradient)"
                className="!stroke-emerald-500"
              />
            )}
            {visibleMetrics.latency && (
              <Area
                yAxisId="latency"
                type="monotone"
                dataKey="latency"
                name="Latency"
                stroke="#F59E0B"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 6 }}
                fill="url(#latencyGradient)"
                className="!stroke-amber-500"
                strokeDasharray="3 3"
              />
            )}
            {visibleMetrics.jitter && (
              <Area
                yAxisId="latency"
                type="monotone"
                dataKey="jitter"
                name="Jitter"
                stroke="#9333EA"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 6 }}
                fill="url(#jitterGradient)"
                className="!stroke-purple-500"
                strokeDasharray="5 5"
              />
            )}
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
