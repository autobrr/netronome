/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import type { SpeedTestResult } from "../types/types.ts";

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
}

/** A server selection tied to the catalogue from which it was chosen. */
export interface KeyedServerSelection<T> {
  key: string;
  servers: T[];
}

/** A persisted server ID paired with current catalogue data when available. */
export interface ServerReference<T> {
  id: string;
  server?: T;
}

const SETTINGS_KEY = "netronome-speedtest-settings";
const SETTINGS_EVENT = "speedtestSettingsChanged";
const DEFAULT_SETTINGS: SpeedtestSettings = {
  source: "local",
  showServerCity: false,
};

const isLatitude = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value) && value >= -90 && value <= 90;

const isLongitude = (value: unknown): value is number =>
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

/** Returns a stable key for the Speedtest.net catalogue represented by the settings. */
export const speedtestSelectionKey = (settings: SpeedtestSettings): string => {
  if (settings.source === "coordinates") {
    return `speedtest:coordinates:${settings.latitude}:${settings.longitude}`;
  }
  return `speedtest:${settings.source}`;
};

/** Returns selected servers only while their originating catalogue is active. */
export const selectedServersForKey = <T>(
  selection: KeyedServerSelection<T>,
  activeKey: string,
): T[] => selection.key === activeKey ? selection.servers : [];

/** Preserves saved server IDs even when the active catalogue cannot resolve their details. */
export const resolveServerReferences = <T extends { id: string }>(
  serverIds: string[] | undefined,
  availableServers: T[],
  matchesProvider: (server: T) => boolean = () => true,
): Array<ServerReference<T>> => {
  return (serverIds ?? []).map((id) => {
    const server = availableServers.find(
      (candidate) => candidate.id === id && matchesProvider(candidate),
    );
    return server ? { id, server } : { id };
  });
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
