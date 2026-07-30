/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";

type ApiErrorPayload = {
  error?: string;
  details?: string;
  message?: string;
};

const parseErrorMessage = async (response: Response): Promise<string> => {
  const text = await response.text().catch(() => "");
  if (text) {
    try {
      const json = JSON.parse(text) as ApiErrorPayload;
      const msg = json.details
        ? `${json.error ?? "Request failed"}: ${json.details}`
        : (json.error ?? json.message);
      if (msg) return msg;
    } catch {
      // fall through: non-JSON error body
    }
    // Only surface raw bodies that read as short plain-text messages - proxy
    // HTML error pages make terrible toasts.
    if (text.length <= 200 && !/[<>]/.test(text)) return text;
  }
  return response.statusText || `Request failed (${response.status})`;
};

const assertOk = async (response: Response, fallback: string) => {
  if (response.ok) return;
  const msg = await parseErrorMessage(response);
  throw new Error(msg || fallback);
};

export interface License {
  /** Masked by the server, never the full key. */
  licenseKey: string;
  status: string;
  productName: string;
  activatedAt: string;
  expiresAt: string | null;
}

export interface LicenseStatus {
  hasPremiumAccess: boolean;
  license: License | null;
}

export interface ThemeSettings {
  theme: string;
  publicTheme: string;
}

// Both fields are required: the server validates them as a pair and rejects an
// empty id, so a partial update would 400.
export type UpdateThemeSettings = ThemeSettings;

export async function getLicense(): Promise<LicenseStatus> {
  const response = await fetch(getApiUrl("/license"), {
    credentials: "include",
  });
  await assertOk(response, "Failed to fetch license");
  return response.json();
}

export async function activateLicense(
  licenseKey: string
): Promise<LicenseStatus> {
  const response = await fetch(getApiUrl("/license/activate"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ licenseKey }),
  });
  await assertOk(response, "Failed to activate license");
  return response.json();
}

export async function deactivateLicense(): Promise<LicenseStatus> {
  const response = await fetch(getApiUrl("/license/deactivate"), {
    method: "POST",
    credentials: "include",
  });
  await assertOk(response, "Failed to deactivate license");
  return response.json();
}

export async function getThemeSettings(): Promise<ThemeSettings> {
  const response = await fetch(getApiUrl("/settings/theme"), {
    credentials: "include",
  });
  await assertOk(response, "Failed to fetch theme settings");
  return response.json();
}

export async function updateThemeSettings(
  input: UpdateThemeSettings
): Promise<ThemeSettings> {
  const response = await fetch(getApiUrl("/settings/theme"), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(input),
  });
  await assertOk(response, "Failed to update theme settings");
  return response.json();
}

/** Unauthenticated: the theme the public dashboard renders with. */
export async function getPublicTheme(): Promise<{ theme: string }> {
  const response = await fetch(getApiUrl("/public/theme"));
  await assertOk(response, "Failed to fetch public theme");
  return response.json();
}
