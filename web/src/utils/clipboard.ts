/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { TracerouteResult, TracerouteHop } from "@/types/types";

export const filterTrailingTimeouts = (hops: TracerouteHop[]) => {
  if (!hops || hops.length === 0) return hops;
  
  // Find the last non-timeout hop
  let lastValidIndex = -1;
  for (let i = hops.length - 1; i >= 0; i--) {
    if (!hops[i].timeout) {
      lastValidIndex = i;
      break;
    }
  }
  
  // If no valid hops found, return all hops
  if (lastValidIndex === -1) return hops;
  
  // Check if there are 3 or more consecutive timeouts at the end
  const trailingTimeouts = hops.length - 1 - lastValidIndex;
  if (trailingTimeouts >= 3) {
    // Return hops up to the last valid hop
    return hops.slice(0, lastValidIndex + 1);
  }
  
  return hops;
};

export const formatTracerouteForClipboard = (results: TracerouteResult): string => {
  const filteredHops = filterTrailingTimeouts(results.hops);
  
  // Simple line-by-line format
  let output = `Traceroute to ${results.destination} (${results.ip || results.destination})\n`;
  output += `${filteredHops.length} hops max\n\n`;
  
  filteredHops.forEach((hop) => {
    if (hop.timeout) {
      output += `${hop.number}  * * *\n`;
    } else {
      const rtt1 = hop.rtt1 > 0 ? `${hop.rtt1.toFixed(1)}ms` : '*';
      const rtt2 = hop.rtt2 > 0 ? `${hop.rtt2.toFixed(1)}ms` : '*';
      const rtt3 = hop.rtt3 > 0 ? `${hop.rtt3.toFixed(1)}ms` : '*';
      
      const asInfo = hop.as ? ` [AS${hop.as}]` : '';
      const countryInfo = hop.countryCode ? ` (${hop.countryCode})` : '';
      
      output += `${hop.number}  ${hop.host}${asInfo}${countryInfo}  ${rtt1} ${rtt2} ${rtt3}\n`;
    }
  });

  return `\`\`\`\n${output}\`\`\``;
};

export const copyToClipboard = async (text: string): Promise<boolean> => {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error('Failed to copy to clipboard:', err);
    return false;
  }
};