/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Theme registry.
 *
 * A theme is a plain CSS file that redefines Tailwind's colour ramp
 * (`--color-gray-*`, `--color-white`, `--color-blue-*`, ...) under
 * `:root.theme-<id>` and, for dark mode, `:root.theme-<id>.dark`.
 *
 * Because Tailwind v4 emits every utility as a `var()` reference, overriding
 * the ramp retints the whole app with no component changes. The CSS is
 * injected verbatim, so there is no parser beyond the metadata header.
 */

export const DEFAULT_THEME_ID = "netronome";

export interface ColorTheme {
  id: string;
  name: string;
  description?: string;
  isPremium: boolean;
  /** Representative colours for the picker, sampled from the dark block. */
  swatches: string[];
  /** Raw CSS, injected as-is. Empty for the built-in default. */
  css: string;
}

// Premium themes are fetched from the private repo at release time into
// src/themes/premium/. Absent in source builds, which is expected.
const premiumFiles = import.meta.glob("/src/themes/premium/*.css", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// Fixture theme so the engine stays exercisable without THEMES_REPO_TOKEN.
// Registered in dev only - it must never appear in a production picker.
const devFiles = import.meta.glob("/src/themes/__dev__/*.css", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const metaValue = (css: string, key: string): string | undefined =>
  css.match(new RegExp(`@${key}:\\s*(.+)`))?.[1]?.trim();

// Later declarations win, so the last match for a var is the dark-block value.
const lastVarValue = (css: string, name: string): string | undefined => {
  const matches = [...css.matchAll(new RegExp(`--color-${name}:\\s*([^;]+);`, "g"))];
  return matches[matches.length - 1]?.[1]?.trim();
};

const SWATCH_VARS = [
  "gray-950",
  "gray-800",
  "blue-500",
  "emerald-500",
  "gray-100",
];

const extractSwatches = (css: string): string[] => {
  const preferred = SWATCH_VARS.map((v) => lastVarValue(css, v)).filter(
    (v): v is string => Boolean(v)
  );
  if (preferred.length >= 3) return preferred;

  // Fall back to whatever colours the theme does define, deduped.
  const all = [...css.matchAll(/--color-[a-z0-9-]+:\s*([^;]+);/g)].map((m) =>
    m[1].trim()
  );
  return [...new Set([...preferred, ...all])].slice(0, 5);
};

const parseTheme = (path: string, css: string, premiumDefault: boolean): ColorTheme => {
  const id = path.split("/").pop()!.replace(".css", "");
  return {
    id,
    name: metaValue(css, "name") ?? id,
    description: metaValue(css, "description"),
    isPremium: (metaValue(css, "premium") ?? String(premiumDefault)) === "true",
    swatches: extractSwatches(css),
    css,
  };
};

const defaultTheme: ColorTheme = {
  id: DEFAULT_THEME_ID,
  name: "Netronome",
  description: "The default look.",
  isPremium: false,
  swatches: ["#09090b", "#27272a", "#3b82f6", "#10b981", "#f4f4f5"],
  css: "",
};

export const colorThemes: ColorTheme[] = [
  defaultTheme,
  ...Object.entries(premiumFiles).map(([p, css]) => parseTheme(p, css, true)),
  ...(import.meta.env.DEV
    ? Object.entries(devFiles).map(([p, css]) => parseTheme(p, css, false))
    : []),
];

export const getColorThemeById = (id: string): ColorTheme | undefined =>
  colorThemes.find((t) => t.id === id);

export const isColorThemePremium = (id: string): boolean =>
  getColorThemeById(id)?.isPremium ?? false;
