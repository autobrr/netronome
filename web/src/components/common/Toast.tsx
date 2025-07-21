/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

// Simple toast notification utility
// TODO: Replace with a proper toast library or custom implementation

type ToastType = "success" | "error" | "info" | "warning";

export const showToast = (message: string, type: ToastType = "info") => {
  // For now, just log to console
  // In a real implementation, this would show a toast notification
  console.log(`[${type.toUpperCase()}] ${message}`);
  
  // You could also dispatch a custom event that a Toast component listens to
  const event = new CustomEvent("toast", {
    detail: { message, type }
  });
  window.dispatchEvent(event);
};