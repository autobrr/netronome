/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { Outlet } from "@tanstack/react-router";
import {
  ArrowRightStartOnRectangleIcon as LogoutIcon,
  ArrowTopRightOnSquareIcon,
} from "@heroicons/react/24/outline";
import { HeartIcon } from "@heroicons/react/24/solid";
import {
  Bars3Icon,
  ChevronRightIcon,
  SunIcon,
  MoonIcon,
  ComputerDesktopIcon,
} from "@heroicons/react/24/outline";
import {
  initializeDarkMode,
  getCurrentThemeMode,
  setThemeMode,
} from "@/utils/darkMode";
import { useAuth } from "@/context/auth";
import { DonateModal } from "@/components/DonateModal";
import { ThemeDropdown } from "@/components/ThemeDropdown";
import {
  SettingsMenu,
  SettingsDialog,
  settingsSections,
} from "@/components/SettingsMenu";
import { Button } from "@/components/ui/Button";
import { Toaster } from "@/components/ui/sonner";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import logo from "@/assets/logo_small.png";
import { PWAUpdatePrompt } from "@/components/PWAUpdatePrompt";
import {
  formatReleaseTag,
  getLatestRelease,
  latestReleaseQueryKey,
} from "@/api/version";
import { AppUpdatePrompt } from "@/components/AppUpdatePrompt";

const RELEASE_POLL_INTERVAL = 2 * 60 * 1000;

