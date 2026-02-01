/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { MonitorSystemInfo } from "../MonitorSystemInfo";
import { MonitorHardwareStats } from "../MonitorHardwareStats";
import { MonitorOfflineBanner } from "../MonitorOfflineBanner";
import { ServerIcon } from "@heroicons/react/24/outline";
import { formatters } from "@/utils/timeSettings";
import { useTranslation } from "react-i18next";

interface MonitorSystemTabProps {
  agent: MonitorAgent;
}

export const MonitorSystemTab: React.FC<MonitorSystemTabProps> = ({ agent }) => {
  const { t } = useTranslation();
  const { status, systemInfo, hardwareStats } = useMonitorAgent({
    agent,
    includeSystemInfo: true,
    includeHardwareStats: true,
  });

  const isOffline = !status?.connected;
  const isLoading = !systemInfo && !hardwareStats;

  if (isLoading) {
    return (
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3 }}
        className="text-center py-8 sm:py-12"
      >
        <p className="text-lg text-gray-500 dark:text-gray-400">{t('monitoring.loadingSystemInfo')}</p>
      </motion.div>
    );
  }

  const hasData = systemInfo || hardwareStats;
  if (!hasData) {
    return (
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.3 }}
        className="text-center py-8 sm:py-12"
      >
        <ServerIcon className="h-10 w-10 sm:h-12 sm:w-12 text-gray-400 mx-auto mb-3 sm:mb-4" />
        <p className="text-base sm:text-lg text-gray-500 dark:text-gray-400">{t('monitoring.noSystemInfo')}</p>
        <p className="text-xs sm:text-sm text-gray-400 dark:text-gray-500 mt-1.5 sm:mt-2">
          {t('monitoring.noSystemInfoDesc')}
        </p>
      </motion.div>
    );
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Offline Banner */}
      {isOffline && <MonitorOfflineBanner message={t('monitoring.cachedSystemInfo')} />}

      <div className="space-y-4 sm:space-y-6 lg:space-y-0 lg:grid lg:grid-cols-3 lg:gap-6">
        {/* Left Column - System Information */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="order-1"
        >
          {systemInfo && (
            <MonitorSystemInfo systemInfo={systemInfo} isOffline={isOffline} />
          )}
        </motion.div>
        
        {/* CPU - order 2 on mobile, middle column on desktop */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.15 }}
          className="order-2 lg:order-2 space-y-4 sm:space-y-6"
        >
          {hardwareStats && (
            <>
              <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="cpu" />
              <div className="hidden lg:block">
                {hardwareStats.temperature && hardwareStats.temperature.length > 0 && (
                  <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="temperature" />
                )}
              </div>
            </>
          )}
        </motion.div>
        
        {/* Memory - order 3 on mobile, right column on desktop */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.2 }}
          className="order-3 lg:order-3 space-y-4 sm:space-y-6"
        >
          {hardwareStats && (
            <>
              <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="memory" />
              <div className="hidden lg:block">
                {hardwareStats.disks.length > 0 && (
                  <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="disk" />
                )}
              </div>
            </>
          )}
        </motion.div>
        
        {/* Temperature Sensors - order 4 on mobile (shows only on mobile) */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.25 }}
          className="order-4 lg:hidden"
        >
          {hardwareStats && hardwareStats.temperature && hardwareStats.temperature.length > 0 && (
            <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="temperature" />
          )}
        </motion.div>
        
        {/* Disk Usage - order 5 on mobile (shows only on mobile) */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3, delay: 0.3 }}
          className="order-5 lg:hidden"
        >
          {hardwareStats && hardwareStats.disks.length > 0 && (
            <MonitorHardwareStats hardwareStats={hardwareStats} showOnly="disk" />
          )}
        </motion.div>
      </div>

      {/* Single timestamp at bottom */}
      {(systemInfo || hardwareStats) && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.35 }}
          className="text-center text-xs text-gray-500 dark:text-gray-400 mt-4 sm:mt-6"
        >
          {hardwareStats?.from_cache || systemInfo?.from_cache ? "Data collected" : "Last updated"}:{" "}
          {formatters.time(new Date((hardwareStats?.updated_at || systemInfo?.updated_at) || Date.now()))}
          {(hardwareStats?.from_cache || systemInfo?.from_cache) && " (cached)"}
        </motion.div>
      )}
    </div>
  );
};