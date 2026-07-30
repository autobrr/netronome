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
import { cn } from "@/lib/utils";

const icons: Record<CryptoAddress["symbol"], React.ReactNode> = {
  BTC: <FaBitcoin className="h-4 w-4 text-orange-500" />,
  ETH: <FaEthereum className="h-4 w-4 text-indigo-400" />,
  LTC: <SiLitecoin className="h-4 w-4 text-gray-400" />,
  XMR: <SiMonero className="h-4 w-4 text-orange-400" />,
};

// Middle truncation, not CSS truncate: the tail is what people check when
// verifying a payment address. 27 chars still fits the narrow donate dialog
// on a 375px phone; anything longer would end-ellipsize there and hide the
// tail. ponytail: fixed cut, measure the actual row width if it ever matters.
function truncateAddress(address: string): string {
  if (address.length <= 28) return address;
  return `${address.slice(0, 14)}...${address.slice(-10)}`;
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

export function CryptoAddressRow({
  crypto,
  density = "compact",
}: {
  crypto: CryptoAddress;
  density?: "compact" | "comfortable";
}) {
  const [copied, setCopied] = useState(false);
  const isComfortable = density === "comfortable";

  const handleCopy = useCallback(async () => {
    const ok = await copyToClipboard(crypto.address);
    if (ok) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [crypto.address]);

  return (
    <div
      className={cn(
        "flex items-center gap-2 text-sm",
        isComfortable && "min-h-11"
      )}
    >
      {icons[crypto.symbol]}
      <span className="font-medium text-gray-600 dark:text-gray-400 w-8 flex-shrink-0">
        {crypto.symbol}
      </span>
      <code className="min-w-0 flex-1 truncate text-xs text-gray-500 dark:text-gray-500 font-mono">
        {truncateAddress(crypto.address)}
      </code>
      <Button
        variant="ghost"
        size="icon"
        className={cn(
          "flex-shrink-0",
          isComfortable ? "h-11 w-11" : "h-7 w-7"
        )}
        onClick={handleCopy}
        aria-label={`${copied ? "Copied" : "Copy"} ${crypto.symbol} address for ${crypto.owner}`}
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
