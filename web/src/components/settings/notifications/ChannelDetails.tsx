/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { Switch, Button as HeadlessButton, Input } from "@headlessui/react";
import { 
  BellIcon, 
  TrashIcon, 
  PencilIcon, 
  CheckIcon, 
  XMarkIcon 
} from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import { SHOUTRRR_SERVICES } from "@/api/notifications";

interface ChannelDetailsProps {
  channel: any;
  onUpdate: any;
  onDelete: () => void;
  onTest: () => void;
  isDeleting: boolean;
  isTesting: boolean;
}

export const ChannelDetails: React.FC<ChannelDetailsProps> = ({
  channel,
  onUpdate,
  onDelete,
  onTest,
  isDeleting,
  isTesting,
}) => {
  const [isEditingUrl, setIsEditingUrl] = useState(false);
  const [editedUrl, setEditedUrl] = useState(channel.url);
  const [urlError, setUrlError] = useState("");

  // Detect service type from URL
  const detectServiceType = (url: string) => {
    const serviceType = url.match(/^(\w+):\/\//)?.[1];
    return SHOUTRRR_SERVICES.find(s => s.value === serviceType);
  };

  const validateUrl = (url: string): boolean => {
    if (!url.trim()) {
      setUrlError("URL is required");
      return false;
    }
    
    const serviceType = url.match(/^(\w+):\/\//)?.[1];
    if (!serviceType) {
      setUrlError("Invalid URL format. Must start with service://");
      return false;
    }

    const validService = SHOUTRRR_SERVICES.find(s => s.value === serviceType);
    if (!validService) {
      setUrlError(`Unknown service type: ${serviceType}`);
      return false;
    }

    setUrlError("");
    return true;
  };

  const handleSaveUrl = () => {
    if (!validateUrl(editedUrl)) {
      return;
    }

    onUpdate.mutate({
      id: channel.id,
      name: channel.name,
      url: editedUrl,
      enabled: channel.enabled,
    });
    setIsEditingUrl(false);
  };

  const handleCancelEdit = () => {
    setEditedUrl(channel.url);
    setUrlError("");
    setIsEditingUrl(false);
  };

  const detectedService = detectServiceType(isEditingUrl ? editedUrl : channel.url);

  return (
    <div className="bg-gray-50/95 dark:bg-gray-850/95 rounded-xl shadow-lg border border-gray-200 dark:border-gray-800 p-6">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h4 className="text-xl font-semibold text-gray-900 dark:text-white">
            {channel.name}
          </h4>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Channel configuration and settings
          </p>
        </div>

        <div className="flex items-center gap-2">
          <HeadlessButton
            onClick={onTest}
            disabled={isTesting}
            className="flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isTesting ? (
              <div className="w-4 h-4 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full animate-spin" />
            ) : (
              <BellIcon className="w-4 h-4" />
            )}
            Test
          </HeadlessButton>
          <HeadlessButton
            onClick={onDelete}
            disabled={isDeleting}
            className="flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-red-600 hover:bg-red-700 dark:bg-red-500 dark:hover:bg-red-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <TrashIcon className="w-4 h-4" />
            Delete
          </HeadlessButton>
        </div>
      </div>

      <div className="space-y-4">
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Service URL
            </label>
            {!isEditingUrl && (
              <HeadlessButton
                onClick={() => setIsEditingUrl(true)}
                className="p-1 text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
                title="Edit URL"
              >
                <PencilIcon className="w-4 h-4" />
              </HeadlessButton>
            )}
          </div>
          
          {isEditingUrl ? (
            <div className="space-y-2">
              <div className="relative">
                <Input
                  type="text"
                  value={editedUrl}
                  onChange={(e) => {
                    setEditedUrl(e.target.value);
                    setUrlError("");
                  }}
                  className={cn(
                    "w-full px-3 py-2 font-mono text-sm bg-white dark:bg-gray-900 border rounded-lg transition-colors",
                    "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent",
                    urlError 
                      ? "border-red-500 dark:border-red-400" 
                      : "border-gray-300 dark:border-gray-700"
                  )}
                  placeholder="service://..."
                />
              </div>
              
              {urlError && (
                <p className="text-xs text-red-600 dark:text-red-400">
                  {urlError}
                </p>
              )}
              
              {detectedService && !urlError && (
                <div className="p-2 bg-blue-500/10 border border-blue-500/30 rounded-lg">
                  <p className="text-xs text-blue-700 dark:text-blue-300">
                    <span className="font-medium">{detectedService.label} format:</span>{" "}
                    <code className="font-mono">{detectedService.example}</code>
                  </p>
                </div>
              )}
              
              <div className="flex items-center gap-2">
                <HeadlessButton
                  onClick={handleSaveUrl}
                  disabled={onUpdate.isPending}
                  className="flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {onUpdate.isPending ? (
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <>
                      <CheckIcon className="w-4 h-4" />
                      Save
                    </>
                  )}
                </HeadlessButton>
                <HeadlessButton
                  onClick={handleCancelEdit}
                  className="flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors"
                >
                  <XMarkIcon className="w-4 h-4" />
                  Cancel
                </HeadlessButton>
              </div>
            </div>
          ) : (
            <div className="mt-1 p-3 bg-gray-200/50 dark:bg-gray-800/50 rounded-lg group hover:bg-gray-200/70 dark:hover:bg-gray-800/70 transition-colors">
              <code className="text-sm text-gray-800 dark:text-gray-200 break-all">
                {channel.url}
              </code>
            </div>
          )}
        </div>

        <div className="flex items-center justify-between p-4 bg-gray-200/30 dark:bg-gray-800/30 rounded-lg">
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Channel Status
            </label>
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              {channel.enabled
                ? "Notifications will be sent to this channel"
                : "Channel is disabled"}
            </p>
          </div>
          <Switch
            checked={channel.enabled}
            onChange={(checked) => {
              onUpdate.mutate({
                id: channel.id,
                name: channel.name,
                url: channel.url,
                enabled: checked,
              });
            }}
            className={cn(
              "relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2",
              channel.enabled ? "bg-blue-600" : "bg-gray-200 dark:bg-gray-700"
            )}
          >
            <span className="sr-only">Enable channel</span>
            <span
              className={cn(
                "inline-block h-4 w-4 rounded-full bg-white shadow-lg transition-transform duration-200",
                channel.enabled ? "translate-x-6" : "translate-x-1"
              )}
            />
          </Switch>
        </div>
      </div>
    </div>
  );
};
