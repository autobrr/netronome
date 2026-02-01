/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { MapPinIcon, CheckIcon } from "@heroicons/react/24/outline";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import {
  getDistanceSettings,
  saveDistanceSettings,
  formatDistance,
  type DistanceSettings as DistanceSettingsType,
  type DistanceUnit,
} from "@/utils/distanceSettings";
import { showToast } from "@/components/common/Toast";

export const DistanceSettings: React.FC = () => {
  const { t } = useTranslation();
  const [settings, setSettings] = useState<DistanceSettingsType>(getDistanceSettings);
  const [hasChanges, setHasChanges] = useState(false);

  const updateUnit = (unit: DistanceUnit) => {
    setSettings({ unit });
    setHasChanges(true);
  };

  const saveSettings = () => {
    saveDistanceSettings(settings);
    setHasChanges(false);
    showToast(t('settings.saved'), "success");
  };

  const resetToDefaults = () => {
    setSettings({ unit: "km" });
    setHasChanges(true);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
            <MapPinIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
            {t('distanceSettings.title')}
          </h3>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            {t('distanceSettings.configure')}
          </p>
        </div>

        {hasChanges && (
          <div className="flex items-center gap-2">
            <Button
              onClick={saveSettings}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              <CheckIcon className="w-4 h-4" />
              {t('common.saveChanges')}
            </Button>
            <Button
              onClick={() => {
                setSettings(getDistanceSettings());
                setHasChanges(false);
              }}
              variant="secondary"
            >
              {t('common.cancel')}
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MapPinIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              {t('distanceSettings.distanceUnit')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-3">
              <button
                onClick={() => updateUnit("km")}
                className={`flex-1 p-4 rounded-lg border transition-colors text-left ${
                  settings.unit === "km"
                    ? "bg-blue-500/10 border-blue-400/50"
                    : "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800"
                }`}
              >
                <div className="font-medium text-gray-900 dark:text-white">{t('distanceSettings.metric')}</div>
                <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('distanceSettings.kilometersUnit')}</div>
              </button>
              <button
                onClick={() => updateUnit("mi")}
                className={`flex-1 p-4 rounded-lg border transition-colors text-left ${
                  settings.unit === "mi"
                    ? "bg-blue-500/10 border-blue-400/50"
                    : "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800"
                }`}
              >
                <div className="font-medium text-gray-900 dark:text-white">{t('distanceSettings.imperial')}</div>
                <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('distanceSettings.milesUnit')}</div>
              </button>
            </div>

            <div className="p-3 bg-blue-50 dark:bg-blue-950/30 rounded-lg">
              <div className="flex items-start gap-2">
                <MapPinIcon className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                <div className="text-sm">
                  <div className="font-medium text-blue-900 dark:text-blue-300">
                    {t('distanceSettings.preview')}
                  </div>
                  <div className="text-blue-700 dark:text-blue-400 mt-1">
                    100 km = <span className="font-mono">{formatDistance(100, settings)}</span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent className="p-6">
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div>
              <h4 className="font-medium text-gray-900 dark:text-white">
                Quick Actions
              </h4>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Reset settings or apply common configurations
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                onClick={resetToDefaults}
                variant="outline"
              >
                Reset to Defaults
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
