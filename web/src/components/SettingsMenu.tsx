/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useRef, useEffect, Fragment } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Dialog, Transition } from "@headlessui/react";
import {
  Cog6ToothIcon,
  XMarkIcon,
  BellIcon,
  ChevronRightIcon,
} from "@heroicons/react/24/outline";
import { NotificationSettings } from "./settings/NotificationSettings";

interface SettingsSection {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  component: React.ComponentType;
}

const settingsSections: SettingsSection[] = [
  {
    id: "notifications",
    label: "Notifications",
    icon: BellIcon,
    component: NotificationSettings,
  },
];

export const SettingsMenu: React.FC = () => {
  const [showMenu, setShowMenu] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [activeSection, setActiveSection] = useState<string>("notifications");
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setShowMenu(false);
      }
    };

    if (showMenu) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showMenu]);

  const handleSectionClick = (sectionId: string) => {
    setActiveSection(sectionId);
    setShowModal(true);
    setShowMenu(false);
  };

  const ActiveComponent =
    settingsSections.find((s) => s.id === activeSection)?.component ||
    NotificationSettings;

  return (
    <>
      <div className="relative" ref={menuRef}>
        <motion.button
          onClick={() => setShowMenu(!showMenu)}
          className="p-2 text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400 transition-colors"
          aria-label="Settings"
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
        >
          <Cog6ToothIcon className="w-6 h-6" />
        </motion.button>

        <AnimatePresence>
          {showMenu && (
            <motion.div
              initial={{ opacity: 0, y: -10, scale: 0.95 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -10, scale: 0.95 }}
              transition={{
                duration: 0.15,
                ease: [0.23, 1, 0.32, 1],
              }}
              className="absolute right-0 mt-2 w-56 rounded-xl bg-white dark:bg-gray-800 shadow-xl ring-1 ring-black/10 dark:ring-white/10 overflow-hidden z-50"
            >
              <div className="p-1">
                {settingsSections.map((section) => {
                  const Icon = section.icon;

                  return (
                    <motion.button
                      key={section.id}
                      onClick={() => handleSectionClick(section.id)}
                      className="w-full px-3 py-2 text-sm rounded-lg flex items-center gap-3 hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 transition-all duration-200"
                      whileHover={{ x: 2 }}
                    >
                      <Icon className="w-4 h-4" />
                      <span className="flex-1 text-left font-medium">
                        {section.label}
                      </span>
                      <ChevronRightIcon className="w-4 h-4 text-gray-400" />
                    </motion.button>
                  );
                })}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Settings Modal */}
      <Transition appear show={showModal} as={Fragment}>
        <Dialog
          as="div"
          className="relative z-50"
          onClose={() => setShowModal(false)}
        >
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-black/30 backdrop-blur-sm" />
          </Transition.Child>

          <div className="fixed inset-0 overflow-y-auto">
            <div className="flex min-h-full items-center justify-center p-4">
              <Transition.Child
                as={Fragment}
                enter="ease-out duration-300"
                enterFrom="opacity-0 scale-95"
                enterTo="opacity-100 scale-100"
                leave="ease-in duration-200"
                leaveFrom="opacity-100 scale-100"
                leaveTo="opacity-0 scale-95"
              >
                <Dialog.Panel className="w-full max-w-3xl md:max-w-5xl lg:max-w-6xl transform overflow-hidden rounded-2xl bg-white/95 dark:bg-gray-900/95 backdrop-blur-xl border border-gray-200 dark:border-gray-800 shadow-2xl">
                  <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-800">
                    <Dialog.Title className="text-xl font-semibold text-gray-900 dark:text-white">
                      Settings
                    </Dialog.Title>
                    <button
                      onClick={() => setShowModal(false)}
                      className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:scale-110 active:scale-90 transition-all"
                    >
                      <XMarkIcon className="w-5 h-5" />
                    </button>
                  </div>

                  <div className="p-4 sm:p-6 lg:p-8 max-h-[70vh] overflow-y-auto modal-scrollbar">
                    <ActiveComponent />
                  </div>
                </Dialog.Panel>
              </Transition.Child>
            </div>
          </div>
        </Dialog>
      </Transition>
    </>
  );
};
