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
import { hasPremiumAccessCached, reconcileColorTheme } from "@/utils/colorTheme";

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
