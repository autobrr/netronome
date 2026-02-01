/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { motion, AnimatePresence } from "motion/react";
import { useTranslation } from "react-i18next";
import { GlobeAltIcon, CheckIcon } from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/Button";

// Constants
const LANGUAGE_CHANGE_EVENT = "languagechange";

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
type LanguageCode = "en" | "fr";

interface LanguageOption {
  code: LanguageCode;
  label: string;
  flag: string;
}

// Language options configuration
const languageOptions: LanguageOption[] = [
  { code: "en", label: "English", flag: "🇬🇧" },
  { code: "fr", label: "Français", flag: "🇫🇷" },
];

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

export const LanguageSelector: React.FC = () => {
  const { i18n } = useTranslation();
  const [showMenu, setShowMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  
  // Close menu when clicking outside
  useClickOutside(menuRef, () => {
    if (showMenu) setShowMenu(false);
  });

  const currentLanguage = useMemo(() => {
    return i18n.language.startsWith("fr") ? "fr" : "en";
  }, [i18n.language]);

  const handleLanguageSelect = useCallback((langCode: LanguageCode) => {
    i18n.changeLanguage(langCode);
    setShowMenu(false);

    // Dispatch event to notify other components
    window.dispatchEvent(
      new CustomEvent(LANGUAGE_CHANGE_EVENT, {
        detail: { language: langCode },
      })
    );
  }, [i18n]);

  const toggleMenuOpen = useCallback(() => {
    setShowMenu((prev) => !prev);
  }, []);

  return (
    <div className="relative" ref={menuRef}>
      {/* Icon button that opens the menu */}
      <Button
        onClick={toggleMenuOpen}
        variant="ghost"
        size="icon"
        className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400"
        aria-label="Language options"
        aria-expanded={showMenu}
        aria-haspopup="true"
      >
        <GlobeAltIcon className="w-6 h-6" />
      </Button>

      {/* Dropdown menu */}
      <AnimatePresence>
        {showMenu && (
          <motion.div
            {...DROPDOWN_ANIMATION}
            className="absolute right-0 mt-2 w-48 rounded-xl bg-white dark:bg-gray-800 shadow-xl ring-1 ring-black/10 dark:ring-white/10 overflow-hidden z-50"
            role="menu"
            aria-orientation="vertical"
          >
            <div className="p-1">
              {languageOptions.map((option) => {
                const isSelected = currentLanguage === option.code;

                return (
                  <LanguageOptionButton
                    key={option.code}
                    option={option}
                    isSelected={isSelected}
                    onSelect={handleLanguageSelect}
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

// Sub-component for language option buttons
interface LanguageOptionButtonProps {
  option: LanguageOption;
  isSelected: boolean;
  onSelect: (code: LanguageCode) => void;
}

const LanguageOptionButton: React.FC<LanguageOptionButtonProps> = React.memo(
  ({ option, isSelected, onSelect }) => {
    const handleClick = useCallback(() => {
      onSelect(option.code);
    }, [onSelect, option.code]);

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
        <span className="text-lg">{option.flag}</span>
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

LanguageOptionButton.displayName = "LanguageOptionButton";
