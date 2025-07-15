/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { Dialog, Transition } from "@headlessui/react";
import { Fragment } from "react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { VnstatAgent } from "@/api/vnstat";
import { VnstatUsageTable } from "./VnstatUsageTable";
import { getAgentIcon } from "@/utils/agentIcons";
import { useVnstatAgent } from "@/hooks/useVnstatAgent";


interface VnstatUsageModalProps {
  isOpen: boolean;
  onClose: () => void;
  agent: VnstatAgent;
}

export const VnstatUsageModal: React.FC<VnstatUsageModalProps> = ({
  isOpen,
  onClose,
  agent,
}) => {
  const { status } = useVnstatAgent({ agent });
  const { icon: AgentIcon } = getAgentIcon(agent.name);

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
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
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
              <Dialog.Panel className="w-full max-w-4xl transform overflow-hidden rounded-2xl bg-white dark:bg-gray-900 p-6 text-left align-middle shadow-xl transition-all">
                {/* Header */}
                <div className="flex items-center justify-between mb-6">
                  <div className="flex items-center space-x-3">
                    {status?.connected ? (
                      <span className="relative inline-flex h-3 w-3 flex-shrink-0">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                        <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
                      </span>
                    ) : (
                      <div className="h-3 w-3 rounded-full flex-shrink-0 bg-red-500" />
                    )}
                    <AgentIcon className="h-6 w-6 text-gray-600 dark:text-gray-400" />
                    <div>
                      <Dialog.Title
                        as="h3"
                        className="text-lg font-semibold leading-6 text-gray-900 dark:text-white"
                      >
                        {agent.name}
                      </Dialog.Title>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Usage History
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    className="rounded-md p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-300 hover:bg-gray-200/50 dark:hover:bg-gray-800/50 transition-colors"
                    onClick={onClose}
                  >
                    <span className="sr-only">Close</span>
                    <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                  </button>
                </div>

                {/* Content */}
                <div className="max-h-96 overflow-y-auto rounded-lg border border-gray-200 dark:border-gray-800">
                  <VnstatUsageTable agentId={agent.id} />
                </div>
            </Dialog.Panel>
          </Transition.Child>
        </div>
      </Dialog>
    </Transition>
  );
};