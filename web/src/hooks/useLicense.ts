/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  activateLicense,
  deactivateLicense,
  getLicense,
  getThemeSettings,
  updateThemeSettings,
  type LicenseStatus,
  type ThemeSettings,
} from "@/api/license";
import {
  hasPremiumAccessCached,
  reconcileColorTheme,
  getStoredColorThemeId,
  setColorTheme,
} from "@/utils/colorTheme";
import { DEFAULT_THEME_ID, type ColorTheme } from "@/config/themes";
import { showToast } from "@/components/common/Toast";

const LICENSE_KEY = ["license"];
const THEME_SETTINGS_KEY = ["theme-settings"];

/** Revalidation ticker. The server owns the offline grace period. */
const REVALIDATE_INTERVAL = 24 * 60 * 60 * 1000;

export const useLicense = () => {
  const query = useQuery<LicenseStatus>({
    queryKey: LICENSE_KEY,
    queryFn: getLicense,
    staleTime: REVALIDATE_INTERVAL,
    refetchInterval: REVALIDATE_INTERVAL,
  });

  const authoritative = query.data?.hasPremiumAccess;

  // Authoritative state arrived: revert an active premium theme if entitlement
  // was lost. colorTheme.ts owns the actual reconciliation.
  useEffect(() => {
    if (authoritative === undefined) return;
    reconcileColorTheme(authoritative);
  }, [authoritative]);

  // Returns the react-query result as-is so callers use the usual {data,
  // isLoading, ...} shape. hasPremiumAccess falls back to the last known
  // entitlement until the server answers, so a premium theme does not flash
  // back to default on load or while the server is unreachable.
  return {
    ...query,
    data: query.data
      ? query.data
      : ({
          hasPremiumAccess: hasPremiumAccessCached(),
          license: null,
        } satisfies LicenseStatus),
  };
};

export const useActivateLicense = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (licenseKey: string) => activateLicense(licenseKey),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LICENSE_KEY });
    },
  });
};

export const useDeactivateLicense = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deactivateLicense,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LICENSE_KEY });
    },
  });
};

export const useThemeSettings = () =>
  useQuery<ThemeSettings>({
    queryKey: THEME_SETTINGS_KEY,
    queryFn: getThemeSettings,
  });

export const useUpdateThemeSettings = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateThemeSettings,
    // The response is the new state, so no refetch is needed.
    onSuccess: (data) => queryClient.setQueryData(THEME_SETTINGS_KEY, data),
  });
};

/**
 * Shared color-theme selection: entitlement, active/public theme, and
 * persistence with optimistic apply + revert on save failure. Used by the
 * header dropdown, the mobile menu, and the settings page so they stay in
 * lockstep.
 */
export const useColorThemeSelection = () => {
  const { data: license } = useLicense();
  const { data: settings, isLoading } = useThemeSettings();
  const mutation = useUpdateThemeSettings();

  // Fall back to the cached entitlement so premium themes do not flash
  // "locked" while the license request is in flight.
  const hasPremium = license?.hasPremiumAccess ?? hasPremiumAccessCached();

  const activeTheme = settings?.theme ?? getStoredColorThemeId();
  const publicTheme = settings?.publicTheme ?? DEFAULT_THEME_ID;

  // Boot only applied the localStorage guess; the server is authoritative.
  useEffect(() => {
    if (settings?.theme) setColorTheme(settings.theme, hasPremium);
  }, [settings?.theme, hasPremium]);

  const save = (theme: string, publicThemeId: string, revertTo?: string) => {
    mutation.mutate(
      { theme, publicTheme: publicThemeId },
      {
        onError: (err: unknown) => {
          if (revertTo) setColorTheme(revertTo, hasPremium);
          showToast("Failed to save theme", "error", {
            description: err instanceof Error ? err.message : undefined,
          });
        },
      }
    );
  };

  const selectTheme = (theme: ColorTheme) => {
    if (theme.id === activeTheme) return;
    // Returns false for premium themes without entitlement - locked themes
    // are never previewed, deliberately.
    if (!setColorTheme(theme.id, hasPremium)) return;
    save(theme.id, publicTheme, activeTheme);
  };

  const selectPublicTheme = (id: string) => save(activeTheme, id);

  return {
    hasPremium,
    activeTheme,
    publicTheme,
    isLoading,
    isPending: mutation.isPending,
    selectTheme,
    selectPublicTheme,
  };
};
