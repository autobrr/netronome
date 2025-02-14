/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGithub } from "@fortawesome/free-brands-svg-icons";
import { BookOpenIcon } from "@heroicons/react/24/solid";

export const Footer = () => {
  return (
    <footer className="mt-6">
      <div className="flex justify-center space-x-4">
        <a
          href="https://netrono.me"
          target="_blank"
          rel="noopener noreferrer"
          className="text-gray-400 hover:text-gray-300 transition-colors"
          title="Documentation"
        >
          <BookOpenIcon className="h-5 w-5" />
        </a>
        <a
          href="https://github.com/autobrr/netronome"
          target="_blank"
          rel="noopener noreferrer"
          className="text-gray-400 hover:text-gray-300 transition-colors"
          title="View on GitHub"
        >
          <FontAwesomeIcon icon={faGithub} className="h-5 w-5" />
        </a>
      </div>
    </footer>
  );
}; 