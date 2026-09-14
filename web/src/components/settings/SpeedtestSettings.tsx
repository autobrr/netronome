/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import {
  CheckIcon,
  GlobeAltIcon,
  MapPinIcon,
  ServerStackIcon,
} from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { showToast } from "@/components/common/Toast";
import {
  formatSpeedtestServerName,
  getSpeedtestSettings,
  saveSpeedtestSettings,
  type SpeedtestServerSource,
  type SpeedtestSettingsDraft,
} from "@/utils/speedtestSettings";

const SOURCE_OPTIONS: Array<{
  value: SpeedtestServerSource;
  label: string;
  description: string;
}> = [
  { value: "local", label: "Local", description: "Use servers near your detected location." },
  { value: "global", label: "Global", description: "Build a worldwide catalogue from known regions." },
  { value: "coordinates", label: "Coordinates", description: "Find servers near a latitude and longitude." },
];

/** Configures Speedtest.net server discovery and historical city labels. */
export const SpeedtestSettings = () => {
  const [settings, setSettings] = useState<SpeedtestSettingsDraft>(getSpeedtestSettings);
  const [hasChanges, setHasChanges] = useState(false);

  const updateSettings = (change: Partial<SpeedtestSettingsDraft>) => {
    setSettings((current) => ({ ...current, ...change }));
    setHasChanges(true);
  };

  const coordinatesValid =
    settings.source !== "coordinates" ||
    (typeof settings.latitude === "number" &&
      Number.isFinite(settings.latitude) &&
      settings.latitude >= -90 &&
      settings.latitude <= 90 &&
      typeof settings.longitude === "number" &&
      Number.isFinite(settings.longitude) &&
      settings.longitude >= -180 &&
      settings.longitude <= 180);

  const saveSettings = () => {
    if (!saveSpeedtestSettings(settings)) {
      showToast("Failed to save speedtest settings", "error");
      return;
    }
    setSettings(getSpeedtestSettings());
    setHasChanges(false);
    showToast("Speedtest settings saved", "success", {
      description: "Server lists and history labels have been refreshed",
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="flex items-center gap-2 text-xl font-bold text-gray-900 dark:text-white sm:text-2xl">
            <ServerStackIcon className="h-6 w-6 text-gray-600 dark:text-gray-400" />
            Speedtest.net Settings
          </h3>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Choose where servers are discovered and how saved results are labelled.
          </p>
        </div>
        {hasChanges && (
          <div className="flex items-center gap-2">
            <Button onClick={saveSettings} disabled={!coordinatesValid}>
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
                  className={`rounded-lg border p-4 text-left transition-colors ${
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
                The first worldwide fetch can take longer. Results are cached by the server for 30 minutes.
              </p>
            )}
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
