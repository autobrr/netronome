/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Dialog, Transition } from "@headlessui/react";
import React, { Fragment, useEffect, useState } from "react";
import {
  RocketLaunchIcon,
  GlobeAltIcon,
  UserGroupIcon,
  ArrowTopRightOnSquareIcon,
  XMarkIcon,
  HeartIcon,
} from "@heroicons/react/24/solid";
import s0upIcon from "@/assets/sponsors/s0up4200.png";
import zze0sIcon from "@/assets/sponsors/zze0s.png";
import { Button } from "@/components/ui/button";

interface DonationLink {
  name: string;
  url: string;
  description: string;
  icon: string;
}

// Polar SVG component
const PolarIcon: React.FC<{ className?: string }> = ({ className = "w-full h-full" }) => (
  <svg viewBox="-0.5 -0.5 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M7.5 14.337187499999999C3.7239375000000003 14.337187499999999 0.6628125 11.276062499999998 0.6628125 7.5 0.6628125 3.7239375000000003 3.7239375000000003 0.6628125 7.5 0.6628125c3.7760624999999997 0 6.837187500000001 3.061125 6.837187500000001 6.837187500000001 0 3.7760624999999997 -3.061125 6.837187500000001 -6.837187500000001 6.837187500000001Z" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
    <path d="M7.5 14.337187499999999c-1.5104375 0 -2.7348749999999997 -3.061125 -2.7348749999999997 -6.837187500000001C4.765125 3.7239375000000003 5.9895625 0.6628125 7.5 0.6628125c1.5103749999999998 0 2.7348749999999997 3.061125 2.7348749999999997 6.837187500000001 0 3.7760624999999997 -1.2245 6.837187500000001 -2.7348749999999997 6.837187500000001Z" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
    <path d="M5.4488125 13.653500000000001c-2.051125 -0.6837500000000001 -2.7348749999999997 -3.6845624999999997 -2.7348749999999997 -5.811625 0 -2.1270625 1.025625 -4.7860625 3.418625 -6.495375" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
    <path d="M9.551187500000001 1.3464999999999998c2.051125 0.6837500000000001 2.7348749999999997 3.6846250000000005 2.7348749999999997 5.811625 0 2.1270625 -1.025625 4.7860625 -3.418625 6.495375" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" />
  </svg>
);

const donationLinks: DonationLink[] = [
  {
    name: "Polar",
    url: "https://buy.polar.sh/polar_cl_wWoEUigSOTJIoTrKaGIj3NU6oOCc4xJsKnsDN3NaATF",
    description: "Support netronome development via Polar.sh",
    icon: "polar-svg", // Special identifier for SVG
  },
  {
    name: "s0up",
    url: "https://github.com/sponsors/s0up4200/",
    description: "Support netronome development via GitHub Sponsors",
    icon: s0upIcon,
  },
  {
    name: "zze0s",
    url: "https://github.com/sponsors/zze0s",
    description: "Support netronome development via GitHub Sponsors",
    icon: zze0sIcon,
  },
];

interface DonateModalProps {
  isOpen: boolean;
  onClose: () => void;
}

function shuffleArray<T>(array: T[]): T[] {
  const shuffled = [...array];
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
  }
  return shuffled;
}

