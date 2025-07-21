/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from "@/utils/baseUrl";

// Types
export interface NotificationChannel {
  id: number;
  name: string;
  url: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface NotificationEvent {
  id: number;
  category: string;
  event_type: string;
  name: string;
  description?: string;
  default_enabled: boolean;
  supports_threshold: boolean;
  threshold_unit?: string;
  created_at: string;
}

export interface NotificationRule {
  id: number;
  channel_id: number;
  event_id: number;
  enabled: boolean;
  threshold_value?: number;
  threshold_operator?: "gt" | "lt" | "eq" | "gte" | "lte";
  created_at: string;
  updated_at: string;
  channel?: NotificationChannel;
  event?: NotificationEvent;
}

export interface NotificationChannelInput {
  name: string;
  url: string;
  enabled?: boolean;
}

export interface NotificationRuleInput {
  channel_id: number;
  event_id: number;
  enabled?: boolean;
  threshold_value?: number;
  threshold_operator?: "gt" | "lt" | "eq" | "gte" | "lte";
}

export interface NotificationHistory {
  id: number;
  channel_id: number;
  event_id: number;
  success: boolean;
  error_message?: string;
  payload?: string;
  created_at: string;
}

// API methods
export const notificationsApi = {
  // Channels
  getChannels: async (): Promise<NotificationChannel[]> => {
    const response = await fetch(getApiUrl("/notifications/channels"), {
      credentials: "include",
    });
    if (!response.ok) throw new Error("Failed to fetch channels");
    return response.json();
  },

  createChannel: async (input: NotificationChannelInput): Promise<NotificationChannel> => {
    const response = await fetch(getApiUrl("/notifications/channels"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(input),
    });
    if (!response.ok) throw new Error("Failed to create channel");
    return response.json();
  },

  updateChannel: async (id: number, input: NotificationChannelInput): Promise<NotificationChannel> => {
    const response = await fetch(getApiUrl(`/notifications/channels/${id}`), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(input),
    });
    if (!response.ok) throw new Error("Failed to update channel");
    return response.json();
  },

  deleteChannel: async (id: number): Promise<void> => {
    const response = await fetch(getApiUrl(`/notifications/channels/${id}`), {
      method: "DELETE",
      credentials: "include",
    });
    if (!response.ok) throw new Error("Failed to delete channel");
  },

  // Events
  getEvents: async (category?: string): Promise<NotificationEvent[]> => {
    const url = category 
      ? getApiUrl(`/notifications/events?category=${encodeURIComponent(category)}`)
      : getApiUrl("/notifications/events");
    const response = await fetch(url, {
      credentials: "include",
    });
    if (!response.ok) throw new Error("Failed to fetch events");
    return response.json();
  },

  // Rules
  getRules: async (channelId?: number): Promise<NotificationRule[]> => {
    const url = channelId
      ? getApiUrl(`/notifications/rules?channel_id=${channelId}`)
      : getApiUrl("/notifications/rules");
    const response = await fetch(url, {
      credentials: "include",
    });
    if (!response.ok) throw new Error("Failed to fetch rules");
    return response.json();
  },

  createRule: async (input: NotificationRuleInput): Promise<NotificationRule> => {
    const response = await fetch(getApiUrl("/notifications/rules"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(input),
    });
    if (!response.ok) throw new Error("Failed to create rule");
    return response.json();
  },

  updateRule: async (id: number, input: Partial<NotificationRuleInput>): Promise<NotificationRule> => {
    const response = await fetch(getApiUrl(`/notifications/rules/${id}`), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(input),
    });
    if (!response.ok) throw new Error("Failed to update rule");
    return response.json();
  },

  deleteRule: async (id: number): Promise<void> => {
    const response = await fetch(getApiUrl(`/notifications/rules/${id}`), {
      method: "DELETE",
      credentials: "include",
    });
    if (!response.ok) throw new Error("Failed to delete rule");
  },

  // Test
  testChannel: async (channelId: number): Promise<{ success: boolean; message: string }> => {
    const response = await fetch(getApiUrl("/notifications/test"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ channel_id: channelId }),
    });
    if (!response.ok) throw new Error("Failed to send test notification");
    return response.json();
  },

  testUrl: async (url: string): Promise<{ success: boolean; message: string }> => {
    const response = await fetch(getApiUrl("/notifications/test"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ url }),
    });
    if (!response.ok) throw new Error("Failed to send test notification");
    return response.json();
  },

  // History
  getHistory: async (limit?: number): Promise<NotificationHistory[]> => {
    const url = limit
      ? getApiUrl(`/notifications/history?limit=${limit}`)
      : getApiUrl("/notifications/history");
    const response = await fetch(url, {
      credentials: "include",
    });
    if (!response.ok) throw new Error("Failed to fetch history");
    return response.json();
  },
};

// Helper functions
export const SHOUTRRR_SERVICES = [
  { value: "discord", label: "Discord", example: "discord://TOKEN@ID" },
  { value: "gotify", label: "Gotify", example: "gotify://HOSTNAME/TOKEN" },
  { value: "email", label: "Email", example: "smtp://USERNAME:PASSWORD@HOST:PORT/?from=FROM&to=TO" },
  { value: "googlechat", label: "Google Chat", example: "googlechat://SPACE/KEY/TOKEN" },
  { value: "ifttt", label: "IFTTT", example: "ifttt://KEY/?event=EVENT" },
  { value: "join", label: "Join", example: "join://APIKEY@DEVICE/?icon=URL&title=TITLE" },
  { value: "mattermost", label: "Mattermost", example: "mattermost://HOSTNAME/TOKEN" },
  { value: "matrix", label: "Matrix", example: "matrix://USERNAME:PASSWORD@HOSTNAME:PORT/ROOM" },
  { value: "ntfy", label: "ntfy", example: "ntfy://TOPIC@HOSTNAME" },
  { value: "opsgenie", label: "OpsGenie", example: "opsgenie://APIKEY" },
  { value: "pushbullet", label: "Pushbullet", example: "pushbullet://APIKEY" },
  { value: "pushover", label: "Pushover", example: "pushover://TOKEN@USER" },
  { value: "rocketchat", label: "Rocket.Chat", example: "rocketchat://HOSTNAME/TOKEN@CHANNEL" },
  { value: "slack", label: "Slack", example: "slack://TOKEN@CHANNEL" },
  { value: "teams", label: "Microsoft Teams", example: "teams://WEBHOOK_URL" },
  { value: "telegram", label: "Telegram", example: "telegram://TOKEN@CHAT_ID" },
  { value: "zulip", label: "Zulip", example: "zulip://BOTMAIL:BOTKEY@DOMAIN" },
];

export const getThresholdOperatorLabel = (operator: string): string => {
  switch (operator) {
    case "gt":
      return "Greater than";
    case "lt":
      return "Less than";
    case "eq":
      return "Equal to";
    case "gte":
      return "Greater than or equal";
    case "lte":
      return "Less than or equal";
    default:
      return operator;
  }
};

export const getEventCategoryIcon = (category: string): string => {
  switch (category) {
    case "speedtest":
      return "📊";
    case "packetloss":
      return "📉";
    case "agent":
      return "🖥️";
    default:
      return "📌";
  }
};