/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Joins a saved iperf3 host and port into the endpoint string that the server
 * uses as `serverHost`.
 *
 * An IPv6 address has at least two colons. Without brackets the port runs into
 * the address, for example `2001:db8::1:5203`, and the last group reads as part
 * of the address. Brackets keep the two apart: `[2001:db8::1]:5203`.
 */
export const formatIperfServerEndpoint = (
  host: string,
  port: number | string,
): string => {
  const trimmedHost = host.trim();
  const isBracketed = trimmedHost.startsWith("[");
  const isIPv6 = trimmedHost.split(":").length > 2;
  if (isIPv6 && !isBracketed) {
    return `[${trimmedHost}]:${port}`;
  }
  return `${trimmedHost}:${port}`;
};
