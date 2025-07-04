/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, Fragment } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/solid";
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
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div
            className="fixed inset-0 bg-black/25 dark:bg-black/80 backdrop-blur-sm"
            aria-hidden="true"
          />
        </Transition.Child>

        <div className="fixed inset-0 flex items-center justify-center p-4">
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0 scale-95"
            enterTo="opacity-100 scale-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100 scale-100"
            leaveTo="opacity-0 scale-95"
          >
            <Dialog.Panel className="mx-auto max-w-md w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-850/95 backdrop-blur-xl p-6 shadow-xl transform transition-all relative">
              <button
                onClick={onClose}
                className="absolute top-4 right-4 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
                aria-label="Close dialog"
              >
                <XMarkIcon className="h-6 w-6" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-blue-500/10 rounded-lg">
                  <FaShare className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                </div>
                <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-white">
                  Share Speed Test Results
                </Dialog.Title>
              </div>

              <p className="text-gray-600 dark:text-gray-300 text-sm mb-6">
                Share some proof that your gigabit connection is more like
                maybe-a-bit.
              </p>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Public URL
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={generatePublicUrl()}
                      readOnly
                      className="flex-1 px-3 py-2 bg-gray-100 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50"
                    />
                    <motion.button
                      onClick={handleCopyUrl}
                      className={`px-3 py-2 rounded-lg transition-all duration-200 flex items-center gap-2 ${
                        copySuccess
                          ? "bg-green-500/20 border-green-500/50 text-green-600 dark:text-green-400"
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
                        className="text-green-600 dark:text-green-400 text-xs"
                      >
                        URL copied to clipboard!
                      </motion.p>
                    )}
                  </div>
                </div>
              </div>

              <div className="mt-2 flex justify-end">
                <button
                  onClick={onClose}
                  className="px-4 py-0 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                >
                  Close
                </button>
              </div>
            </Dialog.Panel>
          </Transition.Child>
        </div>
      </Dialog>
    </Transition>
  );
}
