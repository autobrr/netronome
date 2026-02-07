/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/Button";
import { Check, Copy, ExternalLink } from "lucide-react";
import { FaGithub, FaBitcoin, FaEthereum, FaPatreon } from "react-icons/fa";
import { SiBuymeacoffee, SiKofi, SiLitecoin, SiMonero } from "react-icons/si";
import { HeartIcon } from "@heroicons/react/24/solid";
import s0upAvatar from "@/assets/sponsors/s0up4200.png";
import zze0sAvatar from "@/assets/sponsors/zze0s.png";

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

interface CryptoAddress {
  name: string;
  symbol: string;
  address: string;
  icon: React.ReactNode;
}

interface Maintainer {
  name: string;
  avatar: string;
  platforms: PlatformLink[];
  crypto: CryptoAddress[];
}

const projectSponsor: PlatformLink = {
  name: "Polar",
  url: "https://buy.polar.sh/polar_cl_wWoEUigSOTJIoTrKaGIj3NU6oOCc4xJsKnsDN3NaATF",
  icon: <PolarIcon className="h-5 w-5" />,
};

const maintainers: Maintainer[] = [
  {
    name: "s0up",
    avatar: s0upAvatar,
    platforms: [
      { name: "GitHub Sponsors", url: "https://github.com/sponsors/s0up4200/", icon: <FaGithub className="h-4 w-4" /> },
      { name: "Patreon", url: "https://www.patreon.com/c/s0up4200", icon: <FaPatreon className="h-4 w-4" /> },
      { name: "Buy Me a Coffee", url: "https://buymeacoffee.com/s0up4200", icon: <SiBuymeacoffee className="h-4 w-4" /> },
      { name: "Ko-fi", url: "https://ko-fi.com/s0up4200", icon: <SiKofi className="h-4 w-4" /> },
    ],
    crypto: [
      { name: "Bitcoin", symbol: "BTC", address: "bc1qfe093kmhvsa436v4ksz0udfcggg3vtnm2tjgem", icon: <FaBitcoin className="h-4 w-4 text-orange-500" /> },
      { name: "Ethereum", symbol: "ETH", address: "0xD8f517c395a68FEa8d19832398d4dA7b45cbc38F", icon: <FaEthereum className="h-4 w-4 text-indigo-400" /> },
      { name: "Litecoin", symbol: "LTC", address: "ltc1q86nx64mu2j22psj378amm58ghvy4c9dw80z88h", icon: <SiLitecoin className="h-4 w-4 text-gray-400" /> },
      { name: "Monero", symbol: "XMR", address: "8AMPTPgjmLG9armLBvRA8NMZqPWuNT4US3kQoZrxDDVSU21kpYpFr1UCWmmtcBKGsvDCFA3KTphGXExWb3aHEu67JkcjAvC", icon: <SiMonero className="h-4 w-4 text-orange-400" /> },
    ],
  },
  {
    name: "zze0s",
    avatar: zze0sAvatar,
    platforms: [
      { name: "GitHub Sponsors", url: "https://github.com/sponsors/zze0s", icon: <FaGithub className="h-4 w-4" /> },
      { name: "Buy Me a Coffee", url: "https://buymeacoffee.com/ze0s", icon: <SiBuymeacoffee className="h-4 w-4" /> },
    ],
    crypto: [
      { name: "Bitcoin", symbol: "BTC", address: "bc1q2nvdd83hrzelqn4vyjm8tvjwmsuuxsdlg4ws7x", icon: <FaBitcoin className="h-4 w-4 text-orange-500" /> },
      { name: "Ethereum", symbol: "ETH", address: "0xBF7d749574aabF17fC35b27232892d3F0ff4D423", icon: <FaEthereum className="h-4 w-4 text-indigo-400" /> },
      { name: "Litecoin", symbol: "LTC", address: "ltc1qza9ffjr5y43uk8nj9ndjx9hkj0ph3rhur6wudn", icon: <SiLitecoin className="h-4 w-4 text-gray-400" /> },
      { name: "Monero", symbol: "XMR", address: "44AvbWXzFN3bnv2oj92AmEaR26PQf5Ys4W155zw3frvEJf2s4g325bk4tRBgH7umSVMhk88vkU3gw9cDvuCSHgpRPsuWVJp", icon: <SiMonero className="h-4 w-4 text-orange-400" /> },
    ],
  },
];

function truncateAddress(address: string): string {
  if (address.length <= 16) return address;
  return `${address.slice(0, 8)}...${address.slice(-6)}`;
}

function copyToClipboard(text: string): Promise<boolean> {
  return navigator.clipboard.writeText(text).then(
    () => true,
    () => {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.opacity = "0";
      document.body.appendChild(textArea);
      textArea.select();
      try {
        const ok = document.execCommand("copy");
        document.body.removeChild(textArea);
        return ok;
      } catch {
        document.body.removeChild(textArea);
        return false;
      }
    }
  );
}

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

function CryptoAddressRow({ crypto }: { crypto: CryptoAddress }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    const ok = await copyToClipboard(crypto.address);
    if (ok) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [crypto.address]);

  return (
    <div className="flex items-center gap-2 text-sm">
      {crypto.icon}
      <span className="font-medium text-gray-600 dark:text-gray-400 w-8 flex-shrink-0">
        {crypto.symbol}
      </span>
      <code className="flex-1 truncate text-xs text-gray-500 dark:text-gray-500 font-mono">
        {truncateAddress(crypto.address)}
      </code>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 flex-shrink-0"
        onClick={handleCopy}
        aria-label={copied ? `${crypto.symbol} address copied` : `Copy ${crypto.symbol} address`}
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-emerald-500" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
      </Button>
    </div>
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
        <span className="font-medium text-gray-900 dark:text-white">
          {maintainer.name}
        </span>
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

export function DonateModal({ isOpen, onClose }: DonateModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800">
        <DialogHeader>
          <DialogTitle>Support Netronome</DialogTitle>
          <DialogDescription>
            Your sponsorship supports features, infrastructure, and community.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[60vh] overflow-y-auto space-y-5 pr-1">
          {/* Project-level: Polar */}
          <a
            href={projectSponsor.url}
            target="_blank"
            rel="noopener noreferrer"
            className="group flex items-center gap-3 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50 p-4 hover:bg-gray-100/70 dark:hover:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-700 transition-colors"
          >
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 shadow-sm">
              {projectSponsor.icon}
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium text-gray-900 dark:text-white group-hover:text-blue-500 dark:group-hover:text-blue-400 transition-colors">
                  {projectSponsor.name}
                </span>
                <Badge variant="default" className="text-[10px] px-2 py-0">
                  Recommended
                </Badge>
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Sponsor the Netronome project
              </p>
            </div>
            <ExternalLink className="h-4 w-4 text-gray-400 group-hover:text-blue-500 dark:group-hover:text-blue-400 transition-colors flex-shrink-0" />
          </a>

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

          {/* Footer */}
          <p className="text-sm text-gray-500 dark:text-gray-400 text-center flex items-center justify-center gap-1 pt-1">
            Thank you for your support <HeartIcon className="h-4 w-4 text-red-500" />
          </p>
        </div>
      </DialogContent>
    </Dialog>
  );
}
