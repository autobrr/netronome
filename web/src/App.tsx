/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import { Outlet } from "@tanstack/react-router";
import { ArrowRightStartOnRectangleIcon as LogoutIcon } from "@heroicons/react/24/outline";
import { initializeDarkMode } from "@/utils/darkMode";
import { useAuth } from "@/context/auth";

function App() {
  const { isAuthenticated, logout } = useAuth();

  useEffect(() => {
    initializeDarkMode();
  }, []);

  return (
    <div className="min-h-screen bg-white pattern dark:bg-gray-900 relative">
      {isAuthenticated && (
        <button
          onClick={() => logout()}
          className="absolute z-10 top-4 right-4 text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-blue-400"
          aria-label="Logout"
        >
          <LogoutIcon className="h-6 w-6" />
        </button>
      )}
      <Outlet />
    </div>
  );
}

export default App;
