/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo } from "react";
import { ChevronRight } from "lucide-react";
import { SHOUTRRR_SERVICES } from "@/api/notifications";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

interface ChannelCardProps {
  channel: any;
  isActive: boolean;
  hasUnsavedChanges: boolean;
  onClick: () => void;
  onMouseEnter?: () => void;
}

export const ChannelCard: React.FC<ChannelCardProps> = ({
  channel,
  isActive,
  onClick,
  onMouseEnter,
}) => {
  const serviceType = useMemo(() => {
    if (!channel.url) return null;
    const match = channel.url.match(/^(\w+):\/\//);
    return match ? match[1] : null;
  }, [channel.url]);

  const serviceInfo = SHOUTRRR_SERVICES.find((s) => s.value === serviceType);

  return (
    <Card
      className={cn(
        "p-0 cursor-pointer transition-all",
        isActive
          ? "bg-blue-500/10 dark:bg-blue-500/20 border-blue-500/50 dark:border-blue-400/60 shadow-md ring-2 ring-blue-500/20 dark:ring-blue-400/30"
          : "hover:border-gray-300 dark:hover:border-gray-700 hover:shadow-md"
      )}
    >
      <Button
        variant="ghost"
        className={cn(
          "w-full h-auto p-4 justify-start hover:bg-transparent",
          isActive && "hover:bg-blue-500/10 dark:hover:bg-blue-500/20"
        )}
        onClick={onClick}
        onMouseEnter={onMouseEnter}
      >
        <div className="flex items-start justify-between w-full">
          <div className="flex-1 min-w-0 text-left">
            <div className="flex items-center gap-2 mb-1">
              <div
                className={cn(
                  "w-2 h-2 rounded-full flex-shrink-0",
                  channel.enabled ? "bg-emerald-500" : "bg-gray-400"
                )}
              />
              <h5
                className={cn(
                  "font-medium truncate",
                  isActive
                    ? "text-blue-700 dark:text-blue-300"
                    : "text-gray-900 dark:text-white"
                )}
              >
                {channel.name}
              </h5>
            </div>
            <p className="text-xs text-gray-600 dark:text-gray-400 truncate">
              {serviceInfo?.label || serviceType || "Unknown service"}
            </p>
          </div>
          <ChevronRight
            className={cn(
              "w-4 h-4 flex-shrink-0 transition-colors",
              isActive ? "text-blue-600 dark:text-blue-400" : "text-gray-400"
            )}
          />
        </div>
      </Button>
    </Card>
  );
};
