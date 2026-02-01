/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { EyeIcon, EyeSlashIcon, ShieldCheckIcon } from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MonitorAgent, CreateAgentRequest } from "@/api/monitor";
import { formatDateWithSettings } from "@/utils/timeSettings";
import { useTranslation } from "react-i18next";

interface MonitorAgentFormProps {
  agent?: MonitorAgent | null;
  onSubmit: (data: CreateAgentRequest) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  isOpen: boolean;
}

export const MonitorAgentForm: React.FC<MonitorAgentFormProps> = ({
  agent,
  onSubmit,
  onCancel,
  isSubmitting,
  isOpen,
}) => {
  const { t } = useTranslation();
  const [formData, setFormData] = useState<CreateAgentRequest>({
    name: "",
    url: "http://",
    apiKey: "",
    enabled: true,
  });
  const [showApiKey, setShowApiKey] = useState(false);
  
  // Check if this is a Tailscale auto-discovered agent
  const isAutoDiscoveredTailscale = !!(agent?.discoveredAt && agent?.isTailscale);

  useEffect(() => {
    if (agent) {
      setFormData({
        name: agent.name,
        url: agent.url.replace(/\/events\?stream=live-data$/, ""),
        apiKey: agent.apiKey || "",
        enabled: agent.enabled,
      });
    } else if (isOpen) {
      // Reset form for new agent
      setFormData({
        name: "",
        url: "http://",
        apiKey: "",
        enabled: true,
      });
    }
    setShowApiKey(false); // Reset visibility when dialog opens
  }, [agent, isOpen]);

  const sanitizeUrl = (value: string) => {
    // Trim whitespace and drop the SSE streaming suffix if the agent URL was copied from the events endpoint
    let cleanUrl = value.trim();
    cleanUrl = cleanUrl.replace(/\/events\?stream=live-data$/, "");
    return cleanUrl;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const sanitizedData: CreateAgentRequest = {
      ...formData,
      url: sanitizeUrl(formData.url),
    };
    setFormData(sanitizedData);
    onSubmit(sanitizedData);
  };

  const handleUrlChange = (value: string) => {
    setFormData({ ...formData, url: value });
  };

  const generateApiKey = () => {
    // Generate a random API key
    const chars =
      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    let key = "";
    for (let i = 0; i < 32; i++) {
      key += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setFormData({ ...formData, apiKey: key });
    setShowApiKey(true); // Show the generated key
  };

  return (
    <Dialog open={isOpen} onOpenChange={onCancel}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {agent ? (agent.isTailscale ? t('monitoring.editMonitoringSettings') : t('monitoring.editAgent')) : t('monitoring.addAgent')}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Show Tailscale info if this is a Tailscale agent */}
          {agent?.isTailscale && (
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
              <div className="flex items-center gap-2">
                <ShieldCheckIcon className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                <div className="flex-1">
                  <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                    {t('monitoring.tailscaleConnectedAgent')}
                  </p>
                  {agent.tailscaleHostname && (
                    <p className="text-xs text-blue-700 dark:text-blue-300 mt-0.5">
                      {t('monitoring.hostname')}: {agent.tailscaleHostname}
                    </p>
                  )}
                  {agent.discoveredAt && (
                    <p className="text-xs text-blue-700 dark:text-blue-300">
                      {t('monitoring.autoDiscoveredOn', { 
                        date: formatDateWithSettings(agent.discoveredAt, { 
                          year: "numeric", 
                          month: "long", 
                          day: "numeric" 
                        })
                      })}
                    </p>
                  )}
                  {isAutoDiscoveredTailscale && (
                    <p className="text-xs text-blue-600 dark:text-blue-200 mt-1">
                      {t('monitoring.tailscaleManagedDesc')}
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}

          <div>
            <Label htmlFor="name">
              {t('monitoring.agentName')}
            </Label>
            <Input
              type="text"
              id="name"
              value={formData.name}
              data-1p-ignore
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              placeholder={t('monitoring.remoteServerPlaceholder')}
              required
              disabled={isAutoDiscoveredTailscale}
            />
          </div>

          <div>
            <Label htmlFor="url">
              {t('monitoring.agentUrl')}
            </Label>
            <Input
              type="url"
              id="url"
              value={formData.url}
              onChange={(e) => handleUrlChange(e.target.value)}
              placeholder="http://192.168.1.100:8200"
              required
              disabled={isAutoDiscoveredTailscale}
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {t('monitoring.agentUrlDesc')}
            </p>
          </div>

          {!isAutoDiscoveredTailscale && (
            <div>
              <Label htmlFor="apiKey">
                {t('monitoring.apiKeyOptional')}
              </Label>
              <div className="relative">
                <Input
                  type={showApiKey ? "text" : "password"}
                  id="apiKey"
                  value={formData.apiKey || ""}
                  data-1p-ignore
                  onChange={(e) =>
                    setFormData({ ...formData, apiKey: e.target.value })
                  }
                  className="pr-10"
                  placeholder={
                    agent?.apiKey === "configured"
                      ? t('monitoring.apiKeyConfigured')
                      : t('monitoring.apiKeyPlaceholder')
                  }
                />
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  className="absolute inset-y-0 right-0 h-full px-3 rounded-l-none border-0 bg-transparent hover:bg-transparent"
                  onClick={() => setShowApiKey(!showApiKey)}
                >
                  {showApiKey ? (
                    <EyeSlashIcon className="h-5 w-5 text-gray-400" />
                  ) : (
                    <EyeIcon className="h-5 w-5 text-gray-400" />
                  )}
                </Button>
              </div>
              <div className="mt-1 flex items-center justify-between">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {t('monitoring.apiKeyDesc')}
                </p>
                <Button
                  type="button"
                  onClick={generateApiKey}
                  variant="secondary"
                  size="sm"
                  className="text-xs h-auto py-0 px-1 bg-transparent hover:bg-transparent text-blue-600 hover:text-blue-500 dark:text-blue-400 border-0 shadow-none"
                >
                  {t('monitoring.generateRandomKey')}
                </Button>
              </div>
            </div>
          )}

          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) =>
                  setFormData({
                    ...formData,
                    enabled: checked as boolean,
                  })
                }
              />
              <Label htmlFor="enabled" className="cursor-pointer">
                {t('monitoring.enableMonitoring')}
              </Label>
            </div>
          </div>

          {/* Actions */}
          <DialogFooter className="mt-6">
            <Button
              type="button"
              variant="secondary"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="submit"
              variant="default"
              disabled={isSubmitting}
              isLoading={isSubmitting}
            >
              {agent ? t('common.update') : t('common.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
