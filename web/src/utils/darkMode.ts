/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export const toggleDarkMode = () => {
  // Add transition class before changing theme
  document.documentElement.classList.add('theme-transition');
  
  let newTheme: 'light' | 'dark';
  if (document.documentElement.classList.contains('dark')) {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('theme', 'light')
    newTheme = 'light';
  } else {
    document.documentElement.classList.add('dark')
    localStorage.setItem('theme', 'dark')
    newTheme = 'dark';
  }
  
  // Dispatch event to notify components
  window.dispatchEvent(new CustomEvent('themechange', { 
    detail: { 
      theme: newTheme,
      isSystemChange: false 
    } 
  }));
  
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
  const storedTheme = localStorage.getItem('theme');
  
  // If theme is set to 'auto' or not set at all, follow system preference
  if (!storedTheme || storedTheme === 'auto') {
    applyTheme(e.matches, true);
    
    // Dispatch a custom event to notify React components
    window.dispatchEvent(new CustomEvent('themechange', { 
      detail: { 
        theme: e.matches ? 'dark' : 'light',
        isSystemChange: true 
      } 
    }));
  }
};

export const initializeDarkMode = () => {
  
  // Add CSS for smooth transitions
  if (!document.getElementById('theme-transitions')) {
    const style = document.createElement('style');
    style.id = 'theme-transitions';
    style.textContent = `
      /* Main transition for theme switching */
      .theme-transition :not(::-webkit-scrollbar):not(::-webkit-scrollbar-track):not(::-webkit-scrollbar-thumb) {
        transition-property: background-color, border-color, color, fill, box-shadow;
        transition-duration: 0.3s;
        transition-timing-function: ease-in-out;
      }
      
      /* Prevent scrollbar transitions */
      .theme-transition ::-webkit-scrollbar,
      .theme-transition ::-webkit-scrollbar-track,
      .theme-transition ::-webkit-scrollbar-thumb,
      ::-webkit-scrollbar,
      ::-webkit-scrollbar-track,
      ::-webkit-scrollbar-thumb {
        transition: none !important;
      }
      
      /* Prevent scrollbar color from animating */
      html.theme-transition {
        scrollbar-color: initial !important;
      }
    `;
    document.head.appendChild(style);
  }
  
  // Get stored theme preference
  const storedTheme = localStorage.getItem('theme');
  
  // Set up system theme detection
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  
  // Apply initial theme
  if (storedTheme === 'dark' || storedTheme === 'light') {
    // User has explicit preference
    const isDark = storedTheme === 'dark';
    applyTheme(isDark);
    // Applied stored theme preference
  } else {
    // No preference or 'auto' - follow system
    const isDark = mediaQuery.matches;
    applyTheme(isDark);
    if (!storedTheme) {
      // Set to auto if nothing stored
      localStorage.setItem('theme', 'auto');
    }
    // Applied system theme
  }
  
  // Always listen for system theme changes
  try {
    mediaQuery.addEventListener('change', handleSystemThemeChange);
    // System theme listener registered
  } catch (e1) {
    try {
      // @ts-ignore - fallback for older browsers
      mediaQuery.addListener(handleSystemThemeChange);
      // System theme listener registered (legacy)
    } catch (e2) {
      // Failed to register system theme listener
    }
  }
  
  // System theme listener registered
};

// Export function to reset to system preference
export const resetToSystemTheme = () => {
  localStorage.setItem('theme', 'auto');
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  applyTheme(mediaQuery.matches, true);
  // Reset to system theme
};

// Export function to set theme to auto (follow system)
export const setAutoTheme = () => {
  localStorage.setItem('theme', 'auto');
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  applyTheme(mediaQuery.matches, true);
  // Set to auto theme
};

// Export function to check current system preference (useful for debugging)
export const getSystemTheme = () => {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

// Export function to check if user has manual preference
export const hasManualPreference = () => {
  const theme = localStorage.getItem('theme');
  return theme === 'dark' || theme === 'light';
};

// Get current theme mode
export const getCurrentThemeMode = (): 'dark' | 'light' | 'auto' => {
  const theme = localStorage.getItem('theme');
  if (theme === 'dark' || theme === 'light') {
    return theme;
  }
  return 'auto';
}; 