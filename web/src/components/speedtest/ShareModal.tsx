/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useCallback, useMemo } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Check, Copy, Share } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { getBaseUrl } from "@/utils/baseUrl";
import { cn } from "@/lib/utils";

// Animation configurations moved outside component for performance
const FADE_IN_ANIMATION = {
  initial: { opacity: 0, y: -10 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -10 },
  transition: { duration: 0.2 },
} as const;

const BUTTON_INTERACTION = {
  whileHover: { scale: 1.02 },
  whileTap: { scale: 0.98 },
  transition: { duration: 0.2 },
} as const;

interface ShareModalProps extends React.HTMLAttributes<HTMLDivElement> {
  isOpen: boolean;
  onClose: () => void;
}

export const ShareModal: React.FC<ShareModalProps> = ({
  isOpen,
  onClose,
  className,
  ...props
}) => {
  const [copySuccess, setCopySuccess] = useState(false);

  const publicUrl = useMemo(() => {
    const baseUrl = getBaseUrl();
    const origin = window.location.origin;
    return `${origin}${baseUrl}/public`;
  }, []);

  const copyToClipboard = useCallback(async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fallback for older browsers
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.opacity = "0";
      document.body.appendChild(textArea);
      textArea.select();

      try {
        const successful = document.execCommand("copy");
        document.body.removeChild(textArea);
        return successful;
      } catch {
        document.body.removeChild(textArea);
        return false;
      }
    }
  }, []);

  const handleCopyUrl = useCallback(async () => {
    const success = await copyToClipboard(publicUrl);

    if (success) {
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    }
  }, [publicUrl, copyToClipboard]);

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent
        className={cn(
          "max-w-md bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800",
          className
        )}
        {...props}
      >
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 border border-blue-500/30">
              <Share className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <DialogTitle>Share Speed Test Results</DialogTitle>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          <Card className="bg-gray-50/95 dark:bg-gray-850/95 border-gray-200 dark:border-gray-800 p-4 shadow-none">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Share some proof that your gigabit connection is more like
              maybe-a-bit.
            </p>
          </Card>

          <div className="space-y-2">
            <Label
              htmlFor="share-url"
              className="text-gray-700 dark:text-gray-300 font-medium"
            >
              Public Dashboard Link
            </Label>
            <div className="flex gap-2">
              <Input
                id="share-url"
                type="text"
                value={publicUrl}
                readOnly
                className="flex-1 bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300"
                aria-label="Public dashboard URL"
              />
              <motion.div {...BUTTON_INTERACTION}>
                <Button
                  onClick={handleCopyUrl}
                  variant="outline"
                  size="icon"
                  className={cn(
                    "transition-colors",
                    copySuccess
                      ? "bg-emerald-500/20 dark:bg-emerald-500/10 hover:bg-emerald-500/30 dark:hover:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border-emerald-500/30 dark:border-emerald-500/30 hover:border-emerald-500/50 dark:hover:border-emerald-500/50"
                      : "bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-100"
                  )}
                  aria-label={
                    copySuccess ? "URL copied" : "Copy URL to clipboard"
                  }
                  aria-pressed={copySuccess}
                >
                  {copySuccess ? (
                    <Check className="h-4 w-4" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </motion.div>
            </div>

            <div className="h-5">
              <AnimatePresence mode="wait">
                {copySuccess && (
                  <motion.div
                    {...FADE_IN_ANIMATION}
                    className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400"
                    role="status"
                    aria-live="polite"
                  >
                    <Check className="h-3 w-3" />
                    <span>URL copied to clipboard!</span>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
