/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { MonitorNativeData, MonitorUsageSummary } from "@/api/monitor";

/**
 * Parse vnstat native JSON data into usage periods
 * This directly uses vnstat's own period calculations rather than our database aggregations
 */
export function parseMonitorUsagePeriods(
  data: MonitorNativeData,
): Record<string, MonitorUsageSummary> {
  if (!data.interfaces || data.interfaces.length === 0) {
    return getEmptyUsage();
  }

  const traffic = data.interfaces[0].traffic;

  // Use server time if available for accurate timezone handling
  let agentNow: Date;
  if (data.server_time) {
    agentNow = new Date(data.server_time);
    console.log("Using server_time:", data.server_time, "Parsed as:", agentNow);
  } else if (data.server_time_unix) {
    agentNow = new Date(data.server_time_unix * 1000);
    console.log(
      "Using server_time_unix:",
      data.server_time_unix,
      "Parsed as:",
      agentNow,
    );
  } else {
    // Fallback to local time
    agentNow = new Date();
    console.log("No server time available, using local time:", agentNow);
  }

  // If timezone offset is 0 (UTC), use UTC methods
  const isUTC = data.timezone_offset === 0;
  const currentHour = isUTC ? agentNow.getUTCHours() : agentNow.getHours();
  const currentDay = isUTC ? agentNow.getUTCDate() : agentNow.getDate();
  const currentMonth = isUTC
    ? agentNow.getUTCMonth() + 1
    : agentNow.getMonth() + 1;

  console.log("Agent timezone offset:", data.timezone_offset, "isUTC:", isUTC);
  console.log(
    "Looking for hour:",
    currentHour,
    "day:",
    currentDay,
    "month:",
    currentMonth,
  );

  return {
    "This Hour": getCurrentHour(traffic.hour, agentNow, isUTC),
    "Last Hour": getLastHour(traffic.hour, agentNow, isUTC),
    Today: getToday(traffic.day, agentNow, isUTC),
    "This week": getThisWeek(traffic.day, agentNow, isUTC),
    "This Month": getCurrentMonth(traffic.month, agentNow, isUTC),
    "All Time": {
      download: traffic.total.rx,
      upload: traffic.total.tx,
      total: traffic.total.rx + traffic.total.tx,
    },
  };
}

function getEmptyUsage(): Record<string, MonitorUsageSummary> {
  const empty = { download: 0, upload: 0, total: 0 };
  return {
    "This Hour": empty,
    "Last Hour": empty,
    Today: empty,
    "This week": empty,
    "This Month": empty,
    "All Time": empty,
  };
}

function getCurrentHour(
  hours: Array<{
    date: { year: number; month: number; day?: number };
    time?: { hour: number; minute: number };
    rx: number;
    tx: number;
  }>,
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const currentHour = isUTC ? now.getUTCHours() : now.getHours();
  const currentDay = isUTC ? now.getUTCDate() : now.getDate();
  const currentMonth = (isUTC ? now.getUTCMonth() : now.getMonth()) + 1; // vnstat months are 1-based
  const currentYear = isUTC ? now.getUTCFullYear() : now.getFullYear();

  // Find the most recent hour entry that matches current time
  const hourEntry = hours.find(
    (h) =>
      h.date.year === currentYear &&
      h.date.month === currentMonth &&
      h.date.day === currentDay &&
      h.time?.hour === currentHour,
  );

  if (!hourEntry) {
    return { download: 0, upload: 0, total: 0 };
  }

  return {
    download: hourEntry.rx,
    upload: hourEntry.tx,
    total: hourEntry.rx + hourEntry.tx,
  };
}

function getLastHour(
  hours: Array<{
    date: { year: number; month: number; day?: number };
    time?: { hour: number; minute: number };
    rx: number;
    tx: number;
  }>,
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  let lastHour = (isUTC ? now.getUTCHours() : now.getHours()) - 1;
  let targetDay = isUTC ? now.getUTCDate() : now.getDate();
  let targetMonth = (isUTC ? now.getUTCMonth() : now.getMonth()) + 1;
  let targetYear = isUTC ? now.getUTCFullYear() : now.getFullYear();

  // Handle hour wraparound
  if (lastHour < 0) {
    lastHour = 23;
    targetDay -= 1;

    // Handle day wraparound
    if (targetDay <= 0) {
      targetMonth -= 1;

      // Handle month wraparound
      if (targetMonth <= 0) {
        targetMonth = 12;
        targetYear -= 1;
      }

      // Get last day of previous month (simplified)
      targetDay = new Date(targetYear, targetMonth, 0).getDate();
    }
  }

  const hourEntry = hours.find(
    (h) =>
      h.date.year === targetYear &&
      h.date.month === targetMonth &&
      h.date.day === targetDay &&
      h.time?.hour === lastHour,
  );

  if (!hourEntry) {
    return { download: 0, upload: 0, total: 0 };
  }

  return {
    download: hourEntry.rx,
    upload: hourEntry.tx,
    total: hourEntry.rx + hourEntry.tx,
  };
}

function getToday(
  days: Array<{
    date: { year: number; month: number; day?: number };
    rx: number;
    tx: number;
  }>,
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const currentDay = isUTC ? now.getUTCDate() : now.getDate();
  const currentMonth = (isUTC ? now.getUTCMonth() : now.getMonth()) + 1;
  const currentYear = isUTC ? now.getUTCFullYear() : now.getFullYear();

  const dayEntry = days.find(
    (d) =>
      d.date.year === currentYear &&
      d.date.month === currentMonth &&
      d.date.day === currentDay,
  );

  if (!dayEntry) {
    return { download: 0, upload: 0, total: 0 };
  }

  return {
    download: dayEntry.rx,
    upload: dayEntry.tx,
    total: dayEntry.rx + dayEntry.tx,
  };
}

function getThisWeek(
  days: Array<{
    date: { year: number; month: number; day?: number };
    rx: number;
    tx: number;
  }>,
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  // Get the current day of week (0 = Sunday, 6 = Saturday)
  const currentDayOfWeek = isUTC ? now.getUTCDay() : now.getDay();
  
  // Calculate start of week (Monday)
  const startOfWeek = new Date(now);
  const daysFromMonday = currentDayOfWeek === 0 ? 6 : currentDayOfWeek - 1;
  startOfWeek.setDate(startOfWeek.getDate() - daysFromMonday);
  startOfWeek.setHours(0, 0, 0, 0);

  let totalRx = 0;
  let totalTx = 0;

  // Sum up all days from start of week to now
  days.forEach((day) => {
    const dayDate = new Date(day.date.year, day.date.month - 1, day.date.day || 1);
    if (dayDate >= startOfWeek && dayDate <= now) {
      totalRx += day.rx;
      totalTx += day.tx;
    }
  });

  return {
    download: totalRx,
    upload: totalTx,
    total: totalRx + totalTx,
  };
}

function getCurrentMonth(
  months: Array<{
    date: { year: number; month: number };
    rx: number;
    tx: number;
  }>,
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const currentMonth = (isUTC ? now.getUTCMonth() : now.getMonth()) + 1;
  const currentYear = isUTC ? now.getUTCFullYear() : now.getFullYear();

  const monthEntry = months.find(
    (m) => m.date.year === currentYear && m.date.month === currentMonth,
  );

  if (!monthEntry) {
    return { download: 0, upload: 0, total: 0 };
  }

  return {
    download: monthEntry.rx,
    upload: monthEntry.tx,
    total: monthEntry.rx + monthEntry.tx,
  };
}
