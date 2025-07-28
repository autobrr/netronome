/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { toast } from "sonner";
import {
  CheckCircleIcon,
  XCircleIcon,
  InformationCircleIcon,
  ExclamationTriangleIcon,
} from "@heroicons/react/24/outline";

type ToastType = "success" | "error" | "info" | "warning";

interface ToastOptions {
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  duration?: number;
}

const toastIcons = {
  success: <CheckCircleIcon className="h-5 w-5 text-emerald-500" />,
  error: <XCircleIcon className="h-5 w-5 text-red-500" />,
  info: <InformationCircleIcon className="h-5 w-5 text-blue-500" />,
  warning: <ExclamationTriangleIcon className="h-5 w-5 text-amber-500" />,
};

export const showToast = (
  message: string,
  type: ToastType = "info",
  options?: ToastOptions
) => {
  const icon = toastIcons[type];

  // Map our toast types to Sonner's API
  switch (type) {
    case "success":
      toast.success(message, {
        icon,
        description: options?.description,
        action: options?.action,
        duration: options?.duration ?? 4000,
      });
      break;
    case "error":
      toast.error(message, {
        icon,
        description: options?.description,
        action: options?.action,
        duration: options?.duration ?? 6000,
      });
      break;
    case "warning":
      toast.warning(message, {
        icon,
        description: options?.description,
        action: options?.action,
        duration: options?.duration ?? 5000,
      });
      break;
    case "info":
    default:
      toast.info(message, {
        icon,
        description: options?.description,
        action: options?.action,
        duration: options?.duration ?? 4000,
      });
  }
};

// Export additional toast methods for direct usage
export { toast };