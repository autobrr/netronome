/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import { Outlet } from "@tanstack/react-router";
import { ArrowRightStartOnRectangleIcon as LogoutIcon } from "@heroicons/react/24/outline";
import { HeartIcon } from "@heroicons/react/24/solid";
import { initializeDarkMode } from "@/utils/darkMode";
import { useAuth } from "@/context/auth";
import { DonateModal } from "@/components/DonateModal";
import { DarkModeToggle } from "@/components/DarkModeToggle";
import logo from "@/assets/logo_small.png";

function App() {
  const { isAuthenticated, logout } = useAuth();
  const [isDonateOpen, setIsDonateOpen] = useState(false);

  useEffect(() => {
    initializeDarkMode();
  }, []);

  return (
    <div className="min-h-screen bg-white pattern dark:bg-gray-900 relative transition-colors duration-300 ease-in-out">
      {isAuthenticated && (
        <>
          {/* Logo in upper left */}
          <div className="absolute z-10 top-4 left-4 flex items-center gap-3">
            <img
              src={logo}
              alt="netronome Logo"
              className="h-12 w-12 select-none pointer-events-none"
              draggable="false"
            />
            <div>
              <h1 className="text-3xl font-bold text-gray-900 dark:text-white select-none">
                Netronome
              </h1>
              <h2 className="text-sm font-medium text-gray-600 dark:text-gray-300 select-none">
                Network Performance Testing
              </h2>
            </div>
          </div>
          
          {/* Controls in upper right */}
          <div className="absolute z-10 top-4 right-4 flex items-center gap-4">
            <button
              onClick={() => setIsDonateOpen(true)}
              className="text-red-500 hover:text-red-600 dark:text-red-500/50 dark:hover:text-red-500 transition-colors"
              aria-label="Donate"
            >
              <HeartIcon className="h-6 w-6" />
            </button>
            <DarkModeToggle />
            <button
              onClick={() => logout()}
              className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-blue-400"
              aria-label="Logout"
            >
              <LogoutIcon className="h-6 w-6" />
            </button>
          </div>
        </>
      )}

      <DonateModal
        isOpen={isDonateOpen}
        onClose={() => setIsDonateOpen(false)}
      />

      <Outlet />
    </div>
  );
}

export default App;
