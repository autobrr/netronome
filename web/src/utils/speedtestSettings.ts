/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import type { Schedule, Server, SpeedTestResult } from "../types/types.ts";

/** Selects the geographic source used to build the Speedtest.net server list. */
export type SpeedtestServerSource = "local" | "global" | "coordinates";

/** Browser-persisted Speedtest.net server discovery and display preferences. */
export type SpeedtestSettings =
  | { source: "local"; showServerCity: boolean }
  | { source: "global"; showServerCity: boolean }
  | {
      source: "coordinates";
      latitude: number;
      longitude: number;
      showServerCity: boolean;
    };

/** Editable settings state, which may contain incomplete coordinates before validation. */
export interface SpeedtestSettingsDraft {
  source: SpeedtestServerSource;
  latitude?: number;
  longitude?: number;
  showServerCity: boolean;
}

/** Query parameters accepted by the Speedtest.net server-list API. */
export interface SpeedtestServerQuery {
  global?: boolean;
  latitude?: number;
  longitude?: number;
  /** Bypasses a valid server-side catalogue cache when true. */
  refresh?: boolean;
}

/** Identifies all origin-specific views of the retained Speedtest.net catalogue. */
export const speedtestServerCatalogueQueryKey = () => ["servers", "speedtest", "catalogue"] as const;

/** Identifies the retained Speedtest.net catalogue as ordered from one discovery origin. */
export const speedtestServerQueryKey = (query: SpeedtestServerQuery) =>
  [...speedtestServerCatalogueQueryKey(), query] as const;

/** Returns the serialized query identity used to scope Speedtest.net selections. */
export const speedtestServerSelectionKey = (query: SpeedtestServerQuery): string =>
  JSON.stringify(speedtestServerQueryKey(query));

/** Identifies durable fetch status for one Speedtest.net discovery source. */
export const speedtestServerStatusQueryKey = (settings: SpeedtestSettings) =>
  settings.source === "coordinates"
    ? ["servers", "speedtest", "status", settings.source, settings.latitude, settings.longitude] as const
    : ["servers", "speedtest", "status", settings.source] as const;

/** Formats the selected source's durable storage state without treating errors as absence. */
export const formatSpeedtestServerStorageStatus = (
  sourceLabel: string,
  status: { stored?: boolean; isLoading: boolean; isError: boolean },
): string => {
  if (status.isLoading) return "Checking stored server status…";
  if (status.isError) return "Stored server status unavailable";
  return status.stored
    ? `${sourceLabel} servers are stored`
    : `No ${sourceLabel.toLowerCase()} servers stored yet`;
};

const SETTINGS_KEY = "netronome-speedtest-settings";
const SETTINGS_EVENT = "speedtestSettingsChanged";
const DEFAULT_SETTINGS: SpeedtestSettings = {
  source: "local",
  showServerCity: false,
};

/** Reports whether a value is a finite latitude accepted by server discovery. */
export const isLatitude = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value) && value >= -90 && value <= 90;

/** Reports whether a value is a finite longitude accepted by server discovery. */
export const isLongitude = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value) && value >= -180 && value <= 180;

/** Normalizes untrusted persisted settings, falling back to local discovery when coordinates are invalid. */
export const normalizeSpeedtestSettings = (value: unknown): SpeedtestSettings => {
  if (typeof value !== "object" || value === null) {
    return { ...DEFAULT_SETTINGS };
  }

  const saved = value as Record<string, unknown>;
  const showServerCity = saved.showServerCity === true;
  if (saved.source === "global") {
    return { source: "global", showServerCity };
  }
  if (
    saved.source === "coordinates" &&
    isLatitude(saved.latitude) &&
    isLongitude(saved.longitude)
  ) {
    return {
      source: "coordinates",
      latitude: saved.latitude,
      longitude: saved.longitude,
      showServerCity,
    };
  }
  return { source: "local", showServerCity };
};

/** Reads Speedtest.net preferences from local storage. */
export const getSpeedtestSettings = (): SpeedtestSettings => {
  if (typeof window === "undefined") {
    return { ...DEFAULT_SETTINGS };
  }
  try {
    const saved = window.localStorage.getItem(SETTINGS_KEY);
    return saved ? normalizeSpeedtestSettings(JSON.parse(saved)) : { ...DEFAULT_SETTINGS };
  } catch (error) {
    console.warn("Failed to load speedtest settings from localStorage:", error);
    return { ...DEFAULT_SETTINGS };
  }
};

/** Persists Speedtest.net preferences, notifies active views, and reports whether storage succeeded. */
export const saveSpeedtestSettings = (settings: SpeedtestSettingsDraft): boolean => {
  try {
    const normalized = normalizeSpeedtestSettings(settings);
    window.localStorage.setItem(SETTINGS_KEY, JSON.stringify(normalized));
    window.dispatchEvent(new CustomEvent<SpeedtestSettings>(SETTINGS_EVENT, { detail: normalized }));
    return true;
  } catch (error) {
    console.warn("Failed to save speedtest settings to localStorage:", error);
    return false;
  }
};

/** Subscribes a component to browser-persisted Speedtest.net preferences. */
export const useSpeedtestSettings = (): SpeedtestSettings => {
  const [settings, setSettings] = useState<SpeedtestSettings>(getSpeedtestSettings);

  useEffect(() => {
    const handleChange = (event: Event) => {
      setSettings((event as CustomEvent<SpeedtestSettings>).detail);
    };
    window.addEventListener(SETTINGS_EVENT, handleChange);
    return () => window.removeEventListener(SETTINGS_EVENT, handleChange);
  }, []);

  return settings;
};

/** Converts persisted discovery preferences into server-list request parameters. */
export const speedtestServerQuery = (settings: SpeedtestSettings): SpeedtestServerQuery => {
  if (settings.source === "global") {
    return { global: true };
  }
  if (settings.source === "coordinates") {
    return { latitude: settings.latitude, longitude: settings.longitude };
  }
  return {};
};

/** Formats a result's provider name with its recorded city when enabled and available. */
export const formatSpeedtestServerName = (
  serverName: string,
  serverCity: string | null | undefined,
  showServerCity: boolean,
): string => {
  const city = serverCity?.trim();
  return showServerCity && city ? `${serverName} (${city})` : serverName;
};

/** Returns the provider-qualified identity used to filter and group historical server results. */
export const speedtestResultServerKey = (
  result: Pick<SpeedTestResult, "testType" | "serverId" | "serverHost" | "serverName">,
): string => `${result.testType}:${result.serverId || result.serverHost || result.serverName}`;

/** Keeps all-time existence separate from the latest result in the selected time range. */
export const summarizeSpeedtestHistory = (
  currentRange: SpeedTestResult[],
  allTime: SpeedTestResult[],
): { hasAnyTests: boolean; latestTest: SpeedTestResult | null } => ({
  hasAnyTests: allTime.length > 0,
  latestTest: currentRange[0] ?? null,
});

/** Finds a saved schedule's server without crossing provider or LibreSpeed source boundaries. */
export const findScheduleServer = (
  servers: Server[],
  serverID: string,
  options: Schedule["options"],
): Server | undefined => {
  const expectsLibrespeed = options.useLibrespeed === true;
  return servers.find((server) =>
    server.id === serverID &&
    Boolean(server.isLibrespeed) === expectsLibrespeed &&
    (!expectsLibrespeed || Boolean(server.isPublic) === (options.isPublicServer === true))
  );
};
