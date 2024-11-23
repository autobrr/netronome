import { useEffect } from "react";
import { Outlet } from "@tanstack/react-router";
import { ArrowRightOnRectangleIcon } from "@heroicons/react/24/outline";
import { initializeDarkMode } from "./utils/darkMode";
import { useAuth } from "./context/auth";

function App() {
  const { isAuthenticated, logout } = useAuth();

  useEffect(() => {
    initializeDarkMode();
  }, []);

  return (
    <div className="min-h-screen bg-white dark:bg-gray-900">
      {isAuthenticated && (
        <button
          onClick={() => logout()}
          className="fixed top-4 right-4 text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-blue-300"
          aria-label="Logout"
        >
          <ArrowRightOnRectangleIcon className="h-6 w-6" />
        </button>
      )}
      <Outlet />
    </div>
  );
}

export default App;
