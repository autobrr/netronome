/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";

// Manual PWA implementation using workbox-window since virtual module isn't working with Vite 7
export function PWAUpdatePrompt() {
  const [needRefresh, setNeedRefresh] = useState(false);
  const [registration, setRegistration] = useState<ServiceWorkerRegistration | null>(null);

  const updateServiceWorker = useCallback(async (reloadPage = false) => {
    if (registration?.waiting) {
      registration.waiting.postMessage({ type: 'SKIP_WAITING' });
      if (reloadPage) {
        window.location.reload();
      }
    }
  }, [registration]);

  useEffect(() => {
    const registerSW = async () => {
      if ('serviceWorker' in navigator) {
        try {
          // Import workbox-window dynamically to avoid build issues
          const { Workbox } = await import('workbox-window');
          
          const wb = new Workbox('/sw.js');

          wb.addEventListener('controlling', () => {
            window.location.reload();
          });

          wb.addEventListener('waiting', () => {
            setNeedRefresh(true);
          });

          wb.addEventListener('externalwaiting' as any, () => {
            setNeedRefresh(true);
          });

          // Service worker installed successfully (no toast notification needed)

          const reg = await wb.register();
          setRegistration(reg || null);

          console.log('SW Registered:', reg);
        } catch (error) {
          console.log('SW registration error', error);
        }
      }
    };

    registerSW();
  }, []);

  const close = useCallback(() => {
    setNeedRefresh(false);
  }, []);

  useEffect(() => {
    if (needRefresh) {
      toast("New content available, click on reload button to update.", {
        duration: Infinity,
        action: {
          label: "Reload",
          onClick: () => updateServiceWorker(true),
        },
        cancel: {
          label: "Later",
          onClick: close,
        },
      });
    }
  }, [needRefresh, updateServiceWorker, close]);

  return null;
}
