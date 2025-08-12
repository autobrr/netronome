/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState, useCallback, useRef } from "react";
import { toast } from "@/components/common/Toast";

// Manual PWA implementation using workbox-window
export function PWAUpdatePrompt() {
  const [needRefresh, setNeedRefresh] = useState(false);
  const [registration, setRegistration] =
    useState<ServiceWorkerRegistration | null>(null);
  const toastIdRef = useRef<string | number | null>(null);



  const updateServiceWorker = useCallback(async () => {
    if (registration?.waiting) {
      // Clear the toast first
      if (toastIdRef.current) {
        toast.dismiss(toastIdRef.current);
        toastIdRef.current = null;
      }
      
      // Set a flag to prevent showing the prompt again after reload
      sessionStorage.setItem('sw-just-updated', 'true');
      
      // Post message to service worker to skip waiting
      registration.waiting.postMessage({ type: 'SKIP_WAITING' });
      
      // Set up listener for controlling change - use once to avoid multiple listeners
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        window.location.reload();
      }, { once: true });
    }
  }, [registration]);

  useEffect(() => {
    const registerSW = async () => {
      if ("serviceWorker" in navigator) {
        try {
          // Import workbox-window dynamically to avoid build issues
          const { Workbox } = await import("workbox-window");

          const wb = new Workbox("/sw.js");

          // Show update prompt when new service worker is waiting
          const showSkipWaitingPrompt = () => {
            setNeedRefresh(true);
          };

          wb.addEventListener("waiting", showSkipWaitingPrompt);
          // @ts-expect-error - externalwaiting is a valid event but not in the type definitions
          wb.addEventListener("externalwaiting", showSkipWaitingPrompt);

          // Register the service worker
          const reg = await wb.register();
          setRegistration(reg || null);

          // Check if there's already a waiting service worker
          // But skip if we just reloaded (check if the page was reloaded recently)
          if (reg?.waiting) {
            // Check if we have a flag in sessionStorage indicating we just updated
            const justUpdated = sessionStorage.getItem('sw-just-updated');
            if (!justUpdated) {
              showSkipWaitingPrompt();
            } else {
              // Clear the flag after a short delay
              sessionStorage.removeItem('sw-just-updated');
            }
          }


        } catch (error) {
          console.log("SW registration error", error);
        }
      }
    };

    registerSW();
  }, []);

  useEffect(() => {
    if (needRefresh && !toastIdRef.current) {
      // Use the raw toast API for custom cancel button
      toastIdRef.current = toast.info(
        "New content available, click on reload button to update.",
        {
          duration: Infinity,
          action: {
            label: "Reload",
            onClick: () => {
              updateServiceWorker();
            },
          },
          cancel: {
            label: "Later",
            onClick: () => {
              setNeedRefresh(false);
              if (toastIdRef.current) {
                toast.dismiss(toastIdRef.current);
                toastIdRef.current = null;
              }
            },
          },
        }
      );
    }
  }, [needRefresh, updateServiceWorker]);

  return null;
}
