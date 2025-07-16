/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo } from "react";
import { MonitorAgent } from "@/api/monitor";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { MonitorBandwidthChart } from "../MonitorBandwidthChart";

interface MonitorBandwidthTabProps {
  agent: MonitorAgent;
}

export const MonitorBandwidthTab: React.FC<MonitorBandwidthTabProps> = ({ agent }) => {
  const { nativeData } = useMonitorAgent({
    agent,
    includeNativeData: true,
  });

  // Process hourly data (last 48 hours)
  const hourlyChartData = useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.hour) {
      return [];
    }

    return nativeData.interfaces[0].traffic.hour
      .slice(0, 48)
      .reverse()
      .map((hour) => ({
        time: new Date(
          hour.date.year,
          hour.date.month - 1,
          hour.date.day || 1,
          hour.time?.hour || 0
        ).toISOString(),
        rx: hour.rx,
        tx: hour.tx,
      }));
  }, [nativeData]);

  // Process daily data (last 30 days)
  const dailyChartData = useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.day) {
      return [];
    }

    return nativeData.interfaces[0].traffic.day
      .slice(0, 30)
      .reverse()
      .map((day) => ({
        time: new Date(
          day.date.year,
          day.date.month - 1,
          day.date.day
        ).toISOString(),
        rx: day.rx,
        tx: day.tx,
      }));
  }, [nativeData]);

  // Process monthly data (last 12 months)
  const monthlyChartData = useMemo(() => {
    if (!nativeData?.interfaces?.[0]?.traffic?.month) {
      return [];
    }

    return nativeData.interfaces[0].traffic.month
      .slice(0, 12)
      .reverse()
      .map((month) => ({
        time: new Date(
          month.date.year,
          month.date.month - 1,
          1
        ).toISOString(),
        rx: month.rx,
        tx: month.tx,
      }));
  }, [nativeData]);

  if (!nativeData) {
    return (
      <div className="text-center py-12">
        <p className="text-lg text-gray-500 dark:text-gray-400">Loading bandwidth data...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Hourly Chart */}
      {hourlyChartData.length > 0 && (
        <MonitorBandwidthChart
          data={hourlyChartData}
          title="Hourly Bandwidth (48 hours)"
          timeFormat="hour"
        />
      )}

      {/* Daily Chart */}
      {dailyChartData.length > 0 && (
        <MonitorBandwidthChart
          data={dailyChartData}
          title="Daily Bandwidth (30 days)"
          timeFormat="day"
        />
      )}

      {/* Monthly Chart */}
      {monthlyChartData.length > 0 && (
        <MonitorBandwidthChart
          data={monthlyChartData}
          title="Monthly Bandwidth (12 months)"
          timeFormat="month"
        />
      )}

      {/* No data message */}
      {hourlyChartData.length === 0 && 
       dailyChartData.length === 0 && 
       monthlyChartData.length === 0 && (
        <div className="text-center py-12 bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800">
          <p className="text-lg text-gray-500 dark:text-gray-400">No bandwidth data available</p>
          <p className="text-sm text-gray-400 dark:text-gray-500 mt-2">
            The agent may have just been added or monitor hasn't collected enough data yet.
          </p>
        </div>
      )}
    </div>
  );
};