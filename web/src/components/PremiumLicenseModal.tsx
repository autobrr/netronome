/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/Button";
import {
  ArrowTopRightOnSquareIcon,
  KeyIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";
import {
  CRYPTO_VERIFIER_URL,
  POLAR_CHECKOUT_URL,
  PREMIUM_MIN_PRICE,
} from "@/constants/premium";
import { CryptoAddressRow } from "@/components/common/CryptoAddressRow";
import {
  VERIFIABLE_CRYPTO,
  type CryptoAddress,
} from "@/constants/crypto";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";

const DISCORD_URL = "https://discord.gg/WehFCZxq5B";
const SUPPORT_EMAIL = "soup@netrono.me";
const PRIMARY_CRYPTO_RECIPIENT = "s0up";

const CRYPTO_RECIPIENTS = Array.from(
  new Set(VERIFIABLE_CRYPTO.map((crypto) => crypto.owner))
).sort((a, b) => {
  if (a === PRIMARY_CRYPTO_RECIPIENT) return -1;
  if (b === PRIMARY_CRYPTO_RECIPIENT) return 1;
  return a.localeCompare(b);
});

const linkClass =
  "text-blue-600 dark:text-blue-400 hover:underline inline-flex items-center gap-1";

const Step: React.FC<{
  number: number;
  title: string;
  children: React.ReactNode;
}> = ({ number, title, children }) => (
  <div className="rounded-lg border border-gray-200 dark:border-gray-800 bg-white/60 dark:bg-gray-900/40 p-4 space-y-3">
    <div className="flex items-center gap-2">
      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-blue-500 dark:bg-blue-600 text-xs font-medium text-on-accent">
        {number}
      </span>
      <p className="text-sm font-semibold text-gray-900 dark:text-white">
        {title}
      </p>
    </div>
    <div className="pl-8 space-y-3">{children}</div>
  </div>
);

function CryptoRows({ addresses }: { addresses: CryptoAddress[] }) {
  return (
    <div className="space-y-0.5">
      {addresses.map((crypto) => (
        <CryptoAddressRow
          key={crypto.address}
          crypto={crypto}
          density="comfortable"
        />
      ))}
    </div>
  );
}

interface PremiumLicenseModalProps {
  isOpen: boolean;
  onClose: () => void;
  /**
   * Only pass this where an activate form is genuinely behind the dialog. Opened
   * from the donate dialog there is no form to return to - closing would look
   * like the button did nothing - so step 3 says where to go instead.
   */
  onActivate?: () => void;
}

/**
 * How to buy the premium theme license, card or crypto.
 *
 * The crypto route is one step shorter than it looks: the verifier hands back a
 * checkout link with the discount code already in the query string, so there is
 * nothing to paste and the total is already $0.
 */
export function PremiumLicenseModal({
  isOpen,
  onClose,
  onActivate,
}: PremiumLicenseModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <SparklesIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            Get a Premium License
          </DialogTitle>
          <DialogDescription>
            Pay what you want (min {PREMIUM_MIN_PRICE}) &bull; Lifetime license
            &bull; All premium themes
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[60vh] overflow-y-auto space-y-3 pr-1">
          <Step number={1} title="Choose payment method">
            <div className="space-y-2">
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                Card or local methods
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                Pay directly in Polar checkout.
              </p>
              <Button variant="outline" size="sm" asChild>
                <a
                  href={POLAR_CHECKOUT_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                  Open Polar checkout
                </a>
              </Button>
            </div>

            <div className="space-y-2">
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                Crypto
              </p>
              <ol className="list-decimal space-y-2 pl-5 text-xs text-gray-600 dark:text-gray-400">
                <li>
                  <p>
                    Send at least {PREMIUM_MIN_PRICE} worth to one of these
                    addresses:
                  </p>
                  {/* Every recipient stays visible; order and the default
                      selection provide the hierarchy without extra promotion. */}
                  {CRYPTO_RECIPIENTS.length > 1 ? (
                    <Tabs
                      defaultValue={PRIMARY_CRYPTO_RECIPIENT}
                      className="mt-2 gap-2"
                    >
                      {/* items-baseline, not items-end: the triggers center
                          their text inside min-h-10 while the label is a bare
                          span, so box-edge alignment leaves the two baselines
                          visibly offset. */}
                      <div className="flex items-baseline gap-4">
                        <span className="text-xs font-medium text-gray-500 dark:text-gray-400">
                          Send to
                        </span>
                        <TabsList
                          aria-label="Choose who receives your crypto payment"
                          className="gap-4 border-b-0 sm:gap-5"
                        >
                          {CRYPTO_RECIPIENTS.map((recipient) => (
                            <TabsTrigger
                              key={recipient}
                              value={recipient}
                              // items-end overrides the base items-center:
                              // min-h-10 is a touch target, and centering in it
                              // strands the text ~10px above the underline.
                              className="min-h-10 items-end px-0 pb-2"
                            >
                              {recipient}
                            </TabsTrigger>
                          ))}
                        </TabsList>
                      </div>

                      {CRYPTO_RECIPIENTS.map((recipient) => (
                        <TabsContent
                          key={recipient}
                          value={recipient}
                          className="mt-0"
                        >
                          <CryptoRows
                            addresses={VERIFIABLE_CRYPTO.filter(
                              (crypto) => crypto.owner === recipient
                            )}
                          />
                        </TabsContent>
                      ))}
                    </Tabs>
                  ) : (
                    <div className="mt-2">
                      <CryptoRows addresses={VERIFIABLE_CRYPTO} />
                    </div>
                  )}
                </li>
                <li>
                  Verify your transaction at{" "}
                  <a
                    href={CRYPTO_VERIFIER_URL}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={linkClass}
                  >
                    crypto.netrono.me
                    <ArrowTopRightOnSquareIcon className="w-3 h-3" />
                  </a>
                  .
                </li>
                <li>
                  Open the checkout link it gives you - the discount is already
                  applied and the total is $0.
                </li>
              </ol>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                XMR is manual: reach out on{" "}
                <a
                  href={DISCORD_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={linkClass}
                >
                  Discord
                </a>{" "}
                or email{" "}
                <a href={`mailto:${SUPPORT_EMAIL}`} className={linkClass}>
                  {SUPPORT_EMAIL}
                </a>
                .
              </p>
            </div>
          </Step>

          <Step number={2} title="Find your license key">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Your key is shown on the success page right after checkout, and
              emailed to you.
            </p>
          </Step>

          <Step number={3} title="Activate your license">
            {onActivate ? (
              <>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Paste the key into the form behind this dialog and hit
                  Activate.
                </p>
                <Button variant="outline" size="sm" onClick={onActivate}>
                  <KeyIcon className="w-4 h-4" />
                  I have my key
                </Button>
              </>
            ) : (
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Open Settings &rarr; Themes &amp; License and paste your key
                there.
              </p>
            )}
          </Step>
        </div>

        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
