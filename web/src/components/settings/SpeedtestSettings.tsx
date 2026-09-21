/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowPathIcon,
  CheckIcon,
  GlobeAltIcon,
  MapPinIcon,
  ServerStackIcon,
} from "@heroicons/react/24/outline";
import {
  getSpeedtestServerCatalogue,
  getSpeedtestServerCatalogueStatus,
} from "@/api/speedtest";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { showToast } from "@/components/common/Toast";
import {
  formatSpeedtestServerName,
  formatSpeedtestServerStorageStatus,
  getSpeedtestSettings,
  isLatitude,
  isLongitude,
  normalizeSpeedtestSettings,
  saveSpeedtestSettings,
  speedtestServerCatalogueQueryKey,
  speedtestServerQueryKey,
  speedtestServerStatusQueryKey,
  speedtestServerQuery,
  type SpeedtestServerQuery,
  type SpeedtestServerSource,
  type SpeedtestSettingsDraft,
} from "@/utils/speedtestSettings";

const SOURCE_OPTIONS: Array<{
  value: SpeedtestServerSource;
  label: string;
  description: string;
}> = [
  { value: "local", label: "Local", description: "Show the 10 servers nearest your detected location." },
  { value: "global", label: "Global", description: "Show every retained server, fetched from regions worldwide." },
  { value: "coordinates", label: "Coordinates", description: "Show the 10 servers nearest a latitude and longitude." },
];

