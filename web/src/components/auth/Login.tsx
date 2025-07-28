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
      <div className="h-screen flex items-center justify-center bg-white dark:bg-gray-900 overflow-hidden">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-blue-500" />
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-white dark:bg-gray-900 pattern overflow-hidden m-0 p-0">
      <Card className="w-full max-w-md mx-8 bg-white/95 dark:bg-gray-850/95 border-gray-200 dark:border-gray-800 shadow-xl">
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
          {oidcEnabled ? (
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
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="bg-red-100 dark:bg-red-900/20 border border-red-400 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-md text-sm">
                  {error}
                </div>
              )}

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
          <Footer />
        </CardContent>
      </Card>
    </div>
  );
}
