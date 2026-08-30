/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef } from "react";
import { toast } from "@/components/common/Toast";
import { formatReleaseTag, type AppRelease } from "@/api/version";

const DISMISSED_RELEASE_STORAGE_KEY = "netronome.dismissedReleaseTag";

const getDismissedReleaseTag = (): string | null => {
  try {
    return localStorage.getItem(DISMISSED_RELEASE_STORAGE_KEY);
  } catch {
    return null;
  }
};

const dismissReleaseTag = (tagName: string) => {
  try {
    localStorage.setItem(DISMISSED_RELEASE_STORAGE_KEY, tagName);
  } catch {
    // Storage can be unavailable in private browsing modes; the current toast
    // still closes and a future session can try again.
  }
};

const openRelease = (url: string) => {
  const releaseWindow = window.open(url, "_blank", "noopener,noreferrer");
  if (releaseWindow) releaseWindow.opener = null;
};

/** Persistent release notice. The menu markers remain after it is dismissed. */
export function AppUpdatePrompt({ release }: { release: AppRelease | null }) {
  const toastIdRef = useRef<string | number | null>(null);
  const shownTagRef = useRef<string | null>(null);
  const programmaticDismissalsRef = useRef(new Set<string | number>());

  useEffect(() => {
    const dismissCurrentToast = () => {
      const toastId = toastIdRef.current;
      if (toastId === null) return;

      programmaticDismissalsRef.current.add(toastId);
      toast.dismiss(toastId);
      toastIdRef.current = null;
      shownTagRef.current = null;
    };

    if (!release || getDismissedReleaseTag() === release.tag_name) {
      dismissCurrentToast();
      return;
    }

    if (toastIdRef.current !== null) {
      if (shownTagRef.current === release.tag_name) return;
      dismissCurrentToast();
    }

    const toastId = toast.info(
      `Netronome ${formatReleaseTag(release.tag_name)} is available`,
      {
        duration: Infinity,
        action: {
          label: "View release",
          onClick: () => {
            openRelease(release.html_url);
            dismissReleaseTag(release.tag_name);
            toastIdRef.current = null;
            shownTagRef.current = null;
          },
        },
        cancel: {
          label: "Later",
          onClick: () => {
            dismissReleaseTag(release.tag_name);
            toastIdRef.current = null;
            shownTagRef.current = null;
          },
        },
        onDismiss: () => {
          if (!programmaticDismissalsRef.current.delete(toastId)) {
            dismissReleaseTag(release.tag_name);
          }
          if (toastIdRef.current === toastId) {
            toastIdRef.current = null;
            shownTagRef.current = null;
          }
        },
      }
    );

    toastIdRef.current = toastId;
    shownTagRef.current = release.tag_name;
  }, [release]);

  useEffect(
    () => () => {
      const toastId = toastIdRef.current;
      if (toastId === null) return;

      programmaticDismissalsRef.current.add(toastId);
      toast.dismiss(toastId);
      toastIdRef.current = null;
      shownTagRef.current = null;
    },
    []
  );

  return null;
}
