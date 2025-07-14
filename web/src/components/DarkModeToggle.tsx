/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useRef, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import {
  getCurrentThemeMode,
  setAutoTheme,
  getSystemTheme,
  toggleDarkMode,
} from "@/utils/darkMode";
import {
  SunIcon,
  ComputerDesktopIcon,
  ChevronDownIcon,
  CheckIcon,
} from "@heroicons/react/24/outline";
import { MoonIcon } from "@heroicons/react/24/solid";

type ThemeMode = "light" | "dark" | "auto";

interface ThemeOption {
  value: ThemeMode;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

const themeOptions: ThemeOption[] = [
  { value: "light", label: "Light", icon: SunIcon },
  { value: "dark", label: "Dark", icon: MoonIcon },
  { value: "auto", label: "System", icon: ComputerDesktopIcon },
];

export const DarkModeToggle: React.FC = () => {
  const [showMenu, setShowMenu] = useState(false);
  const [currentMode, setCurrentMode] = useState<ThemeMode>(
    getCurrentThemeMode()
  );
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Update current mode when theme changes
    const checkTheme = () => {
      const newMode = getCurrentThemeMode();
      setCurrentMode(newMode);
    };

    // Listen for storage events (theme changes from other tabs)
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "theme") {
        checkTheme();
      }
    };
    window.addEventListener("storage", handleStorageChange);

    // Listen for system theme changes
    const handleThemeChange = (event: Event) => {
      checkTheme();
    };

    window.addEventListener("themechange", handleThemeChange);

    // Also listen directly to media query changes for auto mode
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleMediaQueryChange = (e: MediaQueryListEvent) => {
      const storedTheme = localStorage.getItem("theme");
      // Check if auto mode
      if (!storedTheme || storedTheme === "auto") {
        checkTheme();
      }
    };

    try {
      mediaQuery.addEventListener("change", handleMediaQueryChange);
    } catch (e) {
      // Fallback for older browsers
      // @ts-ignore
      mediaQuery.addListener(handleMediaQueryChange);
    }

    return () => {
      window.removeEventListener("storage", handleStorageChange);
      window.removeEventListener("themechange", handleThemeChange);
      try {
        mediaQuery.removeEventListener("change", handleMediaQueryChange);
      } catch (e) {
        // @ts-ignore
        mediaQuery.removeListener(handleMediaQueryChange);
      }
    };
  }, []);

  useEffect(() => {
    // Close menu when clicking outside
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setShowMenu(false);
      }
    };

    if (showMenu) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showMenu]);

  const handleModeSelect = (mode: ThemeMode) => {
    if (mode === "auto") {
      setAutoTheme();
    } else if (mode === "light") {
      localStorage.setItem("theme", "light");
      document.documentElement.classList.remove("dark");
    } else {
      localStorage.setItem("theme", "dark");
      document.documentElement.classList.add("dark");
    }

    setCurrentMode(mode);
    setShowMenu(false);

    // Dispatch event to notify components
    window.dispatchEvent(
      new CustomEvent("themechange", {
        detail: {
          theme: mode === "auto" ? getSystemTheme() : mode,
          isSystemChange: false,
        },
      })
    );
  };

  const getCurrentIcon = () => {
    if (currentMode === "auto") {
      // Show the actual theme icon when in auto mode
      const systemTheme = getSystemTheme();
      return systemTheme === "dark" ? MoonIcon : SunIcon;
    }
    return currentMode === "dark" ? MoonIcon : SunIcon;
  };

  const CurrentIcon = getCurrentIcon();

  return (
    <div className="relative" ref={menuRef}>
      <div className="flex items-center">
        {/* Icon button for quick toggle */}
        <button
          onClick={() => {
            if (currentMode === "auto") {
              // If in auto mode, switch to manual mode with opposite of current theme
              handleModeSelect(getSystemTheme() === "dark" ? "light" : "dark");
            } else {
              // Simple toggle between light and dark
              toggleDarkMode();
              setCurrentMode(getCurrentThemeMode());
            }
          }}
          className="p-2 text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400 transition-colors"
          aria-label="Toggle theme"
        >
          <CurrentIcon className="w-6 h-6" />
        </button>

        {/* Dropdown button */}
        <button
          onClick={() => setShowMenu(!showMenu)}
          className="p-1 text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400 transition-colors"
          aria-label="Theme options"
        >
          <ChevronDownIcon
            className={`w-4 h-4 transition-transform duration-200 ${
              showMenu ? "rotate-180" : ""
            }`}
          />
        </button>
      </div>

      {/* Dropdown menu */}
      <AnimatePresence>
        {showMenu && (
          <motion.div
            initial={{ opacity: 0, y: -10, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -10, scale: 0.95 }}
            transition={{
              duration: 0.15,
              ease: [0.23, 1, 0.32, 1],
            }}
            className="absolute right-0 mt-2 w-40 rounded-xl bg-white dark:bg-gray-800 shadow-xl ring-1 ring-black/10 dark:ring-white/10 overflow-hidden z-50"
          >
            <div className="p-1">
              {themeOptions.map((option) => {
                const Icon = option.icon;
                const isSelected = currentMode === option.value;

                return (
                  <motion.button
                    key={option.value}
                    onClick={() => handleModeSelect(option.value)}
                    className={`
                      w-full px-3 py-2 text-sm rounded-lg flex items-center gap-3
                      transition-all duration-200
                      ${
                        isSelected
                          ? "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
                          : "hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
                      }
                    `}
                  >
                    <Icon
                      className={`w-4 h-4 ${
                        isSelected ? "text-blue-600 dark:text-blue-400" : ""
                      }`}
                    />
                    <span className="flex-1 text-left font-medium">
                      {option.label}
                    </span>
                    <motion.div
                      initial={false}
                      animate={{
                        opacity: isSelected ? 1 : 0,
                        scale: isSelected ? 1 : 0.8,
                      }}
                      transition={{ duration: 0.15 }}
                    >
                      <CheckIcon className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                    </motion.div>
                  </motion.button>
                );
              })}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
