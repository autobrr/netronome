/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ServerIcon,
  ChartBarIcon,
  CpuChipIcon,
  ClockIcon,
  SignalIcon,
  CalendarIcon,
  CircleStackIcon,
  FireIcon,
} from "@heroicons/react/24/outline";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faLinux, faApple } from "@fortawesome/free-brands-svg-icons";
import { MonitorAgent, MonitorStatus } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { formatBytes } from "@/utils/formatBytes";
import { parseMonitorUsagePeriods } from "@/utils/monitorDataParser";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";

interface MonitorOverviewTabProps {
  agent: MonitorAgent;
}

const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
};

// Animation configurations moved outside component for performance
const CARD_ANIMATION = {
  initial: { opacity: 0, scale: 0.95 },
  animate: { opacity: 1, scale: 1 },
  transition: { duration: 0.3 },
} as const;

const SYSTEM_BAR_ANIMATION = {
  initial: { opacity: 0, y: -20 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.3 },
} as const;

const TEMPERATURE_ANIMATION = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.3, delay: 0.25 },
} as const;

// Sub-component for current speed display
interface CurrentSpeedCardProps {
  status?: MonitorStatus;
  isOffline: boolean;
  delay?: number;
}

const CurrentSpeedCard: React.FC<CurrentSpeedCardProps> = ({ status, isOffline, delay = 0 }) => (
  <motion.div
    {...CARD_ANIMATION}
    transition={{ ...CARD_ANIMATION.transition, delay }}
    className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
  >
    <div className="flex items-center justify-between mb-4">
      <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
        Current Speed
      </h3>
      <SignalIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
    </div>
    {status?.liveData ? (
      <div className="space-y-2">
        <div className="flex items-center space-x-2">
          <ArrowDownIcon className="h-4 w-4 text-blue-500" aria-hidden="true" />
          <span className="text-lg font-bold text-gray-900 dark:text-white tabular-nums">
            {status.liveData.rx.ratestring}
          </span>
        </div>
        <div className="flex items-center space-x-2">
          <ArrowUpIcon className="h-4 w-4 text-green-500" aria-hidden="true" />
          <span className="text-lg font-bold text-gray-900 dark:text-white tabular-nums">
            {status.liveData.tx.ratestring}
          </span>
        </div>
      </div>
    ) : (
      <p className="text-gray-500 dark:text-gray-400">
        {isOffline ? "Offline" : "No data"}
      </p>
    )}
  </motion.div>
);

// Sub-component for usage cards
interface UsageCardProps {
  title: string;
  icon: React.ReactNode;
  usage?: {
    total: number;
    download: number;
    upload: number;
  };
  delay?: number;
}

const UsageCard: React.FC<UsageCardProps> = ({ title, icon, usage, delay = 0 }) => (
  <motion.div
    {...CARD_ANIMATION}
    transition={{ ...CARD_ANIMATION.transition, delay }}
    className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
  >
    <div className="flex items-center justify-between mb-4">
      <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
        {title}
      </h3>
      {icon}
    </div>
    {usage ? (
      <>
        <p className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
          {formatBytes(usage.total)}
        </p>
        <div className="flex items-center space-x-4 text-sm">
          <div className="flex items-center space-x-1">
            <ArrowDownIcon className="h-3 w-3 text-blue-500" />
            <span className="text-gray-600 dark:text-gray-400">
              {formatBytes(usage.download)}
            </span>
          </div>
          <div className="flex items-center space-x-1">
            <ArrowUpIcon className="h-3 w-3 text-green-500" />
            <span className="text-gray-600 dark:text-gray-400">
              {formatBytes(usage.upload)}
            </span>
          </div>
        </div>
      </>
    ) : (
      <p className="text-gray-500 dark:text-gray-400">No data</p>
    )}
  </motion.div>
);

// Progress bar component for resource monitors
interface ResourceProgressBarProps {
  percentage: number;
  thresholds?: {
    low: number;
    medium: number;
  };
}

const ResourceProgressBar: React.FC<ResourceProgressBarProps> = ({
  percentage,
  thresholds = { low: 50, medium: 85 },
}) => {
  const getBarColor = () => {
    if (percentage < thresholds.low) return "#34d399"; // emerald-400
    if (percentage < thresholds.medium) return "#d97706"; // amber-600
    return "#EF4444"; // red-500
  };

  return (
    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
      <div
        className="h-3 rounded-full transition-all duration-500 ease-out"
        style={{
          width: `${Math.min(percentage, 100)}%`,
          backgroundColor: getBarColor(),
        }}
        role="progressbar"
        aria-valuenow={percentage}
        aria-valuemin={0}
        aria-valuemax={100}
      />
    </div>
  );
};

// Temperature alert component
interface TemperatureAlertProps {
  temperatures?: Array<{
    sensor_key: string;
    label?: string;
    temperature: number;
  }>;
}

const TemperatureAlert: React.FC<TemperatureAlertProps> = ({ temperatures }) => {
  if (!temperatures || temperatures.length === 0) return null;

  const hotSensors = temperatures.filter((t) => t.temperature > 80);
  const warmSensors = temperatures.filter((t) => t.temperature > 60 && t.temperature <= 80);

  if (hotSensors.length === 0 && warmSensors.length === 0) return null;

  const isHot = hotSensors.length > 0;
  const alertSensors = [...hotSensors, ...warmSensors];

  return (
    <motion.div
      {...TEMPERATURE_ANIMATION}
      className={`rounded-xl p-4 shadow-lg border ${
        isHot
          ? "bg-red-50/95 dark:bg-red-900/20 border-red-200 dark:border-red-800"
          : "bg-amber-50/95 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800"
      }`}
      role="alert"
      aria-live="polite"
    >
      <div className="flex items-center space-x-3">
        <FireIcon
          className={`h-6 w-6 ${
            isHot
              ? "text-red-600 dark:text-red-400"
              : "text-amber-600 dark:text-amber-400"
          }`}
          aria-hidden="true"
        />
        <div className="flex-1">
          <h4
            className={`font-medium ${
              isHot
                ? "text-red-900 dark:text-red-100"
                : "text-amber-900 dark:text-amber-100"
            }`}
          >
            Temperature Warning
          </h4>
          <div className="flex flex-wrap gap-3 mt-1">
            {alertSensors.map((sensor, idx) => (
              <span
                key={`${sensor.sensor_key}-${idx}`}
                className={`text-sm ${
                  sensor.temperature > 80
                    ? "text-red-700 dark:text-red-300"
                    : "text-amber-700 dark:text-amber-300"
                }`}
              >
                {sensor.label || sensor.sensor_key}: {sensor.temperature.toFixed(1)}°C
              </span>
            ))}
          </div>
        </div>
      </div>
    </motion.div>
  );
};

// System info details component for cleaner rendering
interface SystemInfoDetailsProps {
  cpu: {
    model?: string;
    cores: number;
    threads?: number;
  };
  kernel?: string;
}

const SystemInfoDetails: React.FC<SystemInfoDetailsProps> = ({ cpu, kernel }) => {
  const cpuModel = cpu.model
    ? cpu.model.replace(/\s+/g, " ").trim().split("@")[0].trim()
    : null;

  const getOSIcon = () => {
    if (!kernel) return null;
    
    if (kernel.toLowerCase().includes("darwin")) {
      return <FontAwesomeIcon icon={faApple} className="h-4 w-4 text-gray-500 dark:text-gray-500 ml-2 mr-1" />;
    }
    if (kernel.toLowerCase().includes("linux")) {
      return <FontAwesomeIcon icon={faLinux} className="h-4 w-4 text-gray-500 dark:text-gray-500 ml-2 mr-1" />;
    }
    return null;
  };

  return (
    <>
      {cpuModel && (
        <>
          {cpuModel}
          <span className="text-gray-500 dark:text-gray-500 mx-1">•</span>
        </>
      )}
      {cpu.cores} {cpu.cores === 1 ? "core" : "cores"}
      {cpu.threads && cpu.threads !== cpu.cores && (
        <>, {cpu.threads} threads</>
      )}
      {kernel && (
        <>
          {getOSIcon()}
          <span>{kernel}</span>
        </>
      )}
    </>
  );
};

// Resource monitor card component
interface ResourceMonitorCardProps {
  title: string;
  icon: React.ReactNode;
  percentage: number;
  primaryValue: string;
  secondaryValue: string;
  detailText: string;
  delay?: number;
  thresholds?: {
    low: number;
    medium: number;
  };
}

const ResourceMonitorCard: React.FC<ResourceMonitorCardProps> = ({
  title,
  icon,
  percentage,
  primaryValue,
  secondaryValue,
  detailText,
  delay = 0,
  thresholds,
}) => (
  <motion.div
    {...CARD_ANIMATION}
    transition={{ ...CARD_ANIMATION.transition, delay }}
    className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
  >
    <div className="flex items-center justify-between mb-4">
      <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
        {title}
      </h3>
      {icon}
    </div>
    <div className="space-y-3">
      <div className="flex items-baseline justify-between">
        <span className="text-3xl font-bold text-gray-900 dark:text-white tabular-nums">
          {primaryValue}
        </span>
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {secondaryValue}
        </span>
      </div>
      <ResourceProgressBar percentage={percentage} thresholds={thresholds} />
      <p className="text-xs text-gray-500 dark:text-gray-400">
        {detailText}
      </p>
    </div>
  </motion.div>
);

