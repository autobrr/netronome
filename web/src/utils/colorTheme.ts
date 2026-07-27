/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Colour theme application.
 *
 * Orthogonal to darkMode.ts: that owns the light/dark axis (the `.dark` class),
 * this owns which colour ramp is in play (the `theme-<id>` class). A theme's
 * CSS carries both a base block and a `.dark` block, so the two compose.
 */

import {
  DEFAULT_THEME_ID,
  getColorThemeById,
  type ColorTheme,
} from "@/config/themes";

const COLOR_THEME_KEY = "colorTheme";
const PREMIUM_CACHE_KEY = "colorTheme.premiumAccess";
const STYLE_ELEMENT_ID = "netronome-color-theme";
const THEME_CLASS_PREFIX = "theme-";
const TRANSITION_CLASS = "theme-transition";
const TRANSITION_DURATION = 300;

/**
 * Last known entitlement, used only to avoid a flash of "locked" on first
 * paint before the license request resolves. The server is authoritative and
 * owns the offline grace period, so there is deliberately no grace logic here.
 */
export const hasPremiumAccessCached = (): boolean => {
  try {
    return localStorage.getItem(PREMIUM_CACHE_KEY) === "true";
  } catch {
    return false;
  }
};

const setPremiumAccessCache = (value: boolean): void => {
  try {
    localStorage.setItem(PREMIUM_CACHE_KEY, String(value));
  } catch {
    // ignore quota/private-mode errors
  }
};

export const getStoredColorThemeId = (): string => {
  try {
    return localStorage.getItem(COLOR_THEME_KEY) ?? DEFAULT_THEME_ID;
  } catch {
    return DEFAULT_THEME_ID;
  }
};

const setStoredColorThemeId = (id: string): void => {
  try {
    localStorage.setItem(COLOR_THEME_KEY, id);
  } catch {
    // ignore quota/private-mode errors
  }
};

const isAccessible = (theme: ColorTheme, hasPremium: boolean): boolean =>
  !theme.isPremium || hasPremium;

/** Resolve an id to a theme the current entitlement actually permits. */
const resolveTheme = (id: string, hasPremium: boolean): ColorTheme => {
  const theme = getColorThemeById(id);
  if (theme && isAccessible(theme, hasPremium)) return theme;
  return getColorThemeById(DEFAULT_THEME_ID)!;
};

const applyColorTheme = (theme: ColorTheme, withTransition: boolean): void => {
  const root = document.documentElement;

  if (withTransition) {
    root.classList.add(TRANSITION_CLASS);
    setTimeout(
      () => root.classList.remove(TRANSITION_CLASS),
      TRANSITION_DURATION
    );
  }

  // Snapshot first: removing from a live DOMTokenList mid-iteration skips entries.
  // Note TRANSITION_CLASS also starts with "theme-", hence the explicit guard.
  for (const cls of Array.from(root.classList)) {
    if (cls.startsWith(THEME_CLASS_PREFIX) && cls !== TRANSITION_CLASS) {
      root.classList.remove(cls);
    }
  }

  let style = document.getElementById(STYLE_ELEMENT_ID);
  if (!theme.css) {
    style?.remove();
    return;
  }

  if (!style) {
    style = document.createElement("style");
    style.id = STYLE_ELEMENT_ID;
    document.head.appendChild(style);
  }
  style.textContent = theme.css;
  root.classList.add(`${THEME_CLASS_PREFIX}${theme.id}`);
};

/** True on the anonymous public dashboard (/public). */
export const isPublicDashboardRoute = (): boolean =>
  window.location.pathname.includes("/public");

/** Apply the cached theme synchronously at boot, before first paint. */
export const initColorTheme = (): void => {
  // The public dashboard shows the server-configured public theme (fetched
  // after mount), never the operator's personal choice from localStorage.
  if (isPublicDashboardRoute()) return;
  applyColorTheme(
    resolveTheme(getStoredColorThemeId(), hasPremiumAccessCached()),
    false
  );
};

/**
 * Apply the server-resolved public dashboard theme. The server already gates
 * entitlement (it returns the default id when unlicensed), so there is no
 * premium check here, and nothing is persisted - localStorage keeps the
 * operator's own choice.
 */
export const applyPublicColorTheme = (id: string): void => {
  const theme = getColorThemeById(id);
  if (theme) applyColorTheme(theme, false);
};

/**
 * Switch themes. Returns false if the theme is premium and the current
 * entitlement does not cover it, in which case nothing changes.
 */
export const setColorTheme = (id: string, hasPremium: boolean): boolean => {
  const theme = getColorThemeById(id);
  if (!theme || !isAccessible(theme, hasPremium)) return false;

  applyColorTheme(theme, true);
  setStoredColorThemeId(theme.id);
  return true;
};

/**
 * Reconcile against authoritative license state once it arrives. Reverts to
 * the default theme if entitlement was lost while a premium theme was active.
 */
export const reconcileColorTheme = (hasPremium: boolean): void => {
  setPremiumAccessCache(hasPremium);

  const stored = getStoredColorThemeId();
  const resolved = resolveTheme(stored, hasPremium);
  if (resolved.id === stored) return;

  applyColorTheme(resolved, false);
  setStoredColorThemeId(resolved.id);
};