function App() {
  const { isAuthenticated, logout } = useAuth();
  const { data: latestRelease } = useQuery({
    queryKey: latestReleaseQueryKey,
    queryFn: getLatestRelease,
    enabled: isAuthenticated,
    staleTime: RELEASE_POLL_INTERVAL,
    refetchInterval: RELEASE_POLL_INTERVAL,
    refetchOnWindowFocus: false,
  });
  const [isDonateOpen, setIsDonateOpen] = useState(false);
  const [mobileSettingsSection, setMobileSettingsSection] = useState<
    string | null
  >(null);
  const [currentTheme, setCurrentTheme] = useState<"light" | "dark" | "auto">(
    getCurrentThemeMode()
  );
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    initializeDarkMode();

    // Listen for theme changes from other sources (like desktop toggle)
    const handleThemeChangeEvent = () => {
      setCurrentTheme(getCurrentThemeMode());
    };

    window.addEventListener("themechange", handleThemeChangeEvent);
    return () => {
      window.removeEventListener("themechange", handleThemeChangeEvent);
    };
  }, []);

  // setThemeMode dispatches "themechange", which the effect above turns into
  // a setCurrentTheme - no local state update needed here.
  const handleThemeChange = useCallback((theme: "light" | "dark" | "auto") => {
    setThemeMode(theme);
  }, []);

  const getThemeIcon = (theme: "light" | "dark" | "auto") => {
    switch (theme) {
      case "light":
        return <SunIcon className="h-5 w-5" />;
      case "dark":
        return <MoonIcon className="h-5 w-5" />;
      case "auto":
        return <ComputerDesktopIcon className="h-5 w-5" />;
    }
  };

  // Check if we're on the public route
  const isPublicRoute = window.location.pathname.includes("/public");
  const showHeader = isAuthenticated || isPublicRoute;
  const availableRelease = isAuthenticated ? latestRelease ?? null : null;

  return (
    <div className="min-h-screen bg-white pattern dark:bg-gray-900 relative transition-colors duration-300 ease-in-out">
      {/* Show header for authenticated users OR public route */}
      {showHeader && (
        <>
          {/* Logo in upper left - more compact on mobile */}
          <div className="absolute z-10 top-3 sm:top-4 left-3 sm:left-4 flex items-center gap-2 sm:gap-3">
            <img
              src={logo}
              alt="netronome Logo"
              className="h-10 w-10 sm:h-12 sm:w-12 select-none pointer-events-none"
              draggable="false"
            />
            <div>
              <h1 className="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-white select-none">
                Netronome
              </h1>
              <h2 className="hidden sm:block text-sm font-medium text-gray-600 dark:text-gray-300 select-none">
                Network Performance Testing
              </h2>
            </div>
          </div>
        </>
      )}

      {/* Only show controls for authenticated users */}
      {isAuthenticated && (
        <>
          {/* Desktop controls */}
          <div className="hidden sm:flex absolute z-10 top-4 right-4 items-center gap-2">
            <Button
              onClick={() => setIsDonateOpen(true)}
              variant="ghost"
              size="icon"
              className="text-red-500 hover:text-red-600 dark:text-red-500/50 dark:hover:text-red-500"
              aria-label="Support Netronome"
            >
              <HeartIcon className="h-6 w-6" />
            </Button>
            <ThemeDropdown />
            <SettingsMenu release={availableRelease} />
            <Button
              onClick={() => logout()}
              variant="ghost"
              size="icon"
              className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-blue-400"
              aria-label="Logout"
            >
              <LogoutIcon className="h-6 w-6" />
            </Button>
          </div>

          {/* Mobile hamburger menu */}
          <div className="sm:hidden absolute z-10 top-3 right-3">
            <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
              <SheetTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="relative text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
                  aria-label={
                    availableRelease
                      ? `Open menu, Netronome ${formatReleaseTag(availableRelease.tag_name)} update available`
                      : "Open menu"
                  }
                >
                  <Bars3Icon className="h-6 w-6" />
                  {availableRelease && (
                    <span
                      aria-hidden="true"
                      className="absolute right-1 top-1 h-2 w-2 rounded-full bg-blue-500 ring-2 ring-white dark:ring-gray-900"
                    />
                  )}
                </Button>
              </SheetTrigger>
              <SheetContent
                side="right"
                className="w-72 bg-white dark:bg-gray-900 border-l border-gray-200 dark:border-gray-800"
              >
                <div className="flex flex-col h-full">
                  <SheetHeader className="pb-6 border-b border-gray-200 dark:border-gray-800">
                    <SheetTitle className="text-lg font-semibold text-gray-900 dark:text-white">
                      Settings
                    </SheetTitle>
                  </SheetHeader>

                  <div className="flex-1 py-6">
                    <div className="space-y-2">
                      {availableRelease && (
                        <Button
                          asChild
                          variant="ghost"
                          className="h-auto w-full justify-start gap-3 px-4 py-3"
                        >
                          <a
                            href={availableRelease.html_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={() => setMobileMenuOpen(false)}
                            aria-label={`View Netronome ${formatReleaseTag(availableRelease.tag_name)} release`}
                          >
                            <ArrowTopRightOnSquareIcon className="h-5 w-5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
                            <span className="flex-1 text-left text-sm font-medium text-gray-700 dark:text-gray-300">
                              Netronome {formatReleaseTag(availableRelease.tag_name)} available
                            </span>
                            <span className="text-xs text-blue-600 dark:text-blue-400">
                              View release
                            </span>
                          </a>
                        </Button>
                      )}

                      {/* Support */}
                      <Button
                        onClick={() => {
                          setIsDonateOpen(true);
                          setMobileMenuOpen(false);
                        }}
                        variant="ghost"
                        className="w-full justify-start gap-3 h-auto px-4 py-3"
                      >
                        <HeartIcon className="h-5 w-5 text-red-500 flex-shrink-0" />
                        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                          Support Development
                        </span>
                      </Button>

                      {/* Theme mode - compact segmented toggle */}
                      <div className="px-4 py-3">
                        <h3 className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-3">
                          Theme
                        </h3>
                        <div className="flex gap-1 rounded-lg bg-gray-100 dark:bg-gray-800 p-1">
                          {(["light", "dark", "auto"] as const).map((theme) => (
                            <Button
                              key={theme}
                              onClick={() => handleThemeChange(theme)}
                              variant="ghost"
                              aria-label={theme === "auto" ? "System" : theme}
                              className={`flex-1 h-9 ${
                                currentTheme === theme
                                  ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm hover:bg-white dark:hover:bg-gray-700"
                                  : "text-gray-600 dark:text-gray-400"
                              }`}
                            >
                              {getThemeIcon(theme)}
                            </Button>
                          ))}
                        </div>
                      </div>

                      {/* Settings sections - same dialogs as desktop */}
                      {settingsSections.map((section) => (
                        <Button
                          key={section.id}
                          onClick={() => {
                            setMobileSettingsSection(section.id);
                            setMobileMenuOpen(false);
                          }}
                          variant="ghost"
                          className="w-full justify-start gap-3 h-auto px-4 py-3"
                        >
                          <span className="text-gray-600 dark:text-gray-400 flex-shrink-0">
                            {section.icon}
                          </span>
                          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                            {section.label}
                          </span>
                          <ChevronRightIcon className="w-4 h-4 ml-auto text-gray-400" />
                        </Button>
                      ))}
                    </div>
                  </div>

                  {/* Logout */}
                  <div className="pt-6 border-t border-gray-200 dark:border-gray-800">
                    <Button
                      onClick={() => logout()}
                      variant="ghost"
                      className="w-full justify-start gap-3 h-auto px-4 py-3 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-600 dark:text-red-500 hover:text-red-600 dark:hover:text-red-500"
                    >
                      <LogoutIcon className="h-5 w-5 flex-shrink-0" />
                      <span className="text-sm font-medium">Sign Out</span>
                    </Button>
                  </div>
                </div>
              </SheetContent>
            </Sheet>
          </div>
        </>
      )}

      <DonateModal
        isOpen={isDonateOpen}
        onClose={() => setIsDonateOpen(false)}
      />

      {/* Mobile settings dialog - same sections/dialog as desktop */}
      <SettingsDialog
        sectionId={mobileSettingsSection}
        onClose={() => setMobileSettingsSection(null)}
      />

      <Outlet />
      <Toaster position="bottom-right" />
      <AppUpdatePrompt release={availableRelease} />
      <PWAUpdatePrompt />
    </div>
  );
}

export default App;
