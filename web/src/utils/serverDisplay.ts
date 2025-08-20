/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Server } from "@/types/types";

/**
 * Gets the current setting for showing city in server names
 */
export function getShowCityInServerName(): boolean {
  try {
    const saved = localStorage.getItem("netronome-show-city-in-server-name");
    return saved ? JSON.parse(saved) : false;
  } catch {
    return false;
  }
}

/**
 * Formats a server name based on current display settings
 * @param server - The server object
 * @returns Formatted server name with city if enabled
 */
export function formatServerName(server: Server): string {
  const showCity = getShowCityInServerName();
  const baseName = server.sponsor || server.name || "Unknown Server";
  
  if (showCity && server.city && server.city.trim() !== "" && 
      server.city !== baseName && server.city.toLowerCase() !== baseName.toLowerCase()) {
    return `${baseName} (${server.city})`;
  }
  
  return baseName;
}

/**
 * Formats a server name for display in results tables using stored city data
 * @param serverName - The server name from SpeedTestResult
 * @param serverCity - The server city from SpeedTestResult (stored when test was run)
 * @returns Formatted server name with city if available and enabled
 */
export function formatServerNameFromResult(serverName: string, serverCity?: string): string {
  const showCity = getShowCityInServerName();
  
  if (!showCity || !serverName) {
    return serverName || "Unknown Server";
  }

  if (serverCity && serverCity.trim() !== "" && 
      serverCity !== serverName && serverCity.toLowerCase() !== serverName.toLowerCase()) {
    return `${serverName} (${serverCity})`;
  }

  return serverName;
}

/**
 * Subscribe to city display setting changes
 */
export function subscribeToShowCitySetting(callback: (enabled: boolean) => void): () => void {
  const handleStorageChange = (event: StorageEvent) => {
    if (event.key === "netronome-show-city-in-server-name" && event.newValue !== null) {
      try {
        const newValue = JSON.parse(event.newValue);
        callback(newValue);
      } catch {
        callback(false);
      }
    }
  };

  // Handle storage events from other windows/tabs
  window.addEventListener('storage', handleStorageChange);
  
  // Also listen for custom events within the same window
  const handleCustomEvent = (event: CustomEvent<boolean>) => {
    callback(event.detail);
  };

  window.addEventListener('netronome-city-setting-changed', handleCustomEvent as EventListener);
  
  // Return cleanup function
  return () => {
    window.removeEventListener('storage', handleStorageChange);
    window.removeEventListener('netronome-city-setting-changed', handleCustomEvent as EventListener);
  };
}

/**
 * Sets the show city setting and notifies subscribers
 */
export function setShowCityInServerName(enabled: boolean): void {
  try {
    localStorage.setItem("netronome-show-city-in-server-name", JSON.stringify(enabled));
    
    // Dispatch custom event to notify subscribers in the same window
    const event = new CustomEvent('netronome-city-setting-changed', { detail: enabled });
    window.dispatchEvent(event);
  } catch (error) {
    console.error("Failed to save city display setting:", error);
  }
}