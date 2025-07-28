/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { motion, AnimatePresence } from "motion/react";
import {
  getCurrentThemeMode,
  setAutoTheme,
  getSystemTheme,
} from "@/utils/darkMode";
import {
  SunIcon,
  ComputerDesktopIcon,
  CheckIcon,
} from "@heroicons/react/24/outline";
import { MoonIcon } from "@heroicons/react/24/solid";
import { Button } from "@/components/ui/button";

// Constants
const THEME_STORAGE_KEY = "theme";
const THEME_CHANGE_EVENT = "themechange";
const STORAGE_EVENT = "storage";
const MEDIA_QUERY = "(prefers-color-scheme: dark)";

const DROPDOWN_ANIMATION = {
  initial: { opacity: 0, y: -10, scale: 0.95 },
  animate: { opacity: 1, y: 0, scale: 1 },
  exit: { opacity: 0, y: -10, scale: 0.95 },
  transition: {
    duration: 0.15,
    ease: [0.23, 1, 0.32, 1] as const,
  },
} as const;

const CHECK_ANIMATION = {
  duration: 0.15,
} as const;

// Types
type ThemeMode = "light" | "dark" | "auto";

interface ThemeOption {
  value: ThemeMode;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

// Theme options configuration
const themeOptions: ThemeOption[] = [
  { value: "light", label: "Light", icon: SunIcon },
  { value: "dark", label: "Dark", icon: MoonIcon },
  { value: "auto", label: "System", icon: ComputerDesktopIcon },
];

// Custom hook for theme change detection
const useThemeChange = () => {
  const [currentMode, setCurrentMode] = useState<ThemeMode>(
    getCurrentThemeMode()
  );

  const checkTheme = useCallback(() => {
    const newMode = getCurrentThemeMode();
    setCurrentMode(newMode);
  }, []);

  useEffect(() => {
    // Listen for storage events (theme changes from other tabs)
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === THEME_STORAGE_KEY) {
        checkTheme();
      }
    };

    // Listen for custom theme change events
    const handleThemeChange = () => {
      checkTheme();
    };

    // Listen for system theme changes
    const mediaQuery = window.matchMedia(MEDIA_QUERY);
    const handleMediaQueryChange = () => {
      const storedTheme = localStorage.getItem(THEME_STORAGE_KEY);
      // Only update if in auto mode
      if (!storedTheme || storedTheme === "auto") {
        checkTheme();
      }
    };

    // Add event listeners
    window.addEventListener(STORAGE_EVENT, handleStorageChange);
    window.addEventListener(THEME_CHANGE_EVENT, handleThemeChange);
    mediaQuery.addEventListener("change", handleMediaQueryChange);

    return () => {
      window.removeEventListener(STORAGE_EVENT, handleStorageChange);
      window.removeEventListener(THEME_CHANGE_EVENT, handleThemeChange);
      mediaQuery.removeEventListener("change", handleMediaQueryChange);
    };
  }, [checkTheme]);

  return currentMode;
};

// Custom hook for click outside detection
const useClickOutside = <T extends HTMLElement = HTMLElement>(
  ref: React.RefObject<T | null>,
  handler: () => void
) => {
  useEffect(() => {
    const listener = (event: MouseEvent) => {
      if (!ref.current || ref.current.contains(event.target as Node)) {
        return;
      }
      handler();
    };

    document.addEventListener("mousedown", listener);
    return () => {
      document.removeEventListener("mousedown", listener);
    };
  }, [ref, handler]);
};

export const DarkModeToggle: React.FC = () => {
  const [showMenu, setShowMenu] = useState(false);
  const currentMode = useThemeChange();
  const menuRef = useRef<HTMLDivElement>(null);
  
  // Close menu when clicking outside
  useClickOutside(menuRef, () => {
    if (showMenu) setShowMenu(false);
  });

  const handleModeSelect = useCallback((mode: ThemeMode) => {
    if (mode === "auto") {
      setAutoTheme();
    } else if (mode === "light") {
      localStorage.setItem(THEME_STORAGE_KEY, "light");
      document.documentElement.classList.remove("dark");
    } else {
      localStorage.setItem(THEME_STORAGE_KEY, "dark");
      document.documentElement.classList.add("dark");
    }

    setShowMenu(false);

    // Dispatch event to notify other components
    window.dispatchEvent(
      new CustomEvent(THEME_CHANGE_EVENT, {
        detail: {
          theme: mode === "auto" ? getSystemTheme() : mode,
          isSystemChange: false,
        },
      })
    );
  }, []);

  const toggleMenuOpen = useCallback(() => {
    setShowMenu((prev) => !prev);
  }, []);

  // Determine which icon to show
  const CurrentIcon = useMemo(() => {
    if (currentMode === "auto") {
      return ComputerDesktopIcon;
    }
    return currentMode === "dark" ? MoonIcon : SunIcon;
  }, [currentMode]);

  return (
    <div className="relative" ref={menuRef}>
      {/* Icon button that opens the menu */}
      <Button
        onClick={toggleMenuOpen}
        variant="ghost"
        size="icon"
        className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400"
        aria-label="Theme options"
        aria-expanded={showMenu}
        aria-haspopup="true"
      >
        <CurrentIcon className="w-6 h-6" />
      </Button>

      {/* Dropdown menu */}
      <AnimatePresence>
        {showMenu && (
          <motion.div
            {...DROPDOWN_ANIMATION}
            className="absolute right-0 mt-2 w-40 rounded-xl bg-white dark:bg-gray-800 shadow-xl ring-1 ring-black/10 dark:ring-white/10 overflow-hidden z-50"
            role="menu"
            aria-orientation="vertical"
          >
            <div className="p-1">
              {themeOptions.map((option) => {
                const Icon = option.icon;
                const isSelected = currentMode === option.value;

                return (
                  <ThemeOptionButton
                    key={option.value}
                    option={option}
                    isSelected={isSelected}
                    Icon={Icon}
                    onSelect={handleModeSelect}
                  />
                );
              })}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

// Sub-component for theme option buttons
interface ThemeOptionButtonProps {
  option: ThemeOption;
  isSelected: boolean;
  Icon: React.ComponentType<{ className?: string }>;
  onSelect: (value: ThemeMode) => void;
}

const ThemeOptionButton: React.FC<ThemeOptionButtonProps> = React.memo(
  ({ option, isSelected, Icon, onSelect }) => {
    const handleClick = useCallback(() => {
      onSelect(option.value);
    }, [onSelect, option.value]);

    return (
      <Button
        onClick={handleClick}
        variant="ghost"
        className={`
          w-full px-3 py-2 text-sm justify-start gap-3 h-auto
          ${
            isSelected
              ? "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 hover:bg-blue-100 dark:hover:bg-blue-900/30"
              : "hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
          }
        `}
        role="menuitem"
        aria-selected={isSelected}
      >
        <Icon
          className={`w-4 h-4 ${
            isSelected ? "text-blue-600 dark:text-blue-400" : ""
          }`}
        />
        <span className="flex-1 text-left font-medium">{option.label}</span>
        <motion.div
          initial={false}
          animate={{
            opacity: isSelected ? 1 : 0,
            scale: isSelected ? 1 : 0.8,
          }}
          transition={CHECK_ANIMATION}
        >
          <CheckIcon className="w-4 h-4 text-blue-600 dark:text-blue-400" />
        </motion.div>
      </Button>
    );
  }
);

ThemeOptionButton.displayName = "ThemeOptionButton";
