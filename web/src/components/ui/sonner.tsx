/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
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
        if (mutation.type === "attributes" && mutation.attributeName === "class") {
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
      style={{
        "--normal-bg": "rgb(255 255 255)",
        "--normal-text": "rgb(17 24 39)",
        "--normal-border": "rgb(228 228 231)",
        ...theme === "dark" && {
          "--normal-bg": "rgb(39 39 42 / 0.5)",
          "--normal-text": "rgb(255 255 255)",
          "--normal-border": "rgb(63 63 70 / 0.5)",
        },
      } as React.CSSProperties}
      toastOptions={{
        classNames: {
          toast:
            "group toast bg-white text-gray-900 border border-gray-200 shadow-lg dark:bg-gray-800/50 dark:text-white dark:border dark:border-gray-700",
          description: "group-[.toast]:text-gray-600 dark:group-[.toast]:text-gray-400",
          actionButton:
            "group-[.toast]:bg-blue-500 group-[.toast]:text-white hover:group-[.toast]:bg-blue-600",
          cancelButton:
            "group-[.toast]:bg-gray-200/50 group-[.toast]:text-gray-700 hover:group-[.toast]:bg-gray-300/50 dark:group-[.toast]:bg-gray-800/50 dark:group-[.toast]:text-gray-300 dark:hover:group-[.toast]:bg-gray-700/50",
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
      }}
      {...props}
    />
  );
};

export { Toaster };