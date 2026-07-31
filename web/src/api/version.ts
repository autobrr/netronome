/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";

export interface AppRelease {
  tag_name: string;
  name?: string;
  html_url: string;
  published_at: string;
}

export const latestReleaseQueryKey = ["version", "latest"] as const;

const isSafeReleaseUrl = (value: string): boolean => {
  try {
    return new URL(value).protocol === "https:";
  } catch {
    return false;
  }
};

const isAppRelease = (value: unknown): value is AppRelease => {
  if (!value || typeof value !== "object") return false;

  const release = value as Record<string, unknown>;
  return (
    typeof release.tag_name === "string" &&
    typeof release.html_url === "string" &&
    isSafeReleaseUrl(release.html_url) &&
    typeof release.published_at === "string" &&
    (release.name === undefined || typeof release.name === "string")
  );
};

export async function getLatestRelease(): Promise<AppRelease | null> {
  const response = await fetch(getApiUrl("/version/latest"), {
    credentials: "include",
  });

  if (response.status === 204) return null;
  if (response.status !== 200) {
    throw new Error(response.statusText || "Failed to fetch latest release");
  }

  const release: unknown = await response.json();
  if (!isAppRelease(release)) {
    throw new Error("Invalid latest release response");
  }

  return release;
}

export function formatReleaseTag(tagName: string): string {
  return tagName.startsWith("v") ? tagName : `v${tagName}`;
}