/** Configures which Speedtest.net servers are shown and how history labels are displayed. */
export const SpeedtestSettings = () => {
  const queryClient = useQueryClient();
  const refreshAbortController = useRef<AbortController | null>(null);
  const [settings, setSettings] = useState<SpeedtestSettingsDraft>(getSpeedtestSettings);
  const [hasChanges, setHasChanges] = useState(false);

  const updateSettings = (change: Partial<SpeedtestSettingsDraft>) => {
    const sourceChanged = change.source !== undefined && change.source !== settings.source;
    const coordinatesChanged = settings.source === "coordinates" &&
      (("latitude" in change && change.latitude !== settings.latitude) ||
        ("longitude" in change && change.longitude !== settings.longitude));
    if (sourceChanged || coordinatesChanged) {
      resetRefreshServerCatalogue();
    }
    setSettings((current) => ({ ...current, ...change }));
    setHasChanges(true);
  };

  const coordinatesValid =
    settings.source !== "coordinates" ||
    (isLatitude(settings.latitude) && isLongitude(settings.longitude));

  const normalizedSettings = normalizeSpeedtestSettings(settings);
  const serverQuery = speedtestServerQuery(normalizedSettings);
  const catalogueQueryKey = speedtestServerQueryKey(serverQuery);
  const statusQueryKey = speedtestServerStatusQueryKey(normalizedSettings);
  const sourceLabel = SOURCE_OPTIONS.find((option) => option.value === settings.source)?.label ?? "Selected";
  const { data: fetchedServers } = useQuery({
    queryKey: catalogueQueryKey,
    queryFn: ({ signal }) => getSpeedtestServerCatalogue(serverQuery, signal),
    enabled: false,
  });
  const {
    data: catalogueStatus,
    isError: isStatusError,
    isPending: isStatusLoading,
  } = useQuery({
    queryKey: statusQueryKey,
    queryFn: ({ signal }) => getSpeedtestServerCatalogueStatus(serverQuery, signal),
    enabled: coordinatesValid,
  });
  const {
    data: refreshResult,
    error: fetchError,
    isError: isFetchError,
    isPending: isFetching,
    mutateAsync: refreshServerCatalogue,
    reset: resetRefreshServerCatalogue,
  } = useMutation({
    mutationFn: ({ query, signal }: {
      query: SpeedtestServerQuery;
      signal: AbortSignal;
      sourceLabel: string;
    }) => getSpeedtestServerCatalogue({ ...query, refresh: true }, signal),
    onSuccess: async ({ servers, warnings }, { query, signal, sourceLabel }) => {
      if (signal.aborted) return;
      queryClient.setQueryData(speedtestServerQueryKey(query), { servers, warnings: [] });
      await queryClient.invalidateQueries({
        queryKey: speedtestServerCatalogueQueryKey(),
      });
      await queryClient.invalidateQueries({
        queryKey: ["servers", "speedtest", "status"],
      });
      if (warnings.length > 0) {
        showToast("Server catalogue partially updated", "warning", {
          description: `${sourceLabel}: ${servers.length} retained servers available; ${warnings.length} source${warnings.length === 1 ? "" : "s"} failed`,
        });
      } else {
        showToast("Server catalogue updated", "success", {
          description: `${sourceLabel}: ${servers.length} retained servers available`,
        });
      }
    },
  });

  useEffect(() => () => refreshAbortController.current?.abort(), []);

  const saveSettings = () => {
    if (!saveSpeedtestSettings(settings)) {
      showToast("Failed to save speedtest settings", "error");
      return;
    }
    setSettings(getSpeedtestSettings());
    setHasChanges(false);
    showToast("Speedtest settings saved", "success", {
      description: "Discovery and history preferences are now active",
    });
  };

  const refreshServers = async () => {
    await queryClient.cancelQueries({ queryKey: catalogueQueryKey, exact: true });
    const controller = new AbortController();
    refreshAbortController.current = controller;
    try {
      await refreshServerCatalogue({ query: serverQuery, signal: controller.signal, sourceLabel });
    } catch {
      // Mutation state keeps request failures visible beside the fetch control.
    } finally {
      if (refreshAbortController.current === controller) {
        refreshAbortController.current = null;
      }
    }
  };

  const lastUpdated = catalogueStatus?.updatedAt
    ? new Date(catalogueStatus.updatedAt).toLocaleString()
    : null;
  const sourceStored = coordinatesValid && catalogueStatus?.stored === true;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="flex items-center gap-2 text-xl font-bold text-gray-900 dark:text-white sm:text-2xl">
            <ServerStackIcon className="h-6 w-6 text-gray-600 dark:text-gray-400" />
            Speedtest.net Settings
          </h3>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Choose which servers the Speed Test page shows. Every fetch adds servers to the retained catalogue. Local and Coordinates show the nearest 10. Global shows all of them.
          </p>
        </div>
        {hasChanges && (
          <div className="flex items-center gap-2">
            <Button onClick={saveSettings} disabled={!coordinatesValid || isFetching}>
              <CheckIcon className="h-4 w-4" />
              Save Changes
            </Button>
            <Button
              variant="secondary"
              onClick={() => {
                setSettings(getSpeedtestSettings());
                setHasChanges(false);
              }}
            >
              Cancel
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GlobeAltIcon className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              Server Catalogue
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3">
              {SOURCE_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={settings.source === option.value}
                  onClick={() => updateSettings({ source: option.value })}
                  disabled={isFetching}
                  className={`rounded-lg border p-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                    settings.source === option.value
                      ? "border-blue-400/50 bg-blue-500/10"
                      : "border-gray-300 bg-gray-200/50 hover:bg-gray-300/50 dark:border-gray-800 dark:bg-gray-800/50 dark:hover:bg-gray-800"
                  }`}
                >
                  <div className="font-medium text-gray-900 dark:text-white">{option.label}</div>
                  <div className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                    {option.description}
                  </div>
                </button>
              ))}
            </div>

            {settings.source === "coordinates" && (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label htmlFor="speedtest-latitude" className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Latitude
                  </label>
                  <Input
                    id="speedtest-latitude"
                    type="number"
                    min={-90}
                    max={90}
                    step="any"
                    value={settings.latitude ?? ""}
                    disabled={isFetching}
                    onChange={(event) =>
                      updateSettings({
                        latitude: event.target.value === "" ? undefined : Number(event.target.value),
                      })
                    }
                    placeholder="-27.4698"
                  />
                </div>
                <div>
                  <label htmlFor="speedtest-longitude" className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Longitude
                  </label>
                  <Input
                    id="speedtest-longitude"
                    type="number"
                    min={-180}
                    max={180}
                    step="any"
                    value={settings.longitude ?? ""}
                    disabled={isFetching}
                    onChange={(event) =>
                      updateSettings({
                        longitude: event.target.value === "" ? undefined : Number(event.target.value),
                      })
                    }
                    placeholder="153.0251"
                  />
                </div>
                {!coordinatesValid && (
                  <p className="text-sm text-red-600 dark:text-red-400 sm:col-span-2">
                    Enter a latitude from -90 to 90 and longitude from -180 to 180.
                  </p>
                )}
              </div>
            )}

            {settings.source === "global" && (
              <p className="rounded-lg bg-blue-50 p-3 text-sm text-blue-700 dark:bg-blue-950/30 dark:text-blue-400">
                The first worldwide fetch can take longer. Newly discovered servers are retained indefinitely, including across Netronome restarts.
              </p>
            )}

            <div
              className="rounded-lg border border-gray-300 bg-gray-100/60 p-3 dark:border-gray-800 dark:bg-gray-900/40"
              aria-busy={isFetching}
              aria-live="polite"
            >
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p className="font-medium text-gray-900 dark:text-white">
                    {isFetching
                      ? `Fetching ${sourceLabel.toLowerCase()} servers…`
                      : !coordinatesValid
                        ? "Enter valid coordinates to check this source"
                        : formatSpeedtestServerStorageStatus(sourceLabel, {
                            stored: catalogueStatus?.stored,
                            isLoading: isStatusLoading,
                            isError: isStatusError,
                          })}
                  </p>
                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                    {isFetching
                      ? settings.source === "global"
                        ? "Contacting server regions worldwide. This can take a minute."
                        : "Requesting a fresh catalogue from Speedtest.net."
                      : !coordinatesValid
                        ? "Coordinates identify the exact source whose discovery completion status will be checked."
                        : isStatusError
                        ? "Discovery status is unavailable. You can still fetch this source."
                        : sourceStored
                          ? `${lastUpdated ? `Last updated ${lastUpdated}. ` : ""}${fetchedServers ? `${fetchedServers.servers.length} servers are retained in total. ` : ""}Fetch again to add newly available servers.`
                          : "Fetch this source once to add its servers to the retained catalogue."}
                  </p>
                </div>
                <Button
                  variant="secondary"
                  onClick={refreshServers}
                  disabled={!coordinatesValid}
                  isLoading={isFetching}
                >
                  {!isFetching && <ArrowPathIcon className="h-4 w-4" />}
                  {sourceStored ? "Fetch Again" : "Fetch Servers"}
                </Button>
              </div>

              {isFetching && (
                <div
                  className="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-300 dark:bg-gray-700"
                  role="progressbar"
                  aria-label="Fetching Speedtest.net servers"
                >
                  <div className="h-full w-1/2 animate-pulse rounded-full bg-blue-500" />
                </div>
              )}

              {isFetchError && !isFetching && (
                <p className="mt-3 text-sm text-red-600 dark:text-red-400" role="alert">
                  {fetchError instanceof Error ? fetchError.message : "Failed to fetch servers"}
                </p>
              )}

              {refreshResult && refreshResult.warnings.length > 0 && !isFetching && (
                <p className="mt-3 text-sm text-amber-600 dark:text-amber-400" role="status">
                  Some locations could not be refreshed: {refreshResult.warnings.join("; ")}
                </p>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MapPinIcon className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
              History Labels
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between gap-4">
              <div>
                <label htmlFor="show-speedtest-city" className="font-medium text-gray-900 dark:text-white">
                  Show server city
                </label>
                <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                  Append the recorded city to Speedtest.net provider names in history.
                </p>
              </div>
              <Switch
                id="show-speedtest-city"
                checked={settings.showServerCity}
                onCheckedChange={(checked) => updateSettings({ showServerCity: checked })}
              />
            </div>
            <div className="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-400">
              Preview: {formatSpeedtestServerName("Example ISP", "Brisbane", settings.showServerCity)}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
