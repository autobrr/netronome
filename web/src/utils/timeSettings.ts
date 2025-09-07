/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";

export interface TimeFormatSettings {
  timezone: string;
  use24HourFormat: boolean;
}

export interface TimezoneOption {
  value: string;
  label: string;
  offset: string;
}

// Major world timezones ordered by UTC offset (negative to positive)
export const TIMEZONE_OPTIONS: TimezoneOption[] = [
  // UTC-10
  { value: "Pacific/Honolulu", label: "Hawaii Time (HST)", offset: "UTC-10" },
  
  // UTC-9/-8
  { value: "America/Anchorage", label: "Alaska Time (AKST/AKDT)", offset: "UTC-9/-8" },
  
  // UTC-8/-7
  { value: "America/Los_Angeles", label: "Pacific Time (PST/PDT)", offset: "UTC-8/-7" },
  { value: "America/Vancouver", label: "Pacific Time Canada (PST/PDT)", offset: "UTC-8/-7" },
  
  // UTC-7/-6
  { value: "America/Denver", label: "Mountain Time (MST/MDT)", offset: "UTC-7/-6" },
  { value: "America/Phoenix", label: "Arizona Time (MST)", offset: "UTC-7" },
  
  // UTC-6/-5
  { value: "America/Chicago", label: "Central Time (CST/CDT)", offset: "UTC-6/-5" },
  { value: "America/Mexico_City", label: "Mexico Central Time (CST/CDT)", offset: "UTC-6/-5" },
  
  // UTC-5/-4
  { value: "America/New_York", label: "Eastern Time (EST/EDT)", offset: "UTC-5/-4" },
  { value: "America/Toronto", label: "Eastern Time Canada (EST/EDT)", offset: "UTC-5/-4" },
  
  // UTC-5
  { value: "America/Lima", label: "Peru Time (PET)", offset: "UTC-5" },
  { value: "America/Bogota", label: "Colombia Time (COT)", offset: "UTC-5" },
  
  // UTC-4/-3
  { value: "America/Santiago", label: "Chile Time (CLT/CLST)", offset: "UTC-4/-3" },
  
  // UTC-3/-2
  { value: "America/Sao_Paulo", label: "Brasília Time (BRT/BRST)", offset: "UTC-3/-2" },
  
  // UTC-3
  { value: "America/Buenos_Aires", label: "Argentina Time (ART)", offset: "UTC-3" },
  
  // UTC+0
  { value: "UTC", label: "Coordinated Universal Time (UTC)", offset: "UTC+0" },
  
  // UTC+0/+1
  { value: "Europe/London", label: "Greenwich Mean Time (GMT/BST)", offset: "UTC+0/+1" },
  
  // UTC+1/+2
  { value: "Europe/Berlin", label: "Central European Time (CET/CEST)", offset: "UTC+1/+2" },
  { value: "Europe/Paris", label: "France Time (CET/CEST)", offset: "UTC+1/+2" },
  { value: "Europe/Rome", label: "Italy Time (CET/CEST)", offset: "UTC+1/+2" },
  { value: "Europe/Madrid", label: "Spain Time (CET/CEST)", offset: "UTC+1/+2" },
  { value: "Europe/Amsterdam", label: "Netherlands Time (CET/CEST)", offset: "UTC+1/+2" },
  { value: "Europe/Stockholm", label: "Sweden Time (CET/CEST)", offset: "UTC+1/+2" },
  
  // UTC+2/+3
  { value: "Europe/Helsinki", label: "Finland Time (EET/EEST)", offset: "UTC+2/+3" },
  { value: "Europe/Athens", label: "Greece Time (EET/EEST)", offset: "UTC+2/+3" },
  
  // UTC+3
  { value: "Europe/Istanbul", label: "Turkey Time (TRT)", offset: "UTC+3" },
  { value: "Europe/Moscow", label: "Moscow Time (MSK)", offset: "UTC+3" },
  
  // UTC+4
  { value: "Asia/Dubai", label: "Gulf Standard Time (GST)", offset: "UTC+4" },
  
  // UTC+5
  { value: "Asia/Karachi", label: "Pakistan Time (PKT)", offset: "UTC+5" },
  
  // UTC+5:30
  { value: "Asia/Kolkata", label: "India Standard Time (IST)", offset: "UTC+5:30" },
  
  // UTC+6
  { value: "Asia/Dhaka", label: "Bangladesh Time (BST)", offset: "UTC+6" },
  
  // UTC+7
  { value: "Asia/Bangkok", label: "Indochina Time (ICT)", offset: "UTC+7" },
  
  // UTC+8
  { value: "Asia/Singapore", label: "Singapore Time (SGT)", offset: "UTC+8" },
  { value: "Asia/Shanghai", label: "China Time (CST)", offset: "UTC+8" },
  { value: "Asia/Hong_Kong", label: "Hong Kong Time (HKT)", offset: "UTC+8" },
  { value: "Australia/Perth", label: "Western Australia Time (AWST)", offset: "UTC+8" },
  
  // UTC+9
  { value: "Asia/Tokyo", label: "Japan Time (JST)", offset: "UTC+9" },
  { value: "Asia/Seoul", label: "Korea Time (KST)", offset: "UTC+9" },
  
  // UTC+9:30/+10:30
  { value: "Australia/Adelaide", label: "Central Australia Time (ACST/ACDT)", offset: "UTC+9:30/+10:30" },
  
  // UTC+10/+11
  { value: "Australia/Sydney", label: "Eastern Australia Time (AEST/AEDT)", offset: "UTC+10/+11" },
  
  // UTC+10
  { value: "Australia/Brisbane", label: "Queensland Time (AEST)", offset: "UTC+10" },
  
  // UTC+12/+13
  { value: "Pacific/Auckland", label: "New Zealand Time (NZST/NZDT)", offset: "UTC+12/+13" },
];

