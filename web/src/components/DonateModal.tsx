/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { ExternalLink } from "lucide-react";
import { FaGithub, FaPatreon } from "react-icons/fa";
import { SiBuymeacoffee, SiKofi } from "react-icons/si";
import { HeartIcon } from "@heroicons/react/24/solid";
import s0upAvatar from "@/assets/sponsors/s0up4200.png";
import zze0sAvatar from "@/assets/sponsors/zze0s.png";
import { useLicense } from "@/hooks/useLicense";
import { CryptoAddressRow } from "@/components/common/CryptoAddressRow";
import { SOUP_CRYPTO, ZZE0S_CRYPTO, type CryptoAddress } from "@/constants/crypto";
import { PremiumLicenseModal } from "@/components/PremiumLicenseModal";

// Polar SVG component
const PolarIcon: React.FC<{ className?: string }> = ({ className = "w-full h-full" }) => (
  <svg viewBox="-0.5 -0.5 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M7.5 14.337187499999999C3.7239375000000003 14.337187499999999 0.6628125 11.276062499999998 0.6628125 7.5 0.6628125 3.7239375000000003 3.7239375000000003 0.6628125 7.5 0.6628125c3.7760624999999997 0 6.837187500000001 3.061125 6.837187500000001 6.837187500000001 0 3.7760624999999997 -3.061125 6.837187500000001 -6.837187500000001 6.837187500000001Z" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
    <path d="M7.5 14.337187499999999c-1.5104375 0 -2.7348749999999997 -3.061125 -2.7348749999999997 -6.837187500000001C4.765125 3.7239375000000003 5.9895625 0.6628125 7.5 0.6628125c1.5103749999999998 0 2.7348749999999997 3.061125 2.7348749999999997 6.837187500000001 0 3.7760624999999997 -1.2245 6.837187500000001 -2.7348749999999997 6.837187500000001Z" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
    <path d="M5.4488125 13.653500000000001c-2.051125 -0.6837500000000001 -2.7348749999999997 -3.6845624999999997 -2.7348749999999997 -5.811625 0 -2.1270625 1.025625 -4.7860625 3.418625 -6.495375" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
    <path d="M9.551187500000001 1.3464999999999998c2.051125 0.6837500000000001 2.7348749999999997 3.6846250000000005 2.7348749999999997 5.811625 0 2.1270625 -1.025625 4.7860625 -3.418625 6.495375" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
  </svg>
);

interface PlatformLink {
  name: string;
  url: string;
  icon: React.ReactNode;
}

interface Maintainer {
  name: string;
  role: string;
  avatar: string;
  platforms: PlatformLink[];
  crypto: CryptoAddress[];
}

// One project-level ask, picked by license state. Never both: the premium
// checkout and the sponsorship checkout are easy to confuse side by side.
const sponsorCard = {
  name: "Polar",
  url: "https://buy.polar.sh/polar_cl_wWoEUigSOTJIoTrKaGIj3NU6oOCc4xJsKnsDN3NaATF",
  description: "Sponsor the Netronome project",
};

// No url: this one opens PremiumLicenseModal rather than jumping straight to
// checkout, because crypto is a real payment route here and the checkout page
// cannot explain it.
const premiumCard = {
  name: "Premium Themes",
  description: "Pay by card or crypto, and unlock all premium themes",
};

const maintainers: Maintainer[] = [
  {
    name: "s0up",
    role: "Maintainer",
    avatar: s0upAvatar,
    platforms: [
      { name: "GitHub Sponsors", url: "https://github.com/sponsors/s0up4200/", icon: <FaGithub className="h-4 w-4" /> },
      { name: "Patreon", url: "https://www.patreon.com/c/s0up4200", icon: <FaPatreon className="h-4 w-4" /> },
      { name: "Buy Me a Coffee", url: "https://buymeacoffee.com/s0up4200", icon: <SiBuymeacoffee className="h-4 w-4" /> },
      { name: "Ko-fi", url: "https://ko-fi.com/s0up4200", icon: <SiKofi className="h-4 w-4" /> },
    ],
    crypto: SOUP_CRYPTO,
  },
  {
    name: "zze0s",
    role: "autobrr maintainer",
    avatar: zze0sAvatar,
    platforms: [
      { name: "GitHub Sponsors", url: "https://github.com/sponsors/zze0s", icon: <FaGithub className="h-4 w-4" /> },
      { name: "Buy Me a Coffee", url: "https://buymeacoffee.com/ze0s", icon: <SiBuymeacoffee className="h-4 w-4" /> },
      { name: "Ko-fi", url: "https://ko-fi.com/theze0s", icon: <SiKofi className="h-4 w-4" /> },
    ],
    crypto: ZZE0S_CRYPTO,
  },
];

