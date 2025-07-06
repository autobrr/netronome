/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { motion } from "motion/react";
import { toggleDarkMode } from "@/utils/darkMode";

export const DarkModeToggle: React.FC = () => {
  const [isToggling, setIsToggling] = useState(false);
  
  const handleToggle = () => {
    setIsToggling(true);
    toggleDarkMode();
    setTimeout(() => setIsToggling(false), 300);
  };
  
  return (
    <motion.button
      onClick={handleToggle}
      className="p-2 rounded-full hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-colors"
      aria-label="Toggle dark mode"
      whileHover={{ scale: 1.05 }}
      whileTap={{ scale: 0.95 }}
      disabled={isToggling}
    >
      <div className="relative w-6 h-6">
        {/* Sun Icon */}
        <motion.svg
          className="absolute inset-0 w-6 h-6 text-gray-600 dark:text-gray-400"
          fill="currentColor"
          viewBox="0 0 20 20"
          initial={false}
          animate={{
            scale: [1, 0.8, 1],
            rotate: isToggling ? 180 : 0,
            opacity: isToggling ? [1, 0, 1] : 1
          }}
          transition={{
            duration: 0.3,
            ease: "easeInOut"
          }}
          style={{
            display: 'block',
            visibility: 'visible'
          }}
        >
          <path
            fillRule="evenodd"
            d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z"
            className="hidden dark:block"
          />
          {/* Moon Icon */}
          <path 
            d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" 
            className="block dark:hidden"
          />
        </motion.svg>
      </div>
    </motion.button>
  );
};
