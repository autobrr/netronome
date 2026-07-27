/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useCallback, useEffect, useState } from "react";
import {
  SwatchIcon,
  CheckIcon,
  SunIcon,
  ComputerDesktopIcon,
} from "@heroicons/react/24/outline";
import { MoonIcon } from "@heroicons/react/24/solid";
import { Button } from "@/components/ui/Button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { colorThemes, type ColorTheme } from "@/config/themes";
import { useColorThemeSelection } from "@/hooks/useLicense";
import {
  getCurrentThemeMode,
  setThemeMode,
  type ThemeMode,
} from "@/utils/darkMode";
import { cn } from "@/lib/utils";

const MODES: {
  value: ThemeMode;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}[] = [
  { value: "light", label: "Light", icon: SunIcon },
  { value: "dark", label: "Dark", icon: MoonIcon },
  { value: "auto", label: "System", icon: ComputerDesktopIcon },
];

/** Track the active mode across tabs, system changes, and other pickers. */
const useThemeMode = (): ThemeMode => {
  const [mode, setMode] = useState<ThemeMode>(getCurrentThemeMode());

  useEffect(() => {
    const check = () => setMode(getCurrentThemeMode());
    const onStorage = (e: StorageEvent) => {
      if (e.key === "theme") check();
    };
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");

    window.addEventListener("storage", onStorage);
    window.addEventListener("themechange", check);
    mediaQuery.addEventListener("change", check);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener("themechange", check);
      mediaQuery.removeEventListener("change", check);
    };
  }, []);

  return mode;
};

const sectionHeaderClass =
  "px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400";
const itemClass =
  "cursor-pointer gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700";

/** Accent swatch for the row dot; index 2 is the theme's blue-500 sample. */
const accentOf = (theme: ColorTheme): string =>
  theme.swatches[2] ?? theme.swatches[0];

const ThemeRowBody: React.FC<{
  theme: ColorTheme;
  isActive: boolean;
  isPremium: boolean;
}> = ({ theme, isActive, isPremium }) => (
  <>
    <span
      className="h-4 w-4 shrink-0 rounded-full border border-gray-300 dark:border-gray-600"
      style={{ backgroundColor: accentOf(theme) }}
    />
    <span className="flex-1 truncate text-left font-medium">{theme.name}</span>
    {isPremium && (
      <span className="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">
        Premium
      </span>
    )}
    {isActive && (
      <CheckIcon className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
    )}
  </>
);

/** Header appearance menu - light/dark mode and color theme in one place. */
export const ThemeDropdown: React.FC = () => {
  const { hasPremium, activeTheme, selectTheme } = useColorThemeSelection();
  const mode = useThemeMode();

  const selectMode = useCallback((value: ThemeMode) => {
    setThemeMode(value);
  }, []);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400"
          aria-label="Appearance"
        >
          <SwatchIcon className="h-6 w-6" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-64 bg-white shadow-xl ring-1 ring-black/10 dark:bg-gray-800 dark:ring-white/10"
      >
        <div className={sectionHeaderClass}>Mode</div>
        {MODES.map(({ value, label, icon: Icon }) => (
          <DropdownMenuItem
            key={value}
            onClick={() => selectMode(value)}
            className={itemClass}
          >
            <Icon className="h-4 w-4 shrink-0 text-gray-600 dark:text-gray-400" />
            <span className="flex-1 text-left font-medium">{label}</span>
            {mode === value && (
              <CheckIcon className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
            )}
          </DropdownMenuItem>
        ))}

        <DropdownMenuSeparator className="bg-gray-200 dark:bg-gray-700" />

        <div className={sectionHeaderClass}>Color Theme</div>
        {colorThemes.map((theme) => {
          const locked = theme.isPremium && !hasPremium;
          return (
            <DropdownMenuItem
              key={theme.id}
              disabled={locked}
              onClick={() => selectTheme(theme)}
              className={cn(itemClass, locked && "opacity-60")}
            >
              <ThemeRowBody
                theme={theme}
                isActive={theme.id === activeTheme}
                isPremium={theme.isPremium}
              />
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

/** Color theme list for the mobile sheet, styled like its mode section. */
export const MobileThemePicker: React.FC = () => {
  const { hasPremium, activeTheme, selectTheme } = useColorThemeSelection();

  return (
    <div className="px-4 py-3">
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
        Color Theme
      </h3>
      <div className="space-y-2">
        {colorThemes.map((theme) => {
          const locked = theme.isPremium && !hasPremium;
          const isActive = theme.id === activeTheme;
          return (
            <Button
              key={theme.id}
              variant="ghost"
              disabled={locked}
              onClick={() => selectTheme(theme)}
              className={cn(
                "h-auto w-full justify-start gap-3 px-3 py-2 text-gray-700 dark:text-gray-300",
                isActive &&
                  "bg-blue-50 text-blue-600 hover:bg-blue-50 dark:bg-blue-900/20 dark:text-blue-400 dark:hover:bg-blue-900/20",
                locked && "opacity-60"
              )}
            >
              <ThemeRowBody
                theme={theme}
                isActive={isActive}
                isPremium={theme.isPremium}
              />
            </Button>
          );
        })}
      </div>
    </div>
  );
};