function PlatformLinkItem({ link }: { link: PlatformLink }) {
  return (
    <a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center gap-2 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-700 transition-colors"
    >
      {link.icon}
      <span className="truncate">{link.name}</span>
      <ExternalLink className="h-3 w-3 ml-auto flex-shrink-0 text-gray-400" />
    </a>
  );
}

function MaintainerSection({ maintainer }: { maintainer: Maintainer }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2.5">
        <img
          src={maintainer.avatar}
          alt={maintainer.name}
          className="h-8 w-8 rounded-full border border-gray-200 dark:border-gray-700"
        />
        <div className="min-w-0">
          <p className="font-medium text-gray-900 dark:text-white leading-tight">
            {maintainer.name}
          </p>
          <p className="text-[11px] text-gray-500 dark:text-gray-400 leading-tight">
            {maintainer.role}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        {maintainer.platforms.map((link) => (
          <PlatformLinkItem key={link.name} link={link} />
        ))}
      </div>

      {maintainer.crypto.length > 0 && (
        <div className="space-y-1.5">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
            Crypto
          </span>
          <div className="space-y-1">
            {maintainer.crypto.map((c) => (
              <CryptoAddressRow key={c.symbol} crypto={c} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface DonateModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const cardClass =
  "group flex items-center gap-3 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50 p-4 hover:bg-gray-100/70 dark:hover:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-700 transition-colors";

export function DonateModal({ isOpen, onClose }: DonateModalProps) {
  const { data: license } = useLicense();
  const project = license?.hasPremiumAccess ? sponsorCard : premiumCard;
  const [showPremium, setShowPremium] = useState(false);

  // Hand off rather than stack: two dialogs deep is worse than one that replaces
  // the other. This component stays mounted (App.tsx), so the state survives.
  const openPremium = () => {
    onClose();
    setShowPremium(true);
  };

  const card = (
    <>
      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 shadow-sm">
        <PolarIcon className="h-5 w-5" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-gray-900 dark:text-white group-hover:text-blue-500 dark:group-hover:text-blue-400 transition-colors">
            {project.name}
          </span>
          <Badge variant="default" className="text-[10px] px-2 py-0">
            Recommended
          </Badge>
        </div>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          {project.description}
        </p>
      </div>
      <ExternalLink className="h-4 w-4 text-gray-400 group-hover:text-blue-500 dark:group-hover:text-blue-400 transition-colors flex-shrink-0" />
    </>
  );

  return (
    <>
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800">
        <DialogHeader>
          <DialogTitle>Support Netronome</DialogTitle>
          <DialogDescription>
            Your sponsorship supports features, infrastructure, and community.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[60vh] overflow-y-auto space-y-5 pr-1">
          {/* Project-level: premium purchase, or plain sponsorship once licensed.
              The premium one opens the purchase dialog instead of the checkout,
              because crypto is a real route and checkout cannot explain it. */}
          {project === premiumCard ? (
            <button type="button" onClick={openPremium} className={`${cardClass} w-full text-left`}>
              {card}
            </button>
          ) : (
            <a
              href={sponsorCard.url}
              target="_blank"
              rel="noopener noreferrer"
              className={cardClass}
            >
              {card}
            </a>
          )}

          {/* Divider */}
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-gray-200 dark:border-gray-800" />
            </div>
            <div className="relative flex justify-center text-xs">
              <span className="bg-white dark:bg-gray-900 px-3 text-gray-400 dark:text-gray-500 font-medium">
                Maintainers
              </span>
            </div>
          </div>

          {/* Maintainer sections */}
          {maintainers.map((m) => (
            <MaintainerSection key={m.name} maintainer={m} />
          ))}
        </div>

        {/* Outside the scroll area so it is always visible */}
        <p className="text-sm text-gray-500 dark:text-gray-400 text-center flex items-center justify-center gap-1">
          Thank you for your support <HeartIcon className="h-4 w-4 text-red-500" />
        </p>
      </DialogContent>
    </Dialog>

    <PremiumLicenseModal
      isOpen={showPremium}
      onClose={() => setShowPremium(false)}
    />
    </>
  );
}
