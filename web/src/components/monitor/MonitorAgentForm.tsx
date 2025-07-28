/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useEffect } from "react";
import { EyeIcon, EyeSlashIcon, ShieldCheckIcon } from "@heroicons/react/24/outline";
import { Button } from "@/components/ui/button";
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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const handleUrlChange = (value: string) => {
    // Clean up the URL - remove any trailing slashes and the SSE endpoint if present
    let cleanUrl = value.trim();
    if (cleanUrl.endsWith("/")) {
      cleanUrl = cleanUrl.slice(0, -1);
    }
    setFormData({ ...formData, url: cleanUrl });
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
            {agent ? (agent.isTailscale ? "Edit Monitoring Settings" : "Edit Agent") : "Add Agent"}
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
                    Tailscale Connected Agent
                  </p>
                  {agent.tailscaleHostname && (
                    <p className="text-xs text-blue-700 dark:text-blue-300 mt-0.5">
                      Hostname: {agent.tailscaleHostname}
                    </p>
                  )}
                  {agent.discoveredAt && (
                    <p className="text-xs text-blue-700 dark:text-blue-300">
                      Auto-discovered on{" "}
                      {new Date(agent.discoveredAt).toLocaleDateString()}
                    </p>
                  )}
                  {isAutoDiscoveredTailscale && (
                    <p className="text-xs text-blue-600 dark:text-blue-200 mt-1">
                      Connection details are managed by Tailscale. Only monitoring can be toggled.
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}

          <div>
            <Label htmlFor="name">
              Agent Name
            </Label>
            <Input
              type="text"
              id="name"
              value={formData.name}
              data-1p-ignore
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              placeholder="Remote Server"
              required
              disabled={isAutoDiscoveredTailscale}
            />
          </div>

          <div>
            <Label htmlFor="url">
              Agent URL
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
              Enter the base URL of the monitor agent
            </p>
          </div>

          {!isAutoDiscoveredTailscale && (
            <div>
              <Label htmlFor="apiKey">
                API Key (Optional)
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
                      ? "API key is configured"
                      : "Leave empty for no authentication"
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
                  If set, the agent will require this API key for
                  authentication
                </p>
                <Button
                  type="button"
                  onClick={generateApiKey}
                  variant="secondary"
                  size="sm"
                  className="text-xs h-auto py-0 px-1 bg-transparent hover:bg-transparent text-blue-600 hover:text-blue-500 dark:text-blue-400 border-0 shadow-none"
                >
                  Generate Random Key
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
                Enable monitoring
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
              Cancel
            </Button>
            <Button
              type="submit"
              variant="default"
              disabled={isSubmitting}
              isLoading={isSubmitting}
            >
              {agent ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
