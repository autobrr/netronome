/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Local midnight at the start of a YYYY-MM-DD day, plus a number of days, as
 * an RFC 3339 timestamp. It returns null when the server cannot read the day:
 * Go reads only four-digit years.
 */
function localMidnight(day: string, addDays: number): string | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) return null;
  const [year, month, date] = day.split("-").map(Number);
  const midnight = new Date(0);
  // setFullYear keeps years below 100, which the Date constructor maps to 19xx.
  midnight.setFullYear(year, month - 1, date + addDays);
  midnight.setHours(0, 0, 0, 0);
  if (Number.isNaN(midnight.getTime())) return null;
  const iso = midnight.toISOString();
  return /^\d{4}-/.test(iso) ? iso : null;
}

/**
 * Builds the query of an export URL. An empty date leaves the range open on
 * that side. The end day is included, so "to" is the start of the next day.
 * It returns null when a date is not valid.
 */
export function exportQuery(format: string, fromDay: string, toDay: string): string | null {
  const params = new URLSearchParams({ format });
  for (const [name, day, addDays] of [["from", fromDay, 0], ["to", toDay, 1]] as const) {
    if (!day) continue;
    const timestamp = localMidnight(day, addDays);
    if (!timestamp) return null;
    params.set(name, timestamp);
  }
  return params.toString();
}
