/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { KeyIcon, CheckBadgeIcon } from "@heroicons/react/24/outline";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";
import {
  useLicense,
  useActivateLicense,
  useDeactivateLicense,
} from "@/hooks/useLicense";
import { showToast } from "@/components/common/Toast";
import { POLAR_CHECKOUT_URL } from "@/constants/premium";
import { formatDate as formatDateWithSettings } from "@/utils/timeSettings";

const formatDate = (value: string): string =>
  formatDateWithSettings(value, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

export const LicenseSettings: React.FC = () => {
  const { data, isLoading, isError, refetch } = useLicense();
  const activate = useActivateLicense();
  const deactivate = useDeactivateLicense();

  const [licenseKey, setLicenseKey] = useState("");
  const [confirming, setConfirming] = useState(false);

  const license = data?.license ?? null;
  const activationError =
    activate.error instanceof Error ? activate.error.message : null;

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const key = licenseKey.trim();
    if (!key) return;

    // Failures surface through the inline activationError under the input.
    activate.mutate(key, {
      onSuccess: () => {
        setLicenseKey("");
        showToast("License activated", "success", {
          description: "Premium themes are now unlocked",
        });
      },
    });
  };

  const confirmDeactivate = () => {
    deactivate.mutate(undefined, {
      onSuccess: () => {
        setConfirming(false);
        showToast("License deactivated", "success", {
          description: "Premium themes are no longer available",
        });
      },
      onError: (err: unknown) => {
        showToast("Failed to deactivate license", "error", {
          description: err instanceof Error ? err.message : undefined,
        });
      },
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[120px]">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full mx-auto mb-4 animate-spin" />
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Loading license...
          </p>
        </div>
      </div>
    );
  }

  // A failed fetch must not render the activate/purchase card: on a licensed
  // instance that would look like the license vanished and hide Deactivate.
  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            License
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-red-600 dark:text-red-400">
            Could not load the license status. Your license is unaffected.
          </p>
          <Button variant="secondary" onClick={() => refetch()}>
            Retry
          </Button>
        </CardContent>
      </Card>
    );
  }

  // Rendered as a card inside ThemeSettings; the page header lives there.
  return (
    <>
      {license ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckBadgeIcon className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              Premium Access
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
              <div>
                <dt className="text-gray-600 dark:text-gray-400">License key</dt>
                <dd className="mt-1 font-mono text-gray-900 dark:text-white break-all">
                  {license.licenseKey}
                </dd>
              </div>
              <div>
                <dt className="text-gray-600 dark:text-gray-400">Status</dt>
                <dd className="mt-1 capitalize text-gray-900 dark:text-white">
                  {license.status}
                </dd>
              </div>
              <div>
                <dt className="text-gray-600 dark:text-gray-400">Activated</dt>
                <dd className="mt-1 text-gray-900 dark:text-white">
                  {formatDate(license.activatedAt)}
                </dd>
              </div>
              <div>
                <dt className="text-gray-600 dark:text-gray-400">Expires</dt>
                <dd className="mt-1 text-gray-900 dark:text-white">
                  {license.expiresAt ? formatDate(license.expiresAt) : "Perpetual"}
                </dd>
              </div>
            </dl>

            <p className="text-sm text-gray-600 dark:text-gray-400">
              Deactivating releases this key so it can be used on another instance.
              Premium themes revert to the default until a license is activated again.
            </p>

            {confirming ? (
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  Deactivate this license?
                </span>
                <Button
                  variant="destructive"
                  onClick={confirmDeactivate}
                  isLoading={deactivate.isPending}
                >
                  Confirm
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => setConfirming(false)}
                  disabled={deactivate.isPending}
                >
                  Cancel
                </Button>
              </div>
            ) : (
              <Button variant="secondary" onClick={() => setConfirming(true)}>
                Deactivate
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              Activate License
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <form onSubmit={submit} className="space-y-4">
              <div>
                <label
                  htmlFor="license-key"
                  className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
                >
                  License key
                </label>
                <Input
                  id="license-key"
                  value={licenseKey}
                  onChange={(e) => setLicenseKey(e.target.value)}
                  placeholder="XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
                  autoComplete="off"
                  spellCheck={false}
                  aria-invalid={activationError ? true : undefined}
                  className="w-full sm:w-[420px] font-mono"
                  disabled={activate.isPending}
                />
                {activationError && (
                  <p className="mt-2 text-sm text-red-600 dark:text-red-400">
                    {activationError}
                  </p>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-3">
                <Button
                  type="submit"
                  disabled={!licenseKey.trim()}
                  isLoading={activate.isPending}
                >
                  Activate
                </Button>
                <a
                  href={POLAR_CHECKOUT_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm font-medium text-blue-600 dark:text-blue-400 hover:underline"
                >
                  Don&apos;t have a license? Get one &rarr;
                </a>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
    </>
  );
};
