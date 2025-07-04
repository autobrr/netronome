/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

// Track if user has manually set a preference
let hasUserPreference = false;

export const toggleDarkMode = () => {
  // Mark that user has manually set a preference
  hasUserPreference = true;
  
  // Add transition class before changing theme
  document.documentElement.classList.add('theme-transition');
  
  if (document.documentElement.classList.contains('dark')) {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('theme', 'light')
  } else {
    document.documentElement.classList.add('dark')
    localStorage.setItem('theme', 'dark')
  }
  
  // Remove transition class after animation completes
  setTimeout(() => {
    document.documentElement.classList.remove('theme-transition');
  }, 300);
}

const applyTheme = (isDark: boolean, withTransition = false) => {
  if (withTransition) {
    document.documentElement.classList.add('theme-transition');
  }
  
  if (isDark) {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
  
  if (withTransition) {
    setTimeout(() => {
      document.documentElement.classList.remove('theme-transition');
    }, 300);
  }
};

const handleSystemThemeChange = (e: MediaQueryListEvent) => {
  // Only auto-switch if user hasn't manually set a preference
  if (!('theme' in localStorage)) {
    console.log('System theme changed to:', e.matches ? 'dark' : 'light');
    applyTheme(e.matches, true);
  } else {
    console.log('Ignoring system theme change - user has manual preference');
  }
};

export const initializeDarkMode = () => {
  console.log('Initializing dark mode');
  
  // Add CSS for smooth transitions
  if (!document.getElementById('theme-transitions')) {
    const style = document.createElement('style');
    style.id = 'theme-transitions';
    style.textContent = `
      .theme-transition,
      .theme-transition *,
      .theme-transition *:before,
      .theme-transition *:after {
        transition: background-color 0.3s ease-in-out,
                    border-color 0.3s ease-in-out,
                    color 0.3s ease-in-out,
                    fill 0.3s ease-in-out,
                    box-shadow 0.3s ease-in-out !important;
      }
    `;
    document.head.appendChild(style);
  }
  
  // Check if user has a saved preference
  const hasStoredTheme = 'theme' in localStorage;
  hasUserPreference = false; // Reset on initialization
  
  // Set up system theme detection
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  
  // Apply initial theme
  if (hasStoredTheme) {
    const isDark = localStorage.theme === 'dark';
    applyTheme(isDark);
    console.log('Applied stored theme:', isDark ? 'dark' : 'light');
  } else {
    const isDark = mediaQuery.matches;
    applyTheme(isDark);
    console.log('Applied system theme:', isDark ? 'dark' : 'light');
  }
  
  // Listen for system theme changes
  if (mediaQuery.addEventListener) {
    // Modern browsers
    mediaQuery.addEventListener('change', handleSystemThemeChange);
  } else if (mediaQuery.addListener) {
    // Fallback for older browsers (deprecated but still needed for some)
    // @ts-ignore - suppress deprecation warning for necessary fallback
    mediaQuery.addListener(handleSystemThemeChange);
  }
  
  console.log('System theme listener registered. Current system preference:', mediaQuery.matches ? 'dark' : 'light');
};

// Export function to reset to system preference
export const resetToSystemTheme = () => {
  localStorage.removeItem('theme');
  hasUserPreference = false;
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  applyTheme(mediaQuery.matches, true);
  console.log('Reset to system theme:', mediaQuery.matches ? 'dark' : 'light');
};

// Export function to check current system preference (useful for debugging)
export const getSystemTheme = () => {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

// Export function to check if user has manual preference
export const hasManualPreference = () => {
  return 'theme' in localStorage;
}; 