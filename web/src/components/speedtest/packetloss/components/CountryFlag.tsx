/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { getCountryFlag } from "@/utils/countryFlags";

interface CountryFlagProps {
  countryCode?: string;
  className?: string;
}

export const CountryFlag: React.FC<CountryFlagProps> = ({
  countryCode,
  className = "w-4 h-3",
}) => {
  if (!countryCode) return null;

  return (
    <span
      className={`inline-block ${className}`}
      style={{ verticalAlign: "baseline" }}
    >
      {getCountryFlag(countryCode)}
    </span>
  );
};
