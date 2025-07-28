/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { FaShare, FaCheck, FaCopy } from "react-icons/fa";
import { motion } from "motion/react";
import { getBaseUrl } from "@/utils/baseUrl";

interface ShareModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ShareModal({ isOpen, onClose }: ShareModalProps) {
  const [copySuccess, setCopySuccess] = useState(false);

  const generatePublicUrl = () => {
    const baseUrl = getBaseUrl();
    const origin = window.location.origin;
    return `${origin}${baseUrl}/public`;
  };

  const handleCopyUrl = async () => {
    try {
      const publicUrl = generatePublicUrl();
      await navigator.clipboard.writeText(publicUrl);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    } catch (error) {
      console.error("Failed to copy URL:", error);
      // Fallback for older browsers
      const textArea = document.createElement("textarea");
      textArea.value = generatePublicUrl();
      document.body.appendChild(textArea);
      textArea.select();
      try {
        document.execCommand("copy");
        setCopySuccess(true);
        setTimeout(() => setCopySuccess(false), 2000);
      } catch (fallbackError) {
        console.error("Fallback copy failed:", fallbackError);
      }
      document.body.removeChild(textArea);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="mx-auto max-w-md w-full bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-800">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-500/10 rounded-lg">
              <FaShare className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <DialogTitle className="text-lg font-semibold text-gray-900 dark:text-white">
              Share Speed Test Results
            </DialogTitle>
          </div>
        </DialogHeader>

              <div className="bg-gray-100/50 dark:bg-gray-800/30 rounded-xl p-4 border border-gray-200 dark:border-gray-700/50 mb-6">
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  Share some proof that your gigabit connection is more like
                  maybe-a-bit.
                </p>
              </div>
              <div>
                <Label className="mb-3">
                  🔗 Public Dashboard Link
                </Label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={generatePublicUrl()}
                    readOnly
                    className="flex-1 px-3 py-2.5 cursor-default bg-gray-100 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50"
                  />
                  <motion.button
                    onClick={handleCopyUrl}
                    className={`px-4 py-2.5 rounded-lg transition-all duration-200 flex items-center gap-2 min-w-[44px] h-[42px] justify-center ${
                      copySuccess
                        ? "bg-green-500/20 border border-green-500/50 text-green-600 dark:text-green-400"
                        : "bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 hover:border-blue-500/50 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
                    }`}
                    aria-label="Copy URL to clipboard"
                  >
                    {copySuccess ? (
                      <FaCheck className="w-4 h-4" />
                    ) : (
                      <FaCopy className="w-4 h-4" />
                    )}
                  </motion.button>
                </div>
                <div className="h-5 mt-2">
                  {copySuccess && (
                    <motion.p
                      initial={{ opacity: 0, y: -10 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -10 }}
                      className="text-green-600 dark:text-green-400 text-xs flex items-center gap-1"
                    >
                      <FaCheck className="w-3 h-3" />
                      URL copied to clipboard!
                    </motion.p>
                  )}
                </div>
              </div>
      </DialogContent>
    </Dialog>
  );
}
