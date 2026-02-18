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
    return text;
  }
  return response.statusText || `Request failed (${response.status})`;
};

const assertOk = async (response: Response, fallback: string) => {
  if (response.ok) return;
  const msg = await parseErrorMessage(response);
  throw new Error(msg || fallback);
};

export interface DashboardSettings {
  recentSpeedtestsRows: number;
}

export const settingsApi = {
  getDashboardSettings: async (): Promise<DashboardSettings> => {
    const response = await fetch(getApiUrl("/settings/dashboard"), {
      credentials: "include",
    });
    await assertOk(response, "Failed to fetch dashboard settings");
    return response.json();
  },

  updateDashboardSettings: async (
    input: DashboardSettings
  ): Promise<DashboardSettings> => {
    const response = await fetch(getApiUrl("/settings/dashboard"), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(input),
    });
    await assertOk(response, "Failed to update dashboard settings");
    return response.json();
  },
};