const TIME_SETTINGS_KEY = "netronome-time-settings";
const DEFAULT_SETTINGS: TimeFormatSettings = {
  timezone: "auto", // Special value for browser detection
  use24HourFormat: false,
};

/**
 * Get current time format settings from localStorage
 */
export const getTimeFormatSettings = (): TimeFormatSettings => {
  try {
    const saved = localStorage.getItem(TIME_SETTINGS_KEY);
    if (saved) {
      const settings = JSON.parse(saved);
      // Ensure all required properties exist
      return {
        timezone: settings.timezone || DEFAULT_SETTINGS.timezone,
        use24HourFormat: settings.use24HourFormat ?? DEFAULT_SETTINGS.use24HourFormat,
      };
    }
  } catch (error) {
    console.warn("Failed to load time settings from localStorage:", error);
  }
  return DEFAULT_SETTINGS;
};

/**
 * Save time format settings to localStorage
 */
export const saveTimeFormatSettings = (settings: TimeFormatSettings): void => {
  try {
    localStorage.setItem(TIME_SETTINGS_KEY, JSON.stringify(settings));
    // Dispatch custom event to notify other components
    window.dispatchEvent(new CustomEvent("timeSettingsChanged", { detail: settings }));
  } catch (error) {
    console.warn("Failed to save time settings to localStorage:", error);
  }
};

/**
 * Get the effective timezone for date formatting
 */
export const getEffectiveTimezone = (settings?: TimeFormatSettings): string | undefined => {
  const currentSettings = settings || getTimeFormatSettings();
  return currentSettings.timezone === "auto" ? undefined : currentSettings.timezone;
};

/**
 * Get current browser's detected timezone
 */
export const getBrowserTimezone = (): string => {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
};

/**
 * Get timezone display name
 */
export const getTimezoneDisplayName = (timezone: string): string => {
  if (timezone === "auto") {
    const browserTz = getBrowserTimezone();
    const option = TIMEZONE_OPTIONS.find(tz => tz.value === browserTz);
    return option ? `Auto (${option.label})` : `Auto (${browserTz})`;
  }
  
  const option = TIMEZONE_OPTIONS.find(tz => tz.value === timezone);
  return option ? option.label : timezone;
};

/**
 * Format a date using current time settings
 */
export const formatDateWithSettings = (
  date: Date | string,
  options: Intl.DateTimeFormatOptions = {},
  settings?: TimeFormatSettings
): string => {
  const currentSettings = settings || getTimeFormatSettings();
  const dateObj = typeof date === "string" ? new Date(date) : date;
  
  const formatOptions: Intl.DateTimeFormatOptions = {
    ...options,
    timeZone: getEffectiveTimezone(currentSettings),
  };

  // Apply 12/24 hour format preference for time display
  if (options.hour !== undefined || options.timeStyle !== undefined) {
    formatOptions.hour12 = !currentSettings.use24HourFormat;
  }

  return dateObj.toLocaleString(undefined, formatOptions);
};

/**
 * Format time specifically using current settings
 */
export const formatTimeWithSettings = (
  date: Date | string,
  settings?: TimeFormatSettings
): string => {
  return formatDateWithSettings(date, {
    hour: "numeric",
    minute: "2-digit"
  }, settings);
};

/**
 * Format date and time using current settings
 */
