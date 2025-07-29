/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { Cog6ToothIcon, BellIcon } from "@heroicons/react/24/outline";
import { NotificationSettings } from "./settings/NotificationSettings";
import { Button } from "@/components/ui/Button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface SettingsSection {
  id: string;
  label: string;
  icon: React.ReactNode;
  component: React.ComponentType;
}

const settingsSections: SettingsSection[] = [
  {
    id: "notifications",
    label: "Notifications",
    icon: <BellIcon className="w-4 h-4" />,
    component: NotificationSettings,
  },
];

export const SettingsMenu: React.FC = () => {
  const [showModal, setShowModal] = useState(false);
  const [activeSection, setActiveSection] = useState<string>("notifications");

  const handleSectionClick = (sectionId: string) => {
    setActiveSection(sectionId);
    setShowModal(true);
  };

  const ActiveComponent =
    settingsSections.find((s) => s.id === activeSection)?.component ||
    NotificationSettings;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="text-gray-600 dark:text-gray-600 hover:text-gray-900 dark:hover:text-gray-400"
            aria-label="Settings"
          >
            <Cog6ToothIcon className="w-6 h-6" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          className="w-56 bg-white dark:bg-gray-800 shadow-xl ring-1 ring-black/10 dark:ring-white/10"
        >
          {settingsSections.map((section) => (
            <DropdownMenuItem
              key={section.id}
              onClick={() => handleSectionClick(section.id)}
              className="cursor-pointer px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <div className="flex items-center gap-3 w-full">
                {section.icon}
                <span className="flex-1 text-left font-medium">
                  {section.label}
                </span>
              </div>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={showModal} onOpenChange={setShowModal}>
        <DialogContent
          className="w-[calc(100%-1rem)] max-w-[calc(100%-1rem)] sm:w-full sm:max-w-3xl md:max-w-5xl lg:max-w-6xl bg-white/95 dark:bg-gray-900/95 backdrop-blur-xl border border-gray-200 dark:border-gray-800 shadow-2xl !p-0 gap-0"
          showCloseButton
        >
          <DialogHeader className="p-6 border-b border-gray-200 dark:border-gray-800">
            <DialogTitle className="text-xl font-semibold text-gray-900 dark:text-white">
              Settings
            </DialogTitle>
          </DialogHeader>
          <div className="p-0 sm:p-6 lg:p-8 max-h-[70vh] overflow-y-auto modal-scrollbar">
            <ActiveComponent />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};
