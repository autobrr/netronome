/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  SwatchIcon,
  GlobeAltIcon,
  LockClosedIcon,
  CheckCircleIcon,
} from "@heroicons/react/24/outline";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { colorThemes, type ColorTheme } from "@/config/themes";
import { useColorThemeSelection } from "@/hooks/useLicense";
import { LicenseSettings } from "./LicenseSettings";
import { cn } from "@/lib/utils";

import { PremiumLicenseModal } from "@/components/PremiumLicenseModal";

export const ThemeSettings: React.FC = () => {
  const {
    hasPremium,
    activeTheme,
    publicTheme,
    isLoading,
    isPending,
    selectTheme,
    selectPublicTheme,
  } = useColorThemeSelection();

  // Locked cards and the footer note used to link straight to Polar checkout,
  // which silently hid the crypto route. Both open the purchase dialog now.
  const [showPurchase, setShowPurchase] = useState(false);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[220px]">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full mx-auto mb-4 animate-spin" />
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Loading theme settings...
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <SwatchIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
          Themes & License
        </h3>
        <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
          Choose the color palette used across Netronome and manage the license
          that unlocks premium themes
        </p>
      </div>

      <LicenseSettings />

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <SwatchIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            Color Theme
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {colorThemes.map((theme) => {
              const locked = theme.isPremium && !hasPremium;
              const isActive = theme.id === activeTheme;

              const cardClass = cn(
                "block w-full text-left rounded-lg border p-4 transition-colors",
                locked
                  ? "border-gray-200 dark:border-gray-800 bg-gray-100/60 dark:bg-gray-900/60 opacity-60 hover:opacity-100"
                  : isActive
                    ? "border-blue-500 dark:border-blue-400 bg-blue-50 dark:bg-blue-950/30 ring-1 ring-blue-500/40"
                    : "border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-700"
              );

              const body = (
                <>
                  <div className="flex items-start justify-between gap-2">
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {theme.name}
                    </span>
                    {locked ? (
                      <LockClosedIcon className="w-4 h-4 shrink-0 text-gray-500 dark:text-gray-400" />
                    ) : isActive ? (
                      <CheckCircleIcon className="w-4 h-4 shrink-0 text-blue-600 dark:text-blue-400" />
                    ) : null}
                  </div>

                  {theme.description && (
                    <p className="mt-1 text-xs text-gray-600 dark:text-gray-400">
                      {theme.description}
                    </p>
                  )}

                  <div className="mt-3 flex items-center gap-1.5">
                    {theme.swatches.map((color, i) => (
                      <span
                        key={i}
                        className="h-5 w-5 rounded-full border border-gray-300 dark:border-gray-700"
                        style={{ backgroundColor: color }}
                      />
                    ))}
                  </div>

                  {locked && (
                    <span className="mt-3 block text-xs font-medium text-blue-600 dark:text-blue-400">
                      Unlock with a license
                    </span>
                  )}
                </>
              );

              return locked ? (
                <button
                  key={theme.id}
                  type="button"
                  onClick={() => setShowPurchase(true)}
                  className={cardClass}
                >
                  {body}
                </button>
              ) : (
                <button
                  key={theme.id}
                  type="button"
                  onClick={() => selectTheme(theme)}
                  disabled={isPending}
                  className={cardClass}
                >
                  {body}
                </button>
              );
            })}
          </div>

          {!hasPremium && (
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Premium themes are unlocked with a one-time{" "}
              <button
                type="button"
                onClick={() => setShowPurchase(true)}
                className="text-blue-600 dark:text-blue-400 hover:underline"
              >
                Netronome license
              </button>
              . Already have a key? Activate it above.
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <GlobeAltIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            Public Dashboard Theme
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label
              id="public-theme-label"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              Theme
            </label>
            <Select
              value={publicTheme}
              onValueChange={selectPublicTheme}
              disabled={isPending}
            >
              <SelectTrigger
                aria-labelledby="public-theme-label"
                className="w-full sm:w-[240px]"
              >
                <SelectValue placeholder="Select theme" />
              </SelectTrigger>
              <SelectContent>
                {colorThemes.map((theme: ColorTheme) => {
                  const locked = theme.isPremium && !hasPremium;
                  return (
                    <SelectItem key={theme.id} value={theme.id} disabled={locked}>
                      {locked ? `${theme.name} (locked)` : theme.name}
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </div>

          <p className="text-sm text-gray-600 dark:text-gray-400">
            This is the theme anonymous visitors see on the public dashboard at /public.
          </p>
        </CardContent>
      </Card>

      <PremiumLicenseModal
        isOpen={showPurchase}
        onClose={() => setShowPurchase(false)}
      />
    </div>
  );
};
