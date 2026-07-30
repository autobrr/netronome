/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useCallback } from "react";
import { Button } from "@/components/ui/Button";
import { Check, Copy } from "lucide-react";
import { FaBitcoin, FaEthereum } from "react-icons/fa";
import { SiLitecoin, SiMonero } from "react-icons/si";
import type { CryptoAddress } from "@/constants/crypto";

const icons: Record<CryptoAddress["symbol"], React.ReactNode> = {
  BTC: <FaBitcoin className="h-4 w-4 text-orange-500" />,
  ETH: <FaEthereum className="h-4 w-4 text-indigo-400" />,
  LTC: <SiLitecoin className="h-4 w-4 text-gray-400" />,
  XMR: <SiMonero className="h-4 w-4 text-orange-400" />,
};

function truncateAddress(address: string): string {
  if (address.length <= 16) return address;
  return `${address.slice(0, 8)}...${address.slice(-6)}`;
}

// execCommand fallback: clipboard.writeText needs a secure context, and plenty of
// self-hosted installs are plain http on a LAN address.
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

/**
 * showOwner is opt-in rather than driven off crypto.owner: the donate dialog
 * already groups rows under a maintainer's name, so labelling them there would
 * just repeat the heading. Only the purchase flow mixes owners in one list.
 */
export function CryptoAddressRow({
  crypto,
  showOwner = false,
}: {
  crypto: CryptoAddress;
  showOwner?: boolean;
}) {
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
      {icons[crypto.symbol]}
      <span className="font-medium text-gray-600 dark:text-gray-400 w-8 flex-shrink-0">
        {crypto.symbol}
      </span>
      <code className="flex-1 truncate text-xs text-gray-500 dark:text-gray-500 font-mono">
        {truncateAddress(crypto.address)}
      </code>
      {showOwner && (
        <span className="flex-shrink-0 text-[11px] text-gray-400 dark:text-gray-500">
          {crypto.owner}
        </span>
      )}
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 flex-shrink-0"
        onClick={handleCopy}
        // Two owners can share a coin, so the owner is part of the label -
        // otherwise a screen reader hears "Copy BTC address" twice in a row.
        aria-label={`${copied ? "Copied" : "Copy"} ${crypto.symbol} address${
          showOwner ? ` for ${crypto.owner}` : ""
        }`}
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
