/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { ArrowTopRightOnSquareIcon } from "@heroicons/react/24/outline";
import { formatReleaseTag, type AppRelease } from "@/api/version";
import { formatVersion } from "@/api/formatVersion";
import { useServerVersion } from "@/hooks/useServerVersion";
import { cn } from "@/lib/utils";

export const AvailableReleaseLink: React.FC<
  { release: AppRelease } & React.ComponentProps<"a">
> = ({ release, className, ...props }) => {
  const tag = formatReleaseTag(release.tag_name);

  return (
    <a
      href={release.html_url}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={`View Netronome ${tag} release`}
      {...props}
      className={cn(className, "items-center gap-3 whitespace-normal")}
    >
      <ArrowTopRightOnSquareIcon className="h-4 w-4 flex-shrink-0 text-blue-600 dark:text-blue-400" />
      <span className="flex min-w-0 flex-1 flex-col items-start gap-0.5 text-left">
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Netronome {tag} available
        </span>
        <span className="text-xs text-blue-600 dark:text-blue-400">
          View release
        </span>
      </span>
    </a>
  );
};

export const RunningVersion: React.FC<{
  className?: string;
  divider?: React.ReactNode;
}> = ({ className, divider }) => {
  const serverVersion = useServerVersion();
  if (!serverVersion) return null;

  return (
    <>
      {divider}
      <p
        className={cn("text-xs text-gray-500 dark:text-gray-400", className)}
      >
        Netronome {formatVersion(serverVersion)}
      </p>
    </>
  );
};
