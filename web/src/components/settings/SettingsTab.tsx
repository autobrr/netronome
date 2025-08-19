/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RefreshCw, MapPin, Globe, Trash2 } from "lucide-react";
import { motion } from "framer-motion";

interface SettingsTabProps {
  onRefreshComprehensiveServers: () => void;
  isLoadingComprehensive: boolean;
  useComprehensiveServers: boolean;
  onAddLocationServers: (location: string) => Promise<void>;
  onClearServerCache: () => void;
}

export function SettingsTab({
  onRefreshComprehensiveServers,
  isLoadingComprehensive,
  useComprehensiveServers,
  onAddLocationServers,
  onClearServerCache,
}: SettingsTabProps) {
  const [customLocation, setCustomLocation] = useState("");
  const [isAddingLocation, setIsAddingLocation] = useState(false);
  const [locationError, setLocationError] = useState("");

  const handleAddLocation = async () => {
    if (!customLocation.trim()) {
      setLocationError("Please enter a location in lat,lon format (e.g., 37.7749,-122.4194)");
      return;
    }

    // Validate lat,lon format
    const locationRegex = /^-?\d+\.?\d*,-?\d+\.?\d*$/;
    if (!locationRegex.test(customLocation.trim())) {
      setLocationError("Invalid format. Use: latitude,longitude (e.g., 37.7749,-122.4194)");
      return;
    }

    setLocationError("");
    setIsAddingLocation(true);

    try {
      await onAddLocationServers(customLocation.trim());
      setCustomLocation("");
    } catch (error) {
      setLocationError(`Failed to add servers for location: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setIsAddingLocation(false);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="space-y-6"
    >
      <div>
        <h2 className="text-2xl font-bold tracking-tight mb-2">Settings</h2>
        <p className="text-muted-foreground">
          Configure your speedtest preferences and manage server data.
        </p>
      </div>

      {/* Speedtest Server Management */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Speedtest Server Management
          </CardTitle>
          <CardDescription>
            Manage your comprehensive speedtest server list and add custom locations.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Server Cache Status */}
          <div className="flex items-center justify-between p-4 bg-muted/50 rounded-lg">
            <div>
              <p className="font-medium">Comprehensive Server Cache</p>
              <p className="text-sm text-muted-foreground">
                Status: {useComprehensiveServers ? "Active" : "Using local servers only"}
              </p>
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={onRefreshComprehensiveServers}
                disabled={isLoadingComprehensive}
                className="flex items-center gap-2"
              >
                <RefreshCw className={`h-4 w-4 ${isLoadingComprehensive ? "animate-spin" : ""}`} />
                {isLoadingComprehensive ? "Refreshing..." : "Refresh Global Servers"}
              </Button>
              <Button
                variant="outline"
                onClick={onClearServerCache}
                disabled={isLoadingComprehensive}
                className="flex items-center gap-2"
              >
                <Trash2 className="h-4 w-4" />
                Clear Cache
              </Button>
            </div>
          </div>

          <div className="border-t my-6" />

          {/* Add Custom Location */}
          <div className="space-y-4">
            <div>
              <h4 className="font-medium flex items-center gap-2 mb-2">
                <MapPin className="h-4 w-4" />
                Add Servers by Location
              </h4>
              <p className="text-sm text-muted-foreground mb-4">
                Add speedtest servers from a specific geographic location using coordinates.
              </p>
            </div>

            <div className="space-y-3">
              <div>
                <Label htmlFor="location">Location (latitude,longitude)</Label>
                <Input
                  id="location"
                  placeholder="e.g., 37.7749,-122.4194 (San Francisco)"
                  value={customLocation}
                  onChange={(e) => {
                    setCustomLocation(e.target.value);
                    setLocationError("");
                  }}
                  disabled={isAddingLocation}
                />
                {locationError && (
                  <p className="text-sm text-destructive mt-1">{locationError}</p>
                )}
              </div>

              <Button
                onClick={handleAddLocation}
                disabled={isAddingLocation || !customLocation.trim()}
                className="w-full sm:w-auto"
              >
                <MapPin className={`h-4 w-4 mr-2 ${isAddingLocation ? "animate-pulse" : ""}`} />
                {isAddingLocation ? "Adding Servers..." : "Add Servers for Location"}
              </Button>
            </div>

            <div className="p-3 bg-blue-50 dark:bg-blue-950/20 rounded-lg border border-blue-200 dark:border-blue-800">
              <p className="text-sm text-blue-700 dark:text-blue-300">
                <strong>Tip:</strong> You can find coordinates using Google Maps. Right-click on a location 
                and select "What's here?" to get the latitude and longitude.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Additional Settings Placeholder */}
      <Card>
        <CardHeader>
          <CardTitle>Additional Settings</CardTitle>
          <CardDescription>
            More configuration options coming soon.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground text-sm">
            Future settings for notifications, test preferences, and more will appear here.
          </p>
        </CardContent>
      </Card>
    </motion.div>
  );
}
