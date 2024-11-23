/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { router } from "../routes";
import * as authApi from "../api/auth";

interface User {
  id: number;
  username: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  checkRegistrationStatus: () => Promise<{
    registrationEnabled: boolean;
    hasUsers: boolean;
  }>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    verifySession();
  }, []);

  const verifySession = async () => {
    try {
      const response = await fetch("/api/auth/verify");
      if (response.status === 401) {
        setUser(null);
        setIsLoading(false);
        return;
      }
      const userData = await authApi.verifySession();
      setUser(userData);
    } catch (err) {
      // Only log unexpected errors
      if (err instanceof Error && !err.message.includes("401")) {
        console.error("Session verification failed:", err);
      }
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  const checkRegistrationStatus = async () => {
    return await authApi.checkRegistrationStatus();
  };

  const login = async (username: string, password: string) => {
    try {
      const status = await checkRegistrationStatus();
      if (status.registrationEnabled && !status.hasUsers) {
        router.navigate({ to: "/register" });
        return;
      }

      const userData = await authApi.login({ username, password });
      setUser(userData);
      setTimeout(() => {
        router.navigate({ to: "/" });
      }, 0);
    } catch (error) {
      console.error("Login failed:", error);
      throw error;
    }
  };

  const register = async (username: string, password: string) => {
    try {
      const userData = await authApi.register({ username, password });
      setUser(userData);
      setTimeout(() => {
        router.navigate({ to: "/" });
      }, 0);
    } catch (error) {
      console.error("Registration failed:", error);
      throw error;
    }
  };

  const logout = async () => {
    try {
      await authApi.logout();
      setUser(null);
      const status = await checkRegistrationStatus();
      setTimeout(() => {
        if (status.hasUsers) {
          router.navigate({ to: "/login" });
        } else {
          router.navigate({ to: "/register" });
        }
      }, 0);
    } catch (error) {
      console.error("Logout failed:", error);
      setTimeout(() => {
        router.navigate({ to: "/login" });
      }, 0);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
        checkRegistrationStatus,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
