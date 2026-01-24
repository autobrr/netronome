/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";

export type DistanceUnit = "km" | "mi";

export interface DistanceSettings {
  unit: DistanceUnit;
}

const DISTANCE_SETTINGS_KEY = "netronome-distance-settings";
const DEFAULT_SETTINGS: DistanceSettings = {
  unit: "km",
};

const KM_TO_MI = 0.621371;

export const getDistanceSettings = (): DistanceSettings => {
  try {
    const saved = localStorage.getItem(DISTANCE_SETTINGS_KEY);
    if (saved) {
      const settings = JSON.parse(saved);
      return {
        unit: settings.unit === "mi" ? "mi" : "km",
      };
    }
  } catch (error) {
    console.warn("Failed to load distance settings from localStorage:", error);
  }
  return DEFAULT_SETTINGS;
};

export const saveDistanceSettings = (settings: DistanceSettings): void => {
  try {
    localStorage.setItem(DISTANCE_SETTINGS_KEY, JSON.stringify(settings));
    window.dispatchEvent(new CustomEvent("distanceSettingsChanged", { detail: settings }));
  } catch (error) {
    console.warn("Failed to save distance settings to localStorage:", error);
  }
};

export const formatDistance = (distanceKm: number, settings?: DistanceSettings): string => {
  const currentSettings = settings || getDistanceSettings();
  if (currentSettings.unit === "mi") {
    return `${Math.floor(distanceKm * KM_TO_MI)} mi`;
  }
  return `${Math.floor(distanceKm)} km`;
};

export const useDistanceSettings = (): { settings: DistanceSettings; updateSettings: (newSettings: DistanceSettings) => void } => {
  const [settings, setSettings] = React.useState<DistanceSettings>(getDistanceSettings);

  React.useEffect(() => {
    const handleSettingsChange = (event: CustomEvent<DistanceSettings>) => {
      setSettings(event.detail);
    };

    window.addEventListener("distanceSettingsChanged", handleSettingsChange as EventListener);

    return () => {
      window.removeEventListener("distanceSettingsChanged", handleSettingsChange as EventListener);
    };
  }, []);

  const updateSettings = (newSettings: DistanceSettings) => {
    saveDistanceSettings(newSettings);
    setSettings(newSettings);
  };

  return { settings, updateSettings };
};
