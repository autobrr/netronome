/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { VnstatNativeData, VnstatUsageSummary } from "@/api/vnstat";

/**
 * Parse vnstat native JSON data into usage periods
 * This directly uses vnstat's own period calculations rather than our database aggregations
 */
export function parseVnstatUsagePeriods(
  data: VnstatNativeData,
): Record<string, VnstatUsageSummary> {
  if (!data.interfaces || data.interfaces.length === 0) {
    return getEmptyUsage();
  }

  const traffic = data.interfaces[0].traffic;

  // Use server time if available for accurate timezone handling
  let agentNow: Date;
  if (data.server_time) {
    agentNow = new Date(data.server_time);
  } else if (data.server_time_unix) {
    agentNow = new Date(data.server_time_unix * 1000);
  } else {
    // Fallback to local time
    agentNow = new Date();
  }

  return {
    "This Hour": getCurrentHour(traffic.hour, agentNow),
    "Last Hour": getLastHour(traffic.hour, agentNow),
    Today: getToday(traffic.day, agentNow),
    "This Month": getCurrentMonth(traffic.month, agentNow),
    "All Time": {
      download: traffic.total.rx,
      upload: traffic.total.tx,
      total: traffic.total.rx + traffic.total.tx,
    },
  };
}

function getEmptyUsage(): Record<string, VnstatUsageSummary> {
  const empty = { download: 0, upload: 0, total: 0 };
  return {
    "This Hour": empty,
    "Last Hour": empty,
    Today: empty,
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
): VnstatUsageSummary {
  const currentHour = now.getHours();
  const currentDay = now.getDate();
  const currentMonth = now.getMonth() + 1; // vnstat months are 1-based
  const currentYear = now.getFullYear();

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
): VnstatUsageSummary {
  let lastHour = now.getHours() - 1;
  let targetDay = now.getDate();
  let targetMonth = now.getMonth() + 1;
  let targetYear = now.getFullYear();

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
): VnstatUsageSummary {
  const currentDay = now.getDate();
  const currentMonth = now.getMonth() + 1;
  const currentYear = now.getFullYear();

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

function getCurrentMonth(
  months: Array<{
    date: { year: number; month: number };
    rx: number;
    tx: number;
  }>,
  now: Date,
): VnstatUsageSummary {
  const currentMonth = now.getMonth() + 1;
  const currentYear = now.getFullYear();

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
