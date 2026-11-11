/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useServerVersion } from "@/hooks/useServerVersion";

const UPDATE_AGENT_DOCS_URL =
  "https://netrono.me/monitoring/agents/#update-an-agent";

const stopPropagation = (event: React.SyntheticEvent) =>
  event.stopPropagation();

export const OutdatedAgentBadge: React.FC<{ agentVersion: string }> = ({
  agentVersion,
}) => {
  const serverVersion = useServerVersion();

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge
          variant="warning"
          tabIndex={0}
          className="px-2 py-0.5 flex-shrink-0 cursor-default"
          onClick={stopPropagation}
        >
          Outdated
        </Badge>
      </TooltipTrigger>
      <TooltipContent onClick={stopPropagation}>
        <p>
          Agent {agentVersion}
          {serverVersion && `, server ${serverVersion}`}
        </p>
        <a
          href={UPDATE_AGENT_DOCS_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="underline"
        >
          How to update an agent
        </a>
      </TooltipContent>
    </Tooltip>
  );
};
