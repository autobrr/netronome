/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState } from "react";
import { Outlet } from "@tanstack/react-router";
import { ArrowRightStartOnRectangleIcon as LogoutIcon } from "@heroicons/react/24/outline";
import { HeartIcon } from "@heroicons/react/24/solid";
import { initializeDarkMode } from "@/utils/darkMode";
import { useAuth } from "@/context/auth";
import { DonateModal } from "@/components/DonateModal";

function App() {
  const { isAuthenticated, logout } = useAuth();
  const [isDonateOpen, setIsDonateOpen] = useState(false);

  useEffect(() => {
    initializeDarkMode();
  }, []);

  return (
    <div className="min-h-screen bg-white pattern dark:bg-gray-900 relative">
      {isAuthenticated && (
        <div className="absolute z-10 top-4 right-4 flex items-center gap-4">
          <button
            onClick={() => setIsDonateOpen(true)}
            className="text-red-500 hover:text-red-600 dark:text-red-500/50 dark:hover:text-red-500 transition-colors"
            aria-label="Donate"
          >
            <HeartIcon className="h-6 w-6" />
          </button>
          <button
            onClick={() => logout()}
            className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-blue-400"
            aria-label="Logout"
          >
            <LogoutIcon className="h-6 w-6" />
          </button>
        </div>
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
