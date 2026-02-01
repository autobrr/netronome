/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
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

export default function Login() {
  const { t } = useTranslation();
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
        
        // Map error messages to translations
        if (errorMessage === "Invalid credentials") {
          setError(t('auth.invalidCredentials'));
        } else if (errorMessage === "Invalid request data") {
          setError(t('auth.invalidRequest'));
        } else if (errorMessage === "Failed to get user" || errorMessage === "Failed to generate session token") {
          setError(t('auth.authFailed'));
        } else {
          setError(t('errors.generic'));
        }
      } else {
        setError(t('errors.generic'));
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
            {t('app.subtitle')}
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
              <span className="flex items-center" aria-label={t('auth.loginWithOIDC')}>
                {t('auth.loginWithOIDC')}
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
                    {t('auth.username')}
                  </Label>
                  <Input
                    id="username"
                    name="username"
                    type="text"
                    autoComplete="username"
                    required
                    placeholder={t('auth.username')}
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
                    {t('auth.password')}
                  </Label>
                  <Input
                    id="password"
                    name="password"
                    type="password"
                    autoComplete="current-password"
                    required
                    placeholder={t('auth.password')}
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
                {t('auth.login')}
              </Button>
            </form>
          )}
          <Footer />
        </CardContent>
      </Card>
    </div>
  );
}
