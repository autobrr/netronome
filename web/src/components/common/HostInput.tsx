/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { motion } from "motion/react";
import { Button } from "@/components/ui/Button";

interface HostInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit?: () => void;
  placeholder?: string;
  disabled?: boolean;
  isLoading?: boolean;
  buttonText?: string;
  showQuickOptions?: boolean;
  className?: string;
  error?: string | null;
}

// Common validation patterns
const IP_PATTERN = /^(\d{1,3}\.){3}\d{1,3}$/;
const HOSTNAME_PATTERN =
  /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
const URL_PATTERN = /^https?:\/\//;

// Validation function
export const validateHost = (
  host: string,
): { isValid: boolean; error?: string } => {
  if (!host || host.trim().length === 0) {
    return { isValid: false, error: "Host is required" };
  }

  const trimmedHost = host.trim();

  // Check if it's a URL
  if (URL_PATTERN.test(trimmedHost)) {
    try {
      new URL(trimmedHost);
      return { isValid: true };
    } catch {
      return { isValid: false, error: "Invalid URL format" };
    }
  }

  // Remove port if present for validation
  const hostWithoutPort = trimmedHost.includes(":")
    ? trimmedHost.split(":")[0]
    : trimmedHost;

  // Check if it's an IP address
  if (IP_PATTERN.test(hostWithoutPort)) {
    const parts = hostWithoutPort.split(".");
    const isValidIP = parts.every((part) => {
      const num = parseInt(part, 10);
      return num >= 0 && num <= 255;
    });

    if (!isValidIP) {
      return { isValid: false, error: "Invalid IP address" };
    }
    return { isValid: true };
  }

  // Check if it's a hostname
  if (HOSTNAME_PATTERN.test(hostWithoutPort)) {
    // Additional checks for hostname
    if (hostWithoutPort.length > 253) {
      return { isValid: false, error: "Hostname too long" };
    }

    const labels = hostWithoutPort.split(".");
    const hasValidLabels = labels.every(
      (label) => label.length <= 63 && label.length > 0,
    );

    if (!hasValidLabels) {
      return { isValid: false, error: "Invalid hostname format" };
    }

    return { isValid: true };
  }

  return { isValid: false, error: "Invalid hostname or IP address" };
};

// Extract hostname from URL or host:port format
export const extractHostname = (input: string): string => {
  let hostname = input.trim();

  // Extract hostname from URL if it's a full URL
  if (hostname.startsWith("http://") || hostname.startsWith("https://")) {
    try {
      const url = new URL(hostname);
      hostname = url.hostname;
    } catch {
      // If URL parsing fails, return as-is
      return hostname;
    }
  }

  // Strip port from hostname if present
  if (hostname.includes(":")) {
    hostname = hostname.split(":")[0];
  }

  return hostname;
};

// Quick option suggestions
const quickOptions = [
  { label: "google.com", value: "google.com" },
  { label: "cloudflare.com", value: "cloudflare.com" },
  { label: "github.com", value: "github.com" },
  { label: "8.8.8.8", value: "8.8.8.8" },
];

export const HostInput: React.FC<HostInputProps> = ({
  value,
  onChange,
  onSubmit,
  placeholder = "Enter hostname/IP (e.g., google.com, 8.8.8.8)",
  disabled = false,
  isLoading = false,
  buttonText = "Submit",
  showQuickOptions = true,
  className = "",
  error: externalError,
}) => {
  const [internalError, setInternalError] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);

  // Use external error if provided, otherwise use internal validation
  const displayError = externalError || (touched ? internalError : null);

  // Validate on value change
  useEffect(() => {
    if (value.trim().length > 0) {
      const validation = validateHost(value);
      setInternalError(validation.isValid ? null : validation.error || null);
    } else {
      setInternalError(null);
    }
  }, [value]);

  const handleSubmit = () => {
    setTouched(true);

    if (!value.trim()) {
      setInternalError("Host is required");
      return;
    }

    const validation = validateHost(value);
    if (!validation.isValid) {
      setInternalError(validation.error || "Invalid host");
      return;
    }

    if (onSubmit) {
      onSubmit();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !disabled && !isLoading) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleQuickOptionSelect = (optionValue: string) => {
    onChange(optionValue);
    setTouched(false);
    setInternalError(null);
  };

  const isValid = value.trim().length > 0 && !displayError;

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Host Input */}
      <div className="flex gap-3">
        <div className="flex-1">
          <input
            type="text"
            value={value}
            onChange={(e) => {
              onChange(e.target.value);
              if (touched) setTouched(false); // Reset touched state when typing
            }}
            onBlur={() => setTouched(true)}
            placeholder={placeholder}
            className={`w-full px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50 ${
              displayError
                ? "border-red-300 dark:border-red-600"
                : "border-gray-300 dark:border-gray-800"
            }`}
            disabled={disabled}
            onKeyDown={handleKeyDown}
          />
          {displayError && (
            <motion.p
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="text-red-600 dark:text-red-400 text-sm mt-1"
            >
              {displayError}
            </motion.p>
          )}
        </div>

        {onSubmit && (
          <Button
            onClick={handleSubmit}
            disabled={disabled || isLoading || !isValid}
            isLoading={isLoading}
            className={`${
              isLoading
                ? "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-300 dark:border-emerald-500/30 cursor-not-allowed"
                : !isValid
                  ? "bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500 border-gray-200 dark:border-gray-700 cursor-not-allowed"
                  : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
            }`}
          >
            {isLoading ? "Loading..." : buttonText}
          </Button>
        )}
      </div>

      {/* Quick Options */}
      {showQuickOptions && !disabled && (
        <div>
          <p className="text-gray-600 dark:text-gray-400 text-sm mb-2">
            Quick options:
          </p>
          <div className="flex flex-wrap gap-2">
            {quickOptions.map((option) => (
              <button
                key={option.value}
                onClick={() => handleQuickOptionSelect(option.value)}
                className="px-2 py-1 bg-blue-500/20 hover:bg-blue-500/30 text-blue-700 dark:text-blue-300 rounded text-xs transition-colors"
                disabled={disabled}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
