/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  createRouter,
  createRoute,
  createRootRoute,
  Outlet,
} from "@tanstack/react-router";
import App from "./App";
import SpeedTest from "./components/SpeedTest";
import Login from "./components/auth/Login";
import Register from "./components/auth/Register";
import { useAuth } from "./context/auth";
import { useEffect } from "react";

// Protected route wrapper component
function ProtectedRoute() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = router.navigate;

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate({ to: "/login" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return <Outlet />;
}

// Auth route wrapper component (for login/register)
function AuthRoute() {
  const { isAuthenticated, isLoading, checkRegistrationStatus } = useAuth();
  const location = window.location.pathname;
  const navigate = router.navigate;

  useEffect(() => {
    if (isAuthenticated) {
      navigate({ to: "/" });
      return;
    }

    const checkInitialStatus = async () => {
      if (location === "/register") {
        try {
          const status = await checkRegistrationStatus();
          if (status.hasUsers) {
            console.log("Users exist, redirecting to login...");
            navigate({ to: "/login" });
          }
        } catch (err) {
          console.error("Failed to check registration status:", err);
        }
        return;
      }

      try {
        console.log("AuthRoute: Checking registration status...");
        const status = await checkRegistrationStatus();
        console.log("AuthRoute: Status received:", status);

        if (!status.hasUsers) {
          console.log("AuthRoute: No users found, redirecting...");
          navigate({ to: "/register" });
        }
      } catch (err) {
        console.error("AuthRoute: Registration check failed:", err);
      }
    };

    if (!isLoading) {
      checkInitialStatus();
    }
  }, [isLoading, isAuthenticated, location, checkRegistrationStatus, navigate]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (isAuthenticated) {
    return null;
  }

  return <Outlet />;
}

// Root route with AuthProvider
const rootRoute = createRootRoute({
  component: () => <App />,
});

// Protected routes group
const protectedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "protected",
  component: ProtectedRoute,
});

// Auth routes group
const authRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "auth",
  component: AuthRoute,
});

// Login route
const loginRoute = createRoute({
  getParentRoute: () => authRoute,
  path: "/login",
  component: Login,
});

// Register route
const registerRoute = createRoute({
  getParentRoute: () => authRoute,
  path: "/register",
  component: Register,
});

// Protected index route
const indexRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: "/",
  component: SpeedTest,
});

const routeTree = rootRoute.addChildren([
  protectedRoute.addChildren([indexRoute]),
  authRoute.addChildren([loginRoute, registerRoute]),
]);

export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