export const MonitorOverviewTab: React.FC<MonitorOverviewTabProps> = ({
  agent,
}) => {
  const { status, nativeData, hardwareStats, systemInfo } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeSystemInfo: true,
    includeHardwareStats: true,
    includePeakStats: false,
  });

  const usage = nativeData ? parseMonitorUsagePeriods(nativeData) : null;

  const isOffline = !status?.connected;

  return (
    <div className="space-y-6">
      {/* Offline Banner */}
      {isOffline && <MonitorOfflineBanner />}

      {/* System Identity Bar */}
      <motion.div
        {...SYSTEM_BAR_ANIMATION}
        className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-4 shadow-lg border border-gray-200 dark:border-gray-800"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            {/* Hostname Icon */}
            <ServerIcon className="h-8 w-8 text-gray-600 dark:text-gray-400" aria-hidden="true" />

            {/* System Info */}
            <div className="flex-1">
              <div className="flex items-baseline space-x-3">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {systemInfo?.hostname || agent.name}
                </h2>
              </div>
              {hardwareStats?.cpu && (
                <div className="flex items-center space-x-1.5 mt-0.5">
                  <CpuChipIcon className="h-4 w-4 text-gray-500 dark:text-gray-500" aria-hidden="true" />
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    <SystemInfoDetails
                      cpu={hardwareStats.cpu}
                      kernel={systemInfo?.kernel}
                    />
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Uptime */}
          <div className="flex items-center space-x-2 pr-4">
            <ClockIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {systemInfo?.uptime
                ? `Up ${formatUptime(systemInfo.uptime)}`
                : "N/A"}
            </span>
          </div>
        </div>
      </motion.div>

      {/* Network Performance Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Current Speed */}
        <CurrentSpeedCard
          status={status}
          isOffline={isOffline}
          delay={0.1}
        />

        {/* Today's Usage */}
        <UsageCard
          title="Today's Usage"
          icon={<ChartBarIcon className="h-5 w-5 text-gray-400" />}
          usage={usage?.["Today"]}
          delay={0.15}
        />

        {/* This Week Usage */}
        <UsageCard
          title="This Week"
          icon={<CalendarIcon className="h-5 w-5 text-gray-400" />}
          usage={usage?.["This week"]}
          delay={0.2}
        />

        {/* This Month Usage */}
        <UsageCard
          title="This Month"
          icon={<CalendarIcon className="h-5 w-5 text-gray-400" />}
          usage={usage?.["This Month"]}
          delay={0.25}
        />
      </div>

      {/* Resource Monitors Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* CPU Usage */}
        {hardwareStats?.cpu ? (
          <ResourceMonitorCard
            title="CPU Usage"
            icon={<CpuChipIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />}
            percentage={hardwareStats.cpu.usage_percent}
            primaryValue={`${hardwareStats.cpu.usage_percent.toFixed(1)}%`}
            secondaryValue={
              hardwareStats.cpu.load_avg?.[0]
                ? `Load: ${hardwareStats.cpu.load_avg[0].toFixed(2)}`
                : ""
            }
            detailText={`${hardwareStats.cpu.threads} threads @ ${hardwareStats.cpu.frequency?.toFixed(0) || "?"} MHz`}
            delay={0.3}
          />
        ) : (
          <motion.div
            {...CARD_ANIMATION}
            transition={{ ...CARD_ANIMATION.transition, delay: 0.3 }}
            className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
                CPU Usage
              </h3>
              <CpuChipIcon className="h-5 w-5 text-gray-400" />
            </div>
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          </motion.div>
        )}

        {/* Memory Usage */}
        {hardwareStats?.memory ? (
          <ResourceMonitorCard
            title="Memory Usage"
            icon={<CircleStackIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />}
            percentage={hardwareStats.memory.used_percent}
            primaryValue={`${hardwareStats.memory.used_percent.toFixed(1)}%`}
            secondaryValue={`${formatBytes(hardwareStats.memory.available)} free`}
            detailText={`${formatBytes(hardwareStats.memory.used)} / ${formatBytes(hardwareStats.memory.total)}`}
            delay={0.35}
          />
        ) : (
          <motion.div
            {...CARD_ANIMATION}
            transition={{ ...CARD_ANIMATION.transition, delay: 0.35 }}
            className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
                Memory Usage
              </h3>
              <CircleStackIcon className="h-5 w-5 text-gray-400" />
            </div>
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          </motion.div>
        )}

        {/* Disk Usage */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.4 }}
          className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800"
        >
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">
              Primary Disk
            </h3>
            <ServerIcon className="h-5 w-5 text-gray-400" />
          </div>
          {hardwareStats?.disks && hardwareStats.disks.length > 0 ? (
            (() => {
              const primaryDisk =
                hardwareStats.disks.find((d) => d.path === "/") ||
                hardwareStats.disks[0];
              return (
                <div className="space-y-3">
                  <div className="flex items-baseline justify-between">
                    <span className="text-3xl font-bold text-gray-900 dark:text-white">
                      {primaryDisk.used_percent.toFixed(1)}%
                    </span>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      {formatBytes(primaryDisk.free)} free
                    </span>
                  </div>
                  <ResourceProgressBar
                    percentage={primaryDisk.used_percent}
                    thresholds={{ low: 70, medium: 85 }}
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {primaryDisk.path} • {formatBytes(primaryDisk.used)} /{" "}
                    {formatBytes(primaryDisk.total)}
                  </p>
                </div>
              );
            })()
          ) : (
            <p className="text-gray-500 dark:text-gray-400">No data</p>
          )}
        </motion.div>
      </div>

      {/* Temperature Alert */}
      <TemperatureAlert temperatures={hardwareStats?.temperature} />
    </div>
  );
};