export function DonateModal({ isOpen, onClose }: DonateModalProps) {
  const [shuffledLinks, setShuffledLinks] = useState(donationLinks);

  useEffect(() => {
    if (isOpen) {
      setShuffledLinks(shuffleArray(donationLinks));
    }
  }, [isOpen]);

  // Preload images
  useEffect(() => {
    donationLinks.forEach((link) => {
      const img = new Image();
      img.src = link.icon;
    });
  }, []);

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
            className="fixed inset-0 bg-black/80 backdrop-blur-xs"
            aria-hidden="true"
          />
        </Transition.Child>

        <div className="fixed inset-0 flex items-start justify-center p-4 sm:pt-[20vh]">
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0 scale-95"
            enterTo="opacity-100 scale-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100 scale-100"
            leaveTo="opacity-0 scale-95"
          >
            <Dialog.Panel className="mx-auto max-w-md rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 backdrop-blur-xl p-6 shadow-xl transform transition-all relative">
              <Button
                onClick={onClose}
                variant="ghost"
                size="icon"
                className="absolute top-4 right-4 h-8 w-8 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
                aria-label="Close dialog"
              >
                <XMarkIcon className="h-6 w-6" />
              </Button>
              
              <Dialog.Title className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
                Support Netronome
              </Dialog.Title>
              
              <div className="mb-6 space-y-4">
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">
                  Your donations directly contribute to:
                </p>
                <ul className="list-none space-y-3 text-sm text-gray-600 dark:text-gray-300">
                  <li className="flex items-center gap-3">
                    <RocketLaunchIcon className="h-4 w-4 text-blue-500 flex-shrink-0" />
                    <a
                      href="https://github.com/autobrr/netronome/issues"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-blue-500 dark:hover:text-blue-400 transition-colors inline-flex items-center gap-1"
                    >
                      New features & improvements
                      <ArrowTopRightOnSquareIcon className="h-3 w-3" />
                    </a>
                  </li>
                  <li className="flex items-center gap-3">
                    <GlobeAltIcon className="h-4 w-4 text-emerald-500 flex-shrink-0" />
                    Infrastructure costs
                  </li>
                  <li className="flex items-center gap-3">
                    <UserGroupIcon className="h-4 w-4 text-purple-500 flex-shrink-0" />
                    <a
                      href="https://discord.gg/WQ2eUycxyT"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-purple-500 dark:hover:text-purple-400 transition-colors inline-flex items-center gap-1"
                    >
                      Community support
                      <ArrowTopRightOnSquareIcon className="h-3 w-3" />
                    </a>
                  </li>
                </ul>
                <p className="text-sm text-gray-500 dark:text-gray-400 pt-3 flex items-center gap-1">
                  Thank you <HeartIcon className="h-4 w-4 text-red-500" />
                </p>
              </div>
              
              <div className="space-y-3">
                {shuffledLinks.map((link) => (
                  <a
                    key={link.name}
                    href={link.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="group flex items-center gap-4 p-4 rounded-lg bg-gray-50/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-800 hover:bg-gray-100/70 dark:hover:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-700 transition-colors"
                  >
                    <div className="w-12 h-12 rounded-xl bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 flex items-center justify-center p-2 shadow-sm group-hover:shadow-md transition-shadow duration-200">
                      {link.icon === "polar-svg" ? (
                        <PolarIcon className="w-full h-full text-gray-900 dark:text-white" />
                      ) : (
                        <img
                          src={link.icon}
                          alt={`${link.name} icon`}
                          className="w-full h-full rounded-lg object-cover"
                          onError={(e) => {
                            // Fallback for failed image loads
                            const target = e.target as HTMLImageElement;
                            target.style.display = 'none';
                            const parent = target.parentElement;
                            if (parent && !parent.querySelector('.fallback-icon')) {
                              const fallback = document.createElement('div');
                              fallback.className = 'fallback-icon w-full h-full bg-blue-500 rounded-lg flex items-center justify-center text-white text-xs font-bold';
                              fallback.textContent = link.name.charAt(0).toUpperCase();
                              parent.appendChild(fallback);
                            }
                          }}
                        />
                      )}
                    </div>
                    <div>
                      <h3 className="font-medium text-gray-900 dark:text-white group-hover:text-blue-500 dark:group-hover:text-blue-400 flex items-center gap-1 transition-colors">
                        {link.name}
                        <ArrowTopRightOnSquareIcon className="h-3 w-3 text-gray-400 group-hover:text-blue-500 dark:group-hover:text-blue-400 transition-colors" />
                      </h3>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        {link.description}
                      </p>
                    </div>
                  </a>
                ))}
              </div>
            </Dialog.Panel>
          </Transition.Child>
        </div>
      </Dialog>
    </Transition>
  );
}