/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

// Country code to emoji flag mapping
export const flagEmojis: Record<string, string> = {
  US: "🇺🇸",
  NL: "🇳🇱",
  DE: "🇩🇪",
  GB: "🇬🇧",
  FR: "🇫🇷",
  SE: "🇸🇪",
  NO: "🇳🇴",
  DK: "🇩🇰",
  FI: "🇫🇮",
  IT: "🇮🇹",
  ES: "🇪🇸",
  PL: "🇵🇱",
  CH: "🇨🇭",
  AT: "🇦🇹",
  BE: "🇧🇪",
  CZ: "🇨🇿",
  CA: "🇨🇦",
  AU: "🇦🇺",
  JP: "🇯🇵",
  KR: "🇰🇷",
  CN: "🇨🇳",
  BR: "🇧🇷",
  MX: "🇲🇽",
  IN: "🇮🇳",
  RU: "🇷🇺",
  UA: "🇺🇦",
  IE: "🇮🇪",
  PT: "🇵🇹",
  GR: "🇬🇷",
  HU: "🇭🇺",
  RO: "🇷🇴",
  BG: "🇧🇬",
  HR: "🇭🇷",
  SI: "🇸🇮",
  SK: "🇸🇰",
  LT: "🇱🇹",
  LV: "🇱🇻",
  EE: "🇪🇪",
  IS: "🇮🇸",
  LU: "🇱🇺",
  MT: "🇲🇹",
  CY: "🇨🇾",
  SG: "🇸🇬",
  HK: "🇭🇰",
  TW: "🇹🇼",
  ID: "🇮🇩",
  MY: "🇲🇾",
  TH: "🇹🇭",
  VN: "🇻🇳",
  PH: "🇵🇭",
  AR: "🇦🇷",
  CL: "🇨🇱",
  CO: "🇨🇴",
  PE: "🇵🇪",
  VE: "🇻🇪",
  ZA: "🇿🇦",
  EG: "🇪🇬",
  NG: "🇳🇬",
  KE: "🇰🇪",
  MA: "🇲🇦",
  IL: "🇮🇱",
  TR: "🇹🇷",
  SA: "🇸🇦",
  AE: "🇦🇪",
  QA: "🇶🇦",
  NZ: "🇳🇿",
  FJ: "🇫🇯",
};

/**
 * Get emoji flag for a country code
 * @param countryCode - ISO 3166-1 alpha-2 country code
 * @returns Flag emoji or empty string if not found
 */
export const getCountryFlag = (countryCode?: string): string => {
  if (!countryCode) return "";
  return flagEmojis[countryCode.toUpperCase()] || "";
};
