/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { badgeVariants } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useServerVersion } from "@/hooks/useServerVersion";
import { cn } from "@/lib/utils";

const UPDATE_AGENT_DOCS_URL =
  "https://netrono.me/monitoring/agents/#update-an-agent";

const stopPropagation = (event: React.SyntheticEvent) =>
  event.stopPropagation();

export const OutdatedAgentBadge: React.FC<{ agentVersion: string }> = ({
  agentVersion,
}) => {
  const serverVersion = useServerVersion();

  return (
    <Popover>
      <PopoverTrigger
        className={cn(
          badgeVariants({ variant: "warning" }),
          "px-2 py-0.5 flex-shrink-0 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/50"
        )}
        onClick={stopPropagation}
      >
        Outdated
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-auto max-w-72 p-3 text-xs"
        onClick={stopPropagation}
      >
        <p>
          Agent {agentVersion}
          {serverVersion && `, server ${serverVersion}`}
        </p>
        <a
          href={UPDATE_AGENT_DOCS_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-1 inline-block text-blue-600 underline dark:text-blue-400"
        >
          How to update an agent
        </a>
      </PopoverContent>
    </Popover>
  );
};
