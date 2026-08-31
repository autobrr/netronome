/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";
import {
  DNSMonitor,
  DNSMonitorInput,
  DNSResult,
  DNSUpdate,
  PaginatedResponse,
} from "@/types/types";

// dnsFetch keeps the error text the API sends, so a rejected monitor shows the
// field that was wrong instead of a generic message.
const dnsFetch = async (
  path: string,
  init?: RequestInit,
): Promise<Response> => {
  const response = await fetch(getApiUrl(path), init);
  if (!response.ok) {
    const message = await response
      .json()
      .then((data) => data.error as string | undefined)
      .catch(() => undefined);
    throw new Error(message || "DNS monitor request failed");
  }
  return response;
};

const jsonBody = (monitor: DNSMonitorInput): RequestInit => ({
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(monitor),
});

export const getDNSMonitors = async (): Promise<DNSMonitor[]> => {
  const response = await dnsFetch("/dns/monitors");
  return response.json();
};

export const createDNSMonitor = async (
  monitor: DNSMonitorInput,
): Promise<DNSMonitor> => {
  const response = await dnsFetch("/dns/monitors", {
    method: "POST",
    ...jsonBody(monitor),
  });
  return response.json();
};

export const updateDNSMonitor = async (
  monitor: DNSMonitorInput & { id: number },
): Promise<DNSMonitor> => {
  const response = await dnsFetch(`/dns/monitors/${monitor.id}`, {
    method: "PUT",
    ...jsonBody(monitor),
  });
  return response.json();
};

export const deleteDNSMonitor = async (id: number): Promise<void> => {
  await dnsFetch(`/dns/monitors/${id}`, { method: "DELETE" });
};

export const getDNSMonitorStatus = async (id: number): Promise<DNSUpdate> => {
  const response = await dnsFetch(`/dns/monitors/${id}/status`);
  return response.json();
};

export const getDNSHistory = async (
  id: number,
  page: number = 1,
  limit: number = 25,
): Promise<PaginatedResponse<DNSResult>> => {
  const response = await dnsFetch(
    `/dns/monitors/${id}/history?page=${page}&limit=${limit}`,
  );
  return response.json();
};
