/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import {
  ComputerDesktopIcon,
  DevicePhoneMobileIcon,
  ServerIcon,
  ServerStackIcon,
  CpuChipIcon,
  CircleStackIcon,
  CloudIcon,
  GlobeAltIcon,
  WifiIcon,
  TvIcon,
  HomeIcon,
} from "@heroicons/react/24/outline";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faLaptop } from "@fortawesome/free-solid-svg-icons";
import { IconProp } from "@fortawesome/fontawesome-svg-core";

export type IconComponent = React.FC<{ className?: string }>;

// Helper function to create Font Awesome icon components
const createFAIcon = (icon: IconProp): IconComponent => {
  return ({ className }) => (
    <FontAwesomeIcon icon={icon} className={className} />
  );
};

interface DevicePattern {
  patterns: RegExp[];
  icon: IconComponent;
  description: string;
}

const devicePatterns: DevicePattern[] = [
  // Laptop patterns (separate from desktop for better icon matching)
  {
    patterns: [
      /\b(laptop|macbook|notebook|thinkpad|ideapad|pavilion|inspiron|latitude|xps|chromebook)\b/i,
      /\b(macbook pro|macbook air|surface laptop)\b/i,
    ],
    icon: createFAIcon(faLaptop),
    description: "Laptop",
  },

  // Desktop patterns
  {
    patterns: [
      /\b(pc|desktop|computer|workstation|imac|mac mini|tower)\b/i,
      /\b(windows|win\d+|linux desktop|ubuntu desktop)\b/i,
    ],
    icon: ComputerDesktopIcon,
    description: "Desktop Computer",
  },

  // Mobile device patterns
  {
    patterns: [
      /\b(phone|mobile|android|ios|iphone|ipad|tablet|galaxy|pixel)\b/i,
      /\b(oneplus|xiaomi|huawei|oppo|vivo|realme)\b/i,
    ],
    icon: DevicePhoneMobileIcon,
    description: "Mobile Device",
  },

  // TV/Media device patterns
  {
    patterns: [
      /\b(tv|television|roku|chromecast|apple tv|fire tv|shield|kodi)\b/i,
      /\b(media center|htpc|plex|emby|jellyfin)\b/i,
    ],
    icon: TvIcon,
    description: "TV or Media Device",
  },

  // Router/Network device patterns
  {
    patterns: [
      /\b(router|gateway|firewall|switch|ap|access point|wifi|mesh)\b/i,
      /\b(openwrt|ddwrt|pfsense|opnsense|mikrotik|ubiquiti|unifi)\b/i,
    ],
    icon: WifiIcon,
    description: "Network Device",
  },

  // NAS/Storage patterns
  {
    patterns: [
      /\b(nas|storage|synology|qnap|freenas|truenas|unraid|proxmox|omv|pbs|pve|pve01|pve02|pve03|pve04|pve05|pve06|pve07|pve08|pve09|pve10)\b/i,
      /\b(backup|archive|vault)\b/i,
    ],
    icon: CircleStackIcon,
    description: "Storage Device",
  },

  // Cloud/VPS patterns
  {
    patterns: [
      /\b(cloud|vps|aws|azure|gcp|digitalocean|linode|vultr)\b/i,
      /\b(ec2|droplet|instance|virtual machine|vm)\b/i,
    ],
    icon: CloudIcon,
    description: "Cloud Server",
  },

  // Raspberry Pi/SBC patterns
  {
    patterns: [
      /\b(raspberry|rpi|raspi|pine64|orange pi|odroid|rock pi|nano pi)\b/i,
      /\b(sbc|single board|arm board)\b/i,
    ],
    icon: CpuChipIcon,
    description: "Single Board Computer",
  },

  // Home/Smart Home patterns
  {
    patterns: [
      /\b(home|house|apartment|office|room|bedroom|living room|kitchen)\b/i,
      /\b(homeassistant|home assistant|smart home|iot|hub)\b/i,
    ],
    icon: HomeIcon,
    description: "Home or Location",
  },

  // Web/Domain patterns
  {
    patterns: [
      /\b(web|www|site|domain|blog|portal|cdn)\b/i,
      /\.(com|net|org|io|dev|app)\b/i,
    ],
    icon: GlobeAltIcon,
    description: "Website or Domain",
  },

  // Database/Service patterns
  {
    patterns: [
      /\b(database|db|mysql|postgres|mongodb|redis|elastic)\b/i,
      /\b(service|api|backend|frontend|proxy|cache)\b/i,
    ],
    icon: ServerStackIcon,
    description: "Database or Service",
  },
];

/**
 * Get an appropriate icon based on the agent name
 * @param name The agent name to analyze
 * @returns An icon component and description
 */
export function getAgentIcon(name: string): {
  icon: IconComponent;
  description: string;
} {
  const lowerName = name.toLowerCase();

  // Check each pattern group
  for (const device of devicePatterns) {
    for (const pattern of device.patterns) {
      if (pattern.test(lowerName)) {
        return {
          icon: device.icon,
          description: device.description,
        };
      }
    }
  }

  // Default to generic server icon
  return {
    icon: ServerIcon,
    description: "Server",
  };
}

/**
 * Render an agent icon with consistent styling
 * @param name The agent name
 * @param className Optional additional CSS classes
 */
export function AgentIcon({
  name,
  className = "h-10 w-10 text-gray-400",
}: {
  name: string;
  className?: string;
}) {
  const { icon: Icon } = getAgentIcon(name);
  return <Icon className={className} />;
}
