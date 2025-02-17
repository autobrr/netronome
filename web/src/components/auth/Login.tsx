/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { useAuth } from "@/context/auth";
import { router } from "@/routes";
import logo from "@/assets/logo_small.png";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";
import { Footer } from "@/components/Footer";

export default function Login() {
  const { login, checkRegistrationStatus } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [oidcEnabled, setOidcEnabled] = useState(false);

  useEffect(() => {
    const checkStatus = async () => {
      try {
        const status = await checkRegistrationStatus();
        setOidcEnabled(status.oidcEnabled);

        if (!status.hasUsers && !status.oidcEnabled) {
          await router.navigate({ to: "/register" });
          return;
        }
      } catch (err) {
        console.error("Failed to check registration status:", err);
        if (!oidcEnabled) {
          await router.navigate({ to: "/register" });
        }
      } finally {
        setIsLoading(false);
      }
    };

    checkStatus();
  }, [checkRegistrationStatus, oidcEnabled]);

  const handleOIDCLogin = () => {
    window.location.href = "/api/auth/oidc/login";
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      await login(username, password);
    } catch (err) {
      if (err instanceof Error) {
        const errorMessage = err.message;
        switch (errorMessage) {
          case "Invalid credentials":
            setError("Incorrect username or password");
            break;
          case "Invalid request data":
            setError("Please check your input and try again");
            break;
          case "Failed to get user":
            setError("Unable to verify user credentials");
            break;
          case "Failed to generate session token":
            setError("Authentication failed, please try again");
            break;
          default:
            if (errorMessage.includes("User not found")) {
              console.log("No users found during login, redirecting...");
              router.navigate({ to: "/register" });
              return;
            }
            setError("An error occurred while signing in");
        }
      } else {
        setError("Unable to sign in at this time");
      }
    }
  };

  if (isLoading) {
    return (
      <div className="h-screen flex items-center justify-center bg-gray-900 overflow-hidden">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-gray-900 pattern overflow-hidden m-0 p-0">
      <div className="max-w-md w-full px-8 pt-8 pb-4 bg-gray-850/40 border border-black/40 rounded-lg shadow-lg">
        <div className="flex flex-col items-center">
          <img
            src={logo}
            alt="Netronome Logo"
            className="text-white h-16 w-16 mb-2 select-none pointer-events-none"
          />
          <h2 className="text-3xl font-bold text-white pointer-events-none select-none">
            Netronome
          </h2>
          <p className="text-sm text-gray-500 pointer-events-none select-none mb-0">
            network speed testing
          </p>
        </div>

        {oidcEnabled ? (
          <button
            onClick={() => handleOIDCLogin()}
            className="w-full flex justify-center items-center mt-12 py-2 px-4 border border-gray-750 rounded-md shadow-sm bg-gray-800 hover:bg-gray-825 text-sm font-medium text-white hover:text-blue-450 focus:outline-none focus:ring-1  focus:ring-gray-700"
          >
            <span
              className="group relative inline-block"
              aria-label="Sign in with OpenID"
            >
              Sign in with
              <FontAwesomeIcon
                icon={faOpenid}
                className="text-lg ml-2"
                aria-hidden="true"
              />
            </span>
          </button>
        ) : (
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            {error && (
              <div className="bg-red-500 bg-opacity-10 border border-red-500 text-red-500 px-4 py-3 rounded">
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
                  className="appearance-none rounded-t-md relative block w-full px-3 py-2 border border-gray-700 dark:border-gray-900 bg-gray-700 text-gray-300 placeholder-gray-500 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
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
                  autoComplete="current-password"
                  required
                  className="appearance-none rounded-b-md relative block w-full px-3 py-2 border border-gray-700 dark:border-gray-900 bg-gray-700 text-gray-300 placeholder-gray-500 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
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
                Sign in
              </button>
            </div>
          </form>
        )}
        <Footer />
      </div>
    </div>
  );
}
