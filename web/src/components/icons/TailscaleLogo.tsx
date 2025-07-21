/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";

interface TailscaleLogoProps {
  className?: string;
  title?: string;
}

export const TailscaleLogo: React.FC<TailscaleLogoProps> = ({
  className = "h-4 w-4",
  title = "Tailscale",
}) => {
  return (
    <svg
      className={className}
      viewBox="30 30 70 70"
      xmlns="http://www.w3.org/2000/svg"
      role="img"
      aria-label={title}
    >
      <title>{title}</title>
      {/* Tailscale logo dots - using original color in light mode, lighter in dark mode */}
      <g className="fill-[#141414] dark:fill-gray-300">
        <circle cx="45.6297" cy="60.5186" r="6.62966" />
        <circle cx="65.5186" cy="60.5186" r="6.62966" />
        <circle opacity="0.2" cx="45.6297" cy="80.4077" r="6.62966" />
        <circle opacity="0.2" cx="85.4077" cy="80.4077" r="6.62966" />
        <circle cx="65.5186" cy="80.4077" r="6.62966" />
        <circle cx="85.4077" cy="60.5186" r="6.62966" />
        <circle opacity="0.2" cx="45.6297" cy="40.6297" r="6.62966" />
        <circle opacity="0.2" cx="65.5186" cy="40.6297" r="6.62966" />
        <circle opacity="0.2" cx="85.4077" cy="40.6297" r="6.62966" />
      </g>
    </svg>
  );
};