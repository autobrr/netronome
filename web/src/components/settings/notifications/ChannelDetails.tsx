/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState } from "react";
import { 
  BellIcon, 
  TrashIcon, 
  PencilIcon, 
  CheckIcon, 
  XMarkIcon 
} from "@heroicons/react/24/outline";
import { cn } from "@/lib/utils";
import { SHOUTRRR_SERVICES } from "@/api/notifications";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

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
    <Card>
      <CardHeader className="p-6">
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="text-xl">{channel.name}</CardTitle>
            <CardDescription>
              Channel configuration and settings
            </CardDescription>
          </div>

        <div className="flex items-center gap-2">
          <Button
            onClick={onTest}
            disabled={isTesting}
            variant="secondary"
            size="sm"
          >
            {isTesting ? (
              <div className="w-4 h-4 border-2 border-gray-300 dark:border-gray-700 border-t-blue-500 dark:border-t-blue-400 rounded-full animate-spin" />
            ) : (
              <BellIcon className="w-4 h-4" />
            )}
            Test
          </Button>
          <Button
            onClick={onDelete}
            disabled={isDeleting}
            variant="destructive"
            size="sm"
          >
            <TrashIcon className="w-4 h-4" />
            Delete
          </Button>
        </div>
        </div>
      </CardHeader>
      <CardContent className="p-6 pt-0">

      <div className="space-y-4">
        <div>
          <div className="flex items-center justify-between mb-1">
            <Label>
              Service URL
            </Label>
            {!isEditingUrl && (
              <Button
                onClick={() => setIsEditingUrl(true)}
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                title="Edit URL"
              >
                <PencilIcon className="w-4 h-4" />
              </Button>
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
                    "font-mono text-sm",
                    urlError && "border-red-500 dark:border-red-400"
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
                <Button
                  onClick={handleSaveUrl}
                  disabled={onUpdate.isPending}
                  size="sm"
                >
                  {onUpdate.isPending ? (
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <>
                      <CheckIcon className="w-4 h-4" />
                      Save
                    </>
                  )}
                </Button>
                <Button
                  onClick={handleCancelEdit}
                  variant="secondary"
                  size="sm"
                >
                  <XMarkIcon className="w-4 h-4" />
                  Cancel
                </Button>
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
            <Label>
              Channel Status
            </Label>
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
              {channel.enabled
                ? "Notifications will be sent to this channel"
                : "Channel is disabled"}
            </p>
          </div>
          <Switch
            checked={channel.enabled}
            onCheckedChange={(checked) => {
              onUpdate.mutate({
                id: channel.id,
                name: channel.name,
                url: channel.url,
                enabled: checked,
              });
            }}
          />
        </div>
      </div>
      </CardContent>
    </Card>
  );
};
