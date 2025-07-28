/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { motion } from "motion/react";
import { useAuth } from "@/context/auth";
import { checkRegistrationStatus } from "@/api/auth";
import { router } from "@/routes";
import logo from "@/assets/logo_small.png";
import { Footer } from "@/components/Footer";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loader2 } from "lucide-react";

export default function Register() {
  const [formData, setFormData] = useState({ username: "", password: "" });
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { register } = useAuth();

  useEffect(() => {
    checkRegistrationStatus()
      .then((allowed) => !allowed && router.navigate({ to: "/login" }))
      .catch(() => router.navigate({ to: "/login" }))
      .finally(() => setIsLoading(false));
  }, []);

  const handleInputChange =
    (field: "username" | "password") =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFormData((prev) => ({ ...prev, [field]: e.target.value }));
    };

  const parseErrorMessage = (err: unknown): string => {
    if (!(err instanceof Error)) return "Registration failed";

    const message = err.message;
    return message.includes("Error #01:")
      ? message.split("Error #01:")[1].trim()
      : message;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (formData.password.length < 4) {
      setError("Password must be at least 4 characters");
      return;
    }

    setIsSubmitting(true);

    try {
      await register(formData.username, formData.password);
      router.navigate({ to: "/login" });
    } catch (err) {
      setError(parseErrorMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-white dark:bg-gray-900">
        <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-white dark:bg-gray-900 pattern">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
      >
        <Card className="w-full max-w-md mx-8 bg-white/95 dark:bg-gray-850/95 border-gray-200 dark:border-gray-800 shadow-xl">
          <CardHeader className="text-center">
            <img
              src={logo}
              alt="Netronome Logo"
              className="h-16 w-16 mx-auto mb-2 select-none pointer-events-none"
            />
            <CardTitle className="text-3xl font-bold text-gray-900 dark:text-white pointer-events-none select-none">
              Netronome
            </CardTitle>
            <CardDescription className="text-gray-600 dark:text-gray-400">Create your account</CardDescription>
          </CardHeader>

          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              {error && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ duration: 0.2 }}
                >
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                </motion.div>
              )}

              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="username" className="text-gray-700 dark:text-gray-300">Username</Label>
                  <Input
                    id="username"
                    type="text"
                    placeholder="Enter your username"
                    value={formData.username}
                    onChange={handleInputChange("username")}
                    autoComplete="username"
                    required
                    disabled={isSubmitting}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="password" className="text-gray-700 dark:text-gray-300">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Enter your password"
                    value={formData.password}
                    onChange={handleInputChange("password")}
                    autoComplete="new-password"
                    required
                    disabled={isSubmitting}
                  />
                </div>
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={isSubmitting}
                isLoading={isSubmitting}
              >
                Create account
              </Button>
            </form>

            <Footer />
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}
