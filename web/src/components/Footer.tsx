/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faReadme, faDiscord } from "@fortawesome/free-brands-svg-icons";

export const Footer = () => {
  return (
    <footer className="mt-4">
      <div className="flex justify-end space-x-4">
        <a
          href="https://discord.gg/WQ2eUycxyT"
          target="_blank"
          rel="noopener noreferrer"
          className="text-gray-400 hover:text-blue-500 transition-colors"
          title="Discord"
          style={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <FontAwesomeIcon icon={faDiscord} className="h-4 w-4" />
        </a>
        <a
          href="https://netrono.me"
          target="_blank"
          rel="noopener noreferrer"
          className="text-gray-400 hover:text-blue-500 transition-colors"
          title="Documentation"
          style={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <FontAwesomeIcon icon={faReadme} className="h-4 w-4" />
        </a>
      </div>
    </footer>
  );
};
