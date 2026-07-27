/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import { Toaster as Sonner, type ToasterProps } from "sonner";

const Toaster = ({ ...props }: ToasterProps) => {
  const [theme, setTheme] = useState<"light" | "dark">("light");

  useEffect(() => {
    // Check initial theme
    const isDark = document.documentElement.classList.contains("dark");
    setTheme(isDark ? "dark" : "light");

    // Watch for theme changes
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (
          mutation.type === "attributes" &&
          mutation.attributeName === "class"
        ) {
          const isDark = document.documentElement.classList.contains("dark");
          setTheme(isDark ? "dark" : "light");
        }
      });
    });

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });

    return () => observer.disconnect();
  }, []);

  return (
    <Sonner
      theme={theme}
      className="toaster group"
      style={
        {
          "--normal-bg": "var(--color-white, #ffffff)",
          "--normal-text": "var(--color-gray-900, #18181b)",
          "--normal-border": "var(--color-gray-200, #e4e4e7)",
          ...(theme === "dark" && {
            "--normal-bg":
              "color-mix(in srgb, var(--color-gray-800, #27272a) 50%, transparent)",
            "--normal-text": "var(--color-white, #ffffff)",
            "--normal-border":
              "color-mix(in srgb, var(--color-gray-700, #3f3f46) 50%, transparent)",
          }),
        } as React.CSSProperties
      }
      toastOptions={{
        classNames: {
          toast:
            "group toast bg-white text-gray-900 border border-gray-200 shadow-lg dark:bg-gray-800/50 dark:text-white dark:border dark:border-gray-700",
          description:
            "group-[.toast]:text-gray-600 dark:group-[.toast]:text-gray-400",
          closeButton:
            "group-[.toast]:bg-gray-100 group-[.toast]:border-gray-200 hover:group-[.toast]:bg-gray-200 dark:group-[.toast]:bg-gray-800 dark:group-[.toast]:border-gray-700 dark:hover:group-[.toast]:bg-gray-700",
          error:
            "group-[.toaster]:bg-red-50 group-[.toaster]:text-red-900 group-[.toaster]:border-red-200 dark:group-[.toaster]:bg-red-900/10 dark:group-[.toaster]:text-red-400 dark:group-[.toaster]:border-red-800",
          success:
            "group-[.toaster]:bg-emerald-50 group-[.toaster]:text-emerald-900 group-[.toaster]:border-emerald-200 dark:group-[.toaster]:bg-emerald-900/10 dark:group-[.toaster]:text-emerald-400 dark:group-[.toaster]:border-emerald-800",
          warning:
            "group-[.toaster]:bg-amber-50 group-[.toaster]:text-amber-900 group-[.toaster]:border-amber-200 dark:group-[.toaster]:bg-amber-900/10 dark:group-[.toaster]:text-amber-400 dark:group-[.toaster]:border-amber-800",
          info: "group-[.toaster]:bg-blue-50 group-[.toaster]:text-blue-900 group-[.toaster]:border-blue-200 dark:group-[.toaster]:bg-blue-900/10 dark:group-[.toaster]:text-blue-400 dark:group-[.toaster]:border-blue-800",
        },
        actionButtonStyle: {
          backgroundColor:
            theme === "dark"
              ? "var(--color-blue-700, #1d4ed8)"
              : "var(--color-blue-500, #3b82f6)",
          color: "var(--color-white, #ffffff)",
        },
        cancelButtonStyle: {
          backgroundColor:
            theme === "dark"
              ? "var(--color-gray-700, #3f3f46)"
              : "var(--color-gray-200, #e4e4e7)",
          color:
            theme === "dark"
              ? "var(--color-gray-100, #f4f4f5)"
              : "var(--color-gray-700, #3f3f46)",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
