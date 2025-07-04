/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { useAuth } from "@/context/auth";
import { checkRegistrationStatus } from "@/api/auth";
import { router } from "@/routes";
import logo from "@/assets/logo_small.png";
import { Footer } from "@/components/Footer";

export default function Register() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const { register } = useAuth();

  useEffect(() => {
    const checkAccess = async () => {
      try {
        const allowed = await checkRegistrationStatus();
        if (!allowed) {
          router.navigate({ to: "/login" });
        }
      } catch {
        router.navigate({ to: "/login" });
      } finally {
        setIsLoading(false);
      }
    };
    checkAccess();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      if (password.length < 3) {
        setError("Password must be at least 4 characters");
        return;
      }

      await register(username, password);
      router.navigate({ to: "/login" });
    } catch (err) {
      if (err instanceof Error) {
        const message = err.message.includes("Error #01:")
          ? err.message.split("Error #01:")[1].trim()
          : err.message;
        setError(message);
      } else {
        setError("Registration failed");
      }
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-white dark:bg-gray-900">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-white dark:bg-gray-900 pattern">
      <div className="max-w-md w-full px-8 pt-8 pb-4 bg-white/95 dark:bg-gray-850/40 border border-gray-200 dark:border-black/40 rounded-lg shadow-lg">
        <div className="flex flex-col items-center">
          <img
            src={logo}
            alt="Netronome Logo"
            className="text-white h-16 w-16 mb-2 select-none pointer-events-none"
          />
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white pointer-events-none select-none">
            Netronome
          </h2>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">Create your account</p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          {error && (
            <div className="bg-red-100 dark:bg-red-500 dark:bg-opacity-10 border border-red-400 dark:border-red-500 text-red-700 dark:text-red-500 px-4 py-3 rounded">
              <span className="block sm:inline">{error}</span>
            </div>
          )}

          <div className="rounded-md shadow-sm -space-y-px">
            <div>
              <label htmlFor="username" className="sr-only">
                Username
              </label>
              <input
                id="username"
                name="username"
                type="text"
                autoComplete="username"
                required
                className="appearance-none rounded-t-md relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-300 placeholder-gray-500 dark:placeholder-gray-500 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="password" className="sr-only">
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="new-password"
                required
                className="appearance-none rounded-b-md relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-300 placeholder-gray-500 dark:placeholder-gray-500 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          </div>

          <div>
            <button
              type="submit"
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Create account
            </button>
          </div>
        </form>
        <Footer />
      </div>
    </div>
  );
}
