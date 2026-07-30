/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import {
  Cog6ToothIcon,
  BellIcon,
  ClockIcon,
  MapPinIcon,
  PresentationChartLineIcon,
  CircleStackIcon,
  SwatchIcon,
} from "@heroicons/react/24/outline";
import { NotificationSettings } from "./settings/NotificationSettings";
import { TimeFormatSettings } from "./settings/TimeFormatSettings";
import { DistanceSettings } from "./settings/DistanceSettings";
import { DashboardSettings } from "./settings/DashboardSettings";
import { DataSettings } from "./settings/DataSettings";
import { ThemeSettings } from "./settings/ThemeSettings";
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

export const settingsSections: SettingsSection[] = [
  {
    id: "notifications",
    label: "Notifications",
    icon: <BellIcon className="w-4 h-4" />,
    component: NotificationSettings,
  },
  {
    id: "time-format",
    label: "Time & Timezone",
    icon: <ClockIcon className="w-4 h-4" />,
    component: TimeFormatSettings,
  },
  {
    id: "distance",
    label: "Distance Units",
    icon: <MapPinIcon className="w-4 h-4" />,
    component: DistanceSettings,
  },
  {
    id: "dashboard",
    label: "Dashboard",
    icon: <PresentationChartLineIcon className="w-4 h-4" />,
    component: DashboardSettings,
  },
  {
    id: "data",
    label: "Data",
    icon: <CircleStackIcon className="w-4 h-4" />,
    component: DataSettings,
  },
  {
    id: "themes",
    label: "Themes & License",
    icon: <SwatchIcon className="w-4 h-4" />,
    component: ThemeSettings,
  },
];

/** Settings section dialog shared by the desktop menu and the mobile sheet. */
export const SettingsDialog: React.FC<{
  sectionId: string | null;
  onClose: () => void;
}> = ({ sectionId, onClose }) => {
  const section = settingsSections.find((s) => s.id === sectionId);

  return (
    <Dialog open={section !== undefined} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className="w-[calc(100%-1rem)] max-w-[calc(100%-1rem)] sm:w-full sm:max-w-3xl md:max-w-5xl lg:max-w-6xl bg-white/95 dark:bg-gray-900/95 backdrop-blur-xl border border-gray-200 dark:border-gray-800 shadow-2xl !p-0 gap-0"
        showCloseButton
        // Radix focuses the first focusable child on open, which lands on
        // whatever control a section happens to start with - Deactivate on
        // Themes & License. Focus the dialog itself instead, so nothing reads
        // as pre-selected while the focus trap still starts inside.
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          (event.currentTarget as HTMLElement | null)?.focus();
        }}
      >
        <DialogHeader className="p-6 border-b border-gray-200 dark:border-gray-800">
          <DialogTitle className="text-xl font-semibold text-gray-900 dark:text-white">
            {section?.label ?? "Settings"}
          </DialogTitle>
        </DialogHeader>
        <div className="p-0 sm:p-6 lg:p-8 max-h-[70vh] overflow-y-auto modal-scrollbar">
          {section && <section.component />}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export const SettingsMenu: React.FC = () => {
  const [activeSection, setActiveSection] = useState<string | null>(null);

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
              onClick={() => setActiveSection(section.id)}
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

      <SettingsDialog
        sectionId={activeSection}
        onClose={() => setActiveSection(null)}
      />
    </>
  );
};
