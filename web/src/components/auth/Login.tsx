/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { useAuth } from "@/context/auth";
import { router } from "@/routes";
import logo from "@/assets/logo_small.png";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faOpenid } from "@fortawesome/free-brands-svg-icons";
import { Footer } from "@/components/Footer";
import { getApiUrl } from "@/utils/baseUrl";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/Button";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

// Error message mapping for cleaner code
const ERROR_MESSAGES: Record<string, string> = {
  "Invalid credentials": "Incorrect username or password",
  "Invalid request data": "Please check your input and try again",
  "Failed to get user": "Unable to verify user credentials",
  "Failed to generate session token": "Authentication failed, please try again",
};

const OIDC_ERROR_MESSAGES: Record<string, string> = {
  oidc_unavailable: "OpenID Connect sign-in is currently unavailable. Use local credentials if you have them.",
};

function getInitialLoginError(): string {
  const errorCode = new URLSearchParams(window.location.search).get("error");
  if (!errorCode) {
    return "";
  }

  return OIDC_ERROR_MESSAGES[errorCode] || "Unable to sign in with OpenID Connect.";
}

export default function Login() {
  const { login, checkRegistrationStatus } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(getInitialLoginError);
  const [isLoading, setIsLoading] = useState(true);
  const [authOptions, setAuthOptions] = useState({
    hasUsers: true,
    oidcConfigured: false,
    oidcReady: false,
  });

  useEffect(() => {
    const checkStatus = async () => {
      try {
        const status = await checkRegistrationStatus();
        setAuthOptions({
          hasUsers: status.hasUsers,
          oidcConfigured: status.oidcConfigured,
          oidcReady: status.oidcReady,
        });

        if (!status.hasUsers && !status.oidcConfigured) {
          await router.navigate({ to: "/register" });
        }
      } catch (err) {
        console.error("Failed to check registration status:", err);
      } finally {
        setIsLoading(false);
      }
    };

    checkStatus();
  }, [checkRegistrationStatus]);

  const handleOIDCLogin = () => {
    window.location.href = getApiUrl("/auth/oidc/login");
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      await login(username, password);
    } catch (err) {
      if (err instanceof Error) {
        const errorMessage = err.message;

        if (errorMessage.includes("User not found")) {
          console.log("No users found during login, redirecting...");
          router.navigate({ to: "/register" });
          return;
        }

        setError(ERROR_MESSAGES[errorMessage] || "An error occurred while signing in");
      } else {
        setError("Unable to sign in at this time");
      }
    }
  };

  if (isLoading) {
    return (
      <div className="h-screen flex items-center justify-center bg-white dark:bg-gray-900 overflow-hidden px-4 sm:px-6">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-white dark:bg-gray-900 pattern overflow-hidden m-0 px-4 sm:px-6">
      <Card className="w-full max-w-md bg-white/95 dark:bg-gray-850/95 border-gray-200 dark:border-gray-800 shadow-xl">
        <CardHeader className="text-center pb-2">
          <img
            src={logo}
            alt="Netronome Logo"
            className="h-16 w-16 mx-auto mb-2 select-none pointer-events-none"
          />
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white pointer-events-none select-none">
            Netronome
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 pointer-events-none select-none">
            network performance testing
          </p>
        </CardHeader>

        <CardContent className="pt-6">
          <div className="space-y-4">
            {error && (
              <div className="bg-red-100 dark:bg-red-900/20 border border-red-400 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-md text-sm">
                {error}
              </div>
            )}

            {authOptions.oidcReady && (
              <Button
                onClick={handleOIDCLogin}
                variant="outline"
                className="w-full border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400"
                size="lg"
              >
                <span className="flex items-center" aria-label="Sign in with OpenID">
                  Sign in with
                  <FontAwesomeIcon icon={faOpenid} className="text-lg ml-2" aria-hidden="true" />
                </span>
              </Button>
            )}

            {authOptions.oidcReady && authOptions.hasUsers && (
              <div className="relative">
                <div className="absolute inset-0 flex items-center" aria-hidden="true">
                  <div className="w-full border-t border-gray-200 dark:border-gray-700" />
                </div>
                <div className="relative flex justify-center text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  <span className="bg-white dark:bg-gray-850 px-3">or</span>
                </div>
              </div>
            )}

            {authOptions.hasUsers && (
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="username" className="sr-only">
                      Username
                    </Label>
                    <Input
                      id="username"
                      name="username"
                      type="text"
                      autoComplete="username"
                      required
                      placeholder="Username"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      className={cn(
                        "dark:bg-gray-800 dark:border-gray-700",
                        error && "border-red-500 dark:border-red-500"
                      )}
                    />
                  </div>
                  <div>
                    <Label htmlFor="password" className="sr-only">
                      Password
                    </Label>
                    <Input
                      id="password"
                      name="password"
                      type="password"
                      autoComplete="current-password"
                      required
                      placeholder="Password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className={cn(
                        "dark:bg-gray-800 dark:border-gray-700",
                        error && "border-red-500 dark:border-red-500"
                      )}
                    />
                  </div>
                </div>

                <Button type="submit" className="w-full" size="lg">
                  Sign in
                </Button>
              </form>
            )}

            {!authOptions.hasUsers && authOptions.oidcConfigured && !authOptions.oidcReady && (
              <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200">
                OpenID Connect is configured but the provider is unavailable right now.
              </div>
            )}
          </div>
          <Footer />
        </CardContent>
      </Card>
    </div>
  );
}
