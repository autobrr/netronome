/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { DNSMonitorInput, DNSProtocol } from "@/types/types";

export const protocolOptions: { value: DNSProtocol; label: string }[] = [
  { value: "udp", label: "UDP" },
  { value: "tcp", label: "TCP" },
  { value: "dot", label: "DNS over TLS" },
];

export const recordTypes = ["A", "AAAA", "CNAME", "MX", "NS", "TXT"];

// Quick-picks for well-known public resolvers. They only prefill the form; the
// resolver field stays free text.
export const resolverPresets: {
  label: string;
  host: string;
  protocol: DNSProtocol;
}[] = [
  { label: "Cloudflare", host: "1.1.1.1", protocol: "udp" },
  { label: "Cloudflare DoT", host: "one.one.one.one", protocol: "dot" },
  { label: "Google", host: "8.8.8.8", protocol: "udp" },
  { label: "Google DoT", host: "dns.google", protocol: "dot" },
  { label: "Quad9", host: "9.9.9.9", protocol: "udp" },
  { label: "Quad9 DoT", host: "dns.quad9.net", protocol: "dot" },
  { label: "OpenDNS", host: "208.67.222.222", protocol: "udp" },
  { label: "OpenDNS DoT", host: "dns.opendns.com", protocol: "dot" },
];

// Badges for the states the server reports. A monitor that has not run yet is
// in the "unknown" state and gets no badge.
const STATE_BADGES: Record<string, { label: string; className: string }> = {
  ok: {
    label: "OK",
    className:
      "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20",
  },
  down: {
    label: "Down",
    className: "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20",
  },
  recovered: {
    label: "Recovered",
    className:
      "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20",
  },
};

export const stateBadge = (state?: string) => STATE_BADGES[state ?? ""] ?? null;

export const defaultDNSFormData: DNSMonitorInput = {
  host: "",
  name: "",
  protocol: "udp",
  query: "google.com",
  recordType: "A",
  interval: "1m",
  enabled: true,
};
