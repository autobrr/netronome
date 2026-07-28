/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  KeyIcon,
  CheckBadgeIcon,
  ExclamationTriangleIcon,
} from "@heroicons/react/24/outline";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";
import {
  useLicense,
  useActivateLicense,
  useDeactivateLicense,
} from "@/hooks/useLicense";
import { showToast } from "@/components/common/Toast";
import { PremiumLicenseModal } from "@/components/PremiumLicenseModal";
import { POLAR_PORTAL_URL } from "@/constants/premium";
import { formatDate as formatDateWithSettings } from "@/utils/timeSettings";

const formatDate = (value: string): string =>
  formatDateWithSettings(value, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

/** Link to the Polar portal, where a customer releases their own activations. */
const PortalLink: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <a
    href={POLAR_PORTAL_URL}
    target="_blank"
    rel="noopener noreferrer"
    className="font-medium text-blue-600 dark:text-blue-400 hover:underline"
  >
    {children}
  </a>
);

export const LicenseSettings: React.FC = () => {
  const { data, isLoading, isError, refetch } = useLicense();
  const activate = useActivateLicense();
  const deactivate = useDeactivateLicense();

  const [licenseKey, setLicenseKey] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [showPurchase, setShowPurchase] = useState(false);

  const license = data?.license ?? null;
  const activationError =
    activate.error instanceof Error ? activate.error.message : null;

  // A license row with entitlement switched off: Polar rejected it on the last
  // check, or nothing has reached Polar for longer than the offline grace
  // period. Either way the themes are already gone, so say why.
  const isRejected = license !== null && !data?.hasPremiumAccess;

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
              {isRejected ? (
                <ExclamationTriangleIcon className="w-5 h-5 text-amber-600 dark:text-amber-400" />
              ) : (
                <CheckBadgeIcon className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              )}
              {isRejected ? "Premium Access Inactive" : "Premium Access"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {isRejected && (
              <div className="rounded-md border border-amber-300 dark:border-amber-500/40 bg-amber-50 dark:bg-amber-500/10 p-3 text-sm text-amber-900 dark:text-amber-200">
                <p>
                  Premium themes are disabled because this license did not pass
                  its last check with Polar.
                </p>
                <p className="mt-2">
                  If you released this machine&apos;s activation in the{" "}
                  <PortalLink>Polar customer portal</PortalLink>, or moved this
                  install to a new machine, deactivate here and activate the key
                  again. A temporary outage clears on its own at the next check.
                </p>
              </div>
            )}

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
              Activations from other machines are managed in the{" "}
              <PortalLink>Polar customer portal</PortalLink>.
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
                  placeholder="NETRONOME-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
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
                {/* Activations are seats and a reinstall onto a fresh config
                    directory burns one, so this is a dead end without a way
                    to release the old machine.

                    ponytail: matched against the server's copy rather than an
                    error code. Reword activationErrorResponse and the hint
                    silently stops showing - swap to a code on the response if
                    a second hint ever needs one. */}
                {activationError?.includes("activation limit") && (
                  <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                    Release an activation you no longer use in the{" "}
                    <PortalLink>Polar customer portal</PortalLink>, then try
                    again.
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
                {/* Crypto pays for a license too, and that needs explaining -
                    so the buy link is a dialog, not a jump to checkout. */}
                <button
                  type="button"
                  onClick={() => setShowPurchase(true)}
                  className="text-sm font-medium text-blue-600 dark:text-blue-400 hover:underline"
                >
                  Don&apos;t have a license? Get one &rarr;
                </button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <PremiumLicenseModal
        isOpen={showPurchase}
        onClose={() => setShowPurchase(false)}
      />
    </>
  );
};