export const formatDateTimeWithSettings = (
  date: Date | string,
  settings?: TimeFormatSettings
): string => {
  return formatDateWithSettings(date, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit"
  }, settings);
};

/**
 * Get time format example based on settings
 */
export const getTimeFormatExample = (settings: TimeFormatSettings): string => {
  const now = new Date();
  return formatTimeWithSettings(now, settings);
};

/**
 * Hook to listen for time settings changes
 */
export const useTimeSettings = () => {
  const [settings, setSettings] = React.useState<TimeFormatSettings>(getTimeFormatSettings);

  React.useEffect(() => {
    const handleSettingsChange = (event: CustomEvent<TimeFormatSettings>) => {
      setSettings(event.detail);
    };

    window.addEventListener("timeSettingsChanged", handleSettingsChange as EventListener);
    
    return () => {
      window.removeEventListener("timeSettingsChanged", handleSettingsChange as EventListener);
    };
  }, []);

  const updateSettings = (newSettings: TimeFormatSettings) => {
    saveTimeFormatSettings(newSettings);
    setSettings(newSettings);
  };

  return { settings, updateSettings };
};

/**
 * Global function to format dates consistently across the app
 * This replaces the need for toLocaleString(undefined, options) calls
 */
export const formatDate = (
  date: Date | string,
  options: Intl.DateTimeFormatOptions = {}
): string => {
  return formatDateWithSettings(date, options);
};

/**
 * Specialized formatters for common use cases
 */
export const formatters = {
  time: (date: Date | string) => formatTimeWithSettings(date),
  dateTime: (date: Date | string) => formatDateTimeWithSettings(date),
  
  // Chart-specific formatters
  chartTick: (date: Date | string, timeRange: string, isMobile?: boolean) => {
    const settings = getTimeFormatSettings();
    const dateObj = typeof date === "string" ? new Date(date) : date;
    
    // Dynamic formatting based on time range and mobile context
    switch (timeRange) {
      case "1d":
        if (isMobile) {
          return formatDateWithSettings(dateObj, {
            hour: "numeric",
            minute: "2-digit",
          }, settings);
        }
        return formatDateWithSettings(dateObj, {
          weekday: "short",
          hour: "numeric",
          minute: "2-digit",
        }, settings);
        
      case "3d":
        if (isMobile) {
          return formatDateWithSettings(dateObj, {
            weekday: "short",
            hour: "numeric",
          }, settings);
        }
        return formatDateWithSettings(dateObj, {
          weekday: "short",
          hour: "numeric",
          minute: "2-digit",
        }, settings);
        
      case "1w":
        if (isMobile) {
          return formatDateWithSettings(dateObj, {
            month: "numeric",
            day: "numeric",
          }, settings);
        }
        return formatDateWithSettings(dateObj, {
          weekday: "short",
          month: "short",
          day: "numeric",
        }, settings);
        
      case "1m":
        return formatDateWithSettings(dateObj, {
          month: "short",
          day: "numeric",
        }, settings);
        
      case "all":
        const now = new Date();
        const showYear = dateObj.getFullYear() !== now.getFullYear();
        if (isMobile) {
          return formatDateWithSettings(dateObj, {
            month: "short",
            day: "numeric",
            ...(showYear && { year: "2-digit" }),
          }, settings);
        }
        return formatDateWithSettings(dateObj, {
          month: "short",
          day: "numeric",
          ...(showYear && { year: "numeric" }),
        }, settings);
        
      default:
        return formatDateWithSettings(dateObj, {
          hour: "numeric",
          minute: "2-digit",
        }, settings);
    }
  },
  
  chartTooltip: (date: Date | string, timeRange: string) => {
    const settings = getTimeFormatSettings();
    const dateObj = typeof date === "string" ? new Date(date) : date;
    
    // More detailed formatting for tooltips
    switch (timeRange) {
      case "1d":
      case "3d":
        return formatDateWithSettings(dateObj, {
          weekday: "long",
          month: "long",
          day: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }, settings);
        
      case "1w":
        return formatDateWithSettings(dateObj, {
          weekday: "long",
          month: "long",
          day: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }, settings);
        
      case "1m":
        return formatDateWithSettings(dateObj, {
          weekday: "long",
          month: "long",
          day: "numeric",
          year: "numeric",
        }, settings);
        
      case "all":
        return formatDateWithSettings(dateObj, {
          weekday: "long",
          month: "long",
          day: "numeric",
          year: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }, settings);
        
      default:
        return formatDateWithSettings(dateObj, {
          weekday: "long",
          month: "long",
          day: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }, settings);
    }
  },
};
