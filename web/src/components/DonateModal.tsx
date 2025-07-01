/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Dialog, Transition } from "@headlessui/react";
import { Fragment, useEffect, useState } from "react";
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

interface DonationLink {
  name: string;
  url: string;
  description: string;
  icon: string;
}

const donationLinks: DonationLink[] = [
  {
    name: "Polar",
    url: "https://buy.polar.sh/polar_cl_wWoEUigSOTJIoTrKaGIj3NU6oOCc4xJsKnsDN3NaATF",
    description: "Support netronome development via Polar.sh",
    icon: "https://polar.sh/favicon.ico",
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
            <Dialog.Panel className="mx-auto max-w-md rounded-xl border-2 border-gray-200 dark:border-black/40 bg-white dark:bg-gray-500/10 backdrop-blur-xl p-6 shadow-xl transform transition-all relative">
              <button
                onClick={onClose}
                className="absolute top-4 right-4 text-gray-400 hover:text-gray-500 dark:text-gray-600 dark:hover:text-gray-400 transition-colors"
                aria-label="Close dialog"
              >
                <XMarkIcon className="h-6 w-6" />
              </button>
              <Dialog.Title className="text-lg font-medium leading-6 text-gray-900 dark:text-white mb-4">
                Support Netronome
              </Dialog.Title>
              <div className="mb-6 space-y-4 p-1">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Your donations directly contribute to:
                </p>
                <ul className="list-none space-y-3 text-sm text-gray-600 dark:text-gray-300">
                  <li className="flex items-center gap-2">
                    <RocketLaunchIcon className="h-4 w-4 text-blue-500" />
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
                  <li className="flex items-center gap-2">
                    <GlobeAltIcon className="h-4 w-4 text-green-500" />
                    Infrastructure costs
                  </li>
                  <li className="flex items-center gap-2">
                    <UserGroupIcon className="h-4 w-4 text-purple-500" />
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
                <p className="text-sm text-gray-500 dark:text-gray-400 pt-2 flex items-center gap-1">
                  Thank you <HeartIcon className="h-4 w-4 text-red-500" />
                </p>
              </div>
              <div className="space-y-4">
                {shuffledLinks.map((link) => (
                  <a
                    key={link.name}
                    href={link.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="group flex items-center gap-4 p-4 rounded-lg dark:bg-gray-900/40 border border-gray-200 dark:border-black/60 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
                  >
                    <img
                      src={link.icon}
                      alt={`${link.name} icon`}
                      className="w-10 h-10 rounded-full object-cover"
                    />
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
