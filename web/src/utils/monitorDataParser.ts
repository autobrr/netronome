/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import type { MonitorNativeData, MonitorPeriod, MonitorUsageSummary } from "@/api/monitor";

const EMPTY_SUMMARY: MonitorUsageSummary = { download: 0, upload: 0, total: 0 };

function toSummary(rx: number, tx: number): MonitorUsageSummary {
  return { download: rx, upload: tx, total: rx + tx };
}

function getEmptyUsage(): Record<string, MonitorUsageSummary> {
  return {
    "This Hour": EMPTY_SUMMARY,
    "Last Hour": EMPTY_SUMMARY,
    Today: EMPTY_SUMMARY,
    "This week": EMPTY_SUMMARY,
    "This Month": EMPTY_SUMMARY,
    "All Time": EMPTY_SUMMARY,
  };
}

function resolveAgentTime(data: MonitorNativeData): Date {
  if (data.server_time) {
    return new Date(data.server_time);
  }
  if (data.server_time_unix) {
    return new Date(data.server_time_unix * 1000);
  }
  return new Date();
}

/**
 * Parse vnstat native JSON data into usage periods.
 * Directly uses vnstat's own period calculations rather than database aggregations.
 */
export function parseMonitorUsagePeriods(
  data: MonitorNativeData,
): Record<string, MonitorUsageSummary> {
  if (!data.interfaces || data.interfaces.length === 0) {
    return getEmptyUsage();
  }

  const traffic = data.interfaces[0].traffic;

  if (!traffic.hour && !traffic.day && !traffic.month) {
    return {
      ...getEmptyUsage(),
      "All Time": traffic.total ? toSummary(traffic.total.rx, traffic.total.tx) : EMPTY_SUMMARY,
    };
  }

  const agentNow = resolveAgentTime(data);
  const isUTC = data.timezone_offset === 0;

  return {
    "This Hour": traffic.hour ? getCurrentHour(traffic.hour, agentNow, isUTC) : EMPTY_SUMMARY,
    "Last Hour": traffic.hour ? getLastHour(traffic.hour, agentNow, isUTC) : EMPTY_SUMMARY,
    Today: traffic.day ? getToday(traffic.day, agentNow, isUTC) : EMPTY_SUMMARY,
    "This week": traffic.day ? getThisWeek(traffic.day, agentNow, isUTC) : EMPTY_SUMMARY,
    "This Month": traffic.month ? getCurrentMonth(traffic.month, agentNow, isUTC) : EMPTY_SUMMARY,
    "All Time": traffic.total ? toSummary(traffic.total.rx, traffic.total.tx) : EMPTY_SUMMARY,
  };
}

interface DateComponents {
  year: number;
  month: number; // 1-based (matches vnstat format)
  day: number;
  hour: number;
}

function getDateComponents(now: Date, isUTC: boolean): DateComponents {
  if (isUTC) {
    return {
      year: now.getUTCFullYear(),
      month: now.getUTCMonth() + 1,
      day: now.getUTCDate(),
      hour: now.getUTCHours(),
    };
  }
  return {
    year: now.getFullYear(),
    month: now.getMonth() + 1,
    day: now.getDate(),
    hour: now.getHours(),
  };
}

function getCurrentHour(
  hours: MonitorPeriod[],
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const { year, month, day, hour } = getDateComponents(now, isUTC);

  const entry = hours.find(
    (h) =>
      h.date.year === year &&
      h.date.month === month &&
      h.date.day === day &&
      h.time?.hour === hour,
  );

  return entry ? toSummary(entry.rx, entry.tx) : EMPTY_SUMMARY;
}

function getLastHour(
  hours: MonitorPeriod[],
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  let { year, month, day, hour } = getDateComponents(now, isUTC);
  hour -= 1;

  if (hour < 0) {
    hour = 23;
    day -= 1;

    if (day <= 0) {
      month -= 1;
      if (month <= 0) {
        month = 12;
        year -= 1;
      }
      day = new Date(year, month, 0).getDate();
    }
  }

  const entry = hours.find(
    (h) =>
      h.date.year === year &&
      h.date.month === month &&
      h.date.day === day &&
      h.time?.hour === hour,
  );

  return entry ? toSummary(entry.rx, entry.tx) : EMPTY_SUMMARY;
}

function getToday(
  days: MonitorPeriod[],
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const { year, month, day } = getDateComponents(now, isUTC);

  const entry = days.find(
    (d) =>
      d.date.year === year &&
      d.date.month === month &&
      d.date.day === day,
  );

  return entry ? toSummary(entry.rx, entry.tx) : EMPTY_SUMMARY;
}

function getStartOfWeekMs(now: Date, daysFromMonday: number, isUTC: boolean): number {
  const startOfWeek = new Date(now);
  if (isUTC) {
    startOfWeek.setUTCDate(startOfWeek.getUTCDate() - daysFromMonday);
    startOfWeek.setUTCHours(0, 0, 0, 0);
  } else {
    startOfWeek.setDate(startOfWeek.getDate() - daysFromMonday);
    startOfWeek.setHours(0, 0, 0, 0);
  }
  return startOfWeek.getTime();
}

function periodToMs(period: MonitorPeriod, isUTC: boolean): number {
  const { year, month, day } = period.date;
  if (isUTC) {
    return Date.UTC(year, month - 1, day || 1);
  }
  return new Date(year, month - 1, day || 1).getTime();
}

function getThisWeek(
  days: MonitorPeriod[],
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const dayOfWeek = isUTC ? now.getUTCDay() : now.getDay();
  const daysFromMonday = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
  const startOfWeekMs = getStartOfWeekMs(now, daysFromMonday, isUTC);
  const nowMs = now.getTime();

  let totalRx = 0;
  let totalTx = 0;

  for (const day of days) {
    const dayMs = periodToMs(day, isUTC);
    if (dayMs >= startOfWeekMs && dayMs <= nowMs) {
      totalRx += day.rx;
      totalTx += day.tx;
    }
  }

  return toSummary(totalRx, totalTx);
}

function getCurrentMonth(
  months: MonitorPeriod[],
  now: Date,
  isUTC: boolean = false,
): MonitorUsageSummary {
  const { year, month } = getDateComponents(now, isUTC);

  const entry = months.find(
    (m) => m.date.year === year && m.date.month === month,
  );

  return entry ? toSummary(entry.rx, entry.tx) : EMPTY_SUMMARY;
}
