/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { ClockIcon, GlobeAltIcon, CheckIcon } from "@heroicons/react/24/outline";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { 
  getTimeFormatSettings, 
  saveTimeFormatSettings, 
  TIMEZONE_OPTIONS,
  getBrowserTimezone,
  getTimezoneDisplayName,
  getTimeFormatExample,
  formatDateTimeWithSettings,
  type TimeFormatSettings as TimeSettings
} from "@/utils/timeSettings";
import { showToast } from "@/components/common/Toast";

export const TimeFormatSettings: React.FC = () => {
  const { t } = useTranslation();
  const [settings, setSettings] = useState<TimeSettings>(getTimeFormatSettings);
  const [hasChanges, setHasChanges] = useState(false);

  const updateSetting = (key: keyof TimeSettings, value: string | boolean) => {
    const newSettings = { ...settings, [key]: value };
    setSettings(newSettings);
    setHasChanges(true);
  };

  const saveSettings = () => {
    saveTimeFormatSettings(settings);
    setHasChanges(false);
    showToast(t('settings.saved'), "success", {
      description: t('timeSettings.configure')
    });
  };

  const resetToDefaults = () => {
    const defaultSettings: TimeSettings = {
      timezone: "auto",
      use24HourFormat: false,
    };
    setSettings(defaultSettings);
    setHasChanges(true);
  };

  const currentTimezone = settings.timezone === "auto" ? getBrowserTimezone() : settings.timezone;
  const timezoneDisplayName = getTimezoneDisplayName(settings.timezone);
  const timeExample = getTimeFormatExample(settings);
  const now = new Date();
  const dateTimeExample = formatDateTimeWithSettings(now, settings);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h3 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
            <ClockIcon className="w-6 h-6 text-gray-600 dark:text-gray-400" />
            {t('timeSettings.title')}
          </h3>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            {t('timeSettings.configure')}
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
                setSettings(getTimeFormatSettings());
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
        {/* Timezone Settings */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GlobeAltIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              {t('timeSettings.timezone')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('timeSettings.selectTimezone')}
              </label>
              <Select
                value={settings.timezone}
                onValueChange={(value) => updateSetting("timezone", value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue>
                    {timezoneDisplayName}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent className="max-h-[300px] overflow-y-auto">
                  <SelectItem value="auto">
                    <div className="flex items-center justify-between w-full">
                      <span>Auto (Browser Timezone)</span>
                      <Badge variant="secondary" className="ml-2">Recommended</Badge>
                    </div>
                  </SelectItem>
                  
                  <div className="border-t border-gray-200 dark:border-gray-700 my-2"></div>
                  
                  {TIMEZONE_OPTIONS.map((tz) => (
                    <SelectItem key={tz.value} value={tz.value}>
                      <div className="flex items-center justify-between w-full">
                        <span>{tz.label}</span>
                        <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                          {tz.offset}
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="p-3 bg-blue-50 dark:bg-blue-950/30 rounded-lg">
              <div className="flex items-start gap-2">
                <GlobeAltIcon className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                <div className="text-sm">
                  <div className="font-medium text-blue-900 dark:text-blue-300">
                    Current Timezone: {currentTimezone}
                  </div>
                  <div className="text-blue-700 dark:text-blue-400 mt-1">
                    All dates and times will be displayed in this timezone.
                    {settings.timezone === "auto" && (
                      <span className="block mt-1 text-xs">
                        Using browser detection. This will adapt automatically to your system timezone changes.
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Time Format Settings */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ClockIcon className="w-5 h-5 text-green-600 dark:text-green-400" />
              {t('settings.timeFormat')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {t('timeSettings.use24HourFormat')}
                </label>
                <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                  {t('timeSettings.use24HourFormatDesc')}
                </p>
              </div>
              <Switch
                checked={settings.use24HourFormat}
                onCheckedChange={(checked) => updateSetting("use24HourFormat", checked)}
              />
            </div>

            <div className="p-3 bg-green-50 dark:bg-green-950/30 rounded-lg">
              <div className="flex items-start gap-2">
                <ClockIcon className="w-4 h-4 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                <div className="text-sm">
                  <div className="font-medium text-green-900 dark:text-green-300">
                    {t('timeSettings.preview')}
                  </div>
                  <div className="text-green-700 dark:text-green-400 mt-1 space-y-1">
                    <div>{t('timeSettings.time')} <span className="font-mono">{timeExample}</span></div>
                    <div>{t('timeSettings.full')} <span className="font-mono">{dateTimeExample}</span></div>
                  </div>
                </div>
              </div>
            </div>

            <div className="text-xs text-gray-600 dark:text-gray-400">
              <strong>{t('timeSettings.note')}</strong> {t('timeSettings.noteMessage')}
              <ul className="list-disc list-inside mt-1 space-y-1">
                <li>{t('timeSettings.speedTestHistory')}</li>
                <li>{t('timeSettings.scheduleManager')}</li>
                <li>{t('timeSettings.monitorBandwidth')}</li>
                <li>{t('timeSettings.notificationTimestamps')}</li>
                <li>{t('timeSettings.otherDisplays')}</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Actions */}
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
