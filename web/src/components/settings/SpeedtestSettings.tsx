/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RefreshCw, MapPin, Globe, Trash2, ChevronDown, ChevronRight } from "lucide-react";
import { Server, ComprehensiveServerData } from "@/types/types";
import { getApiUrl } from "@/utils/baseUrl";
import { showToast } from "@/components/common/Toast";
import { useQueryClient } from "@tanstack/react-query";

export function SpeedtestSettings() {
  const [customLocation, setCustomLocation] = useState("");
  const [isAddingLocation, setIsAddingLocation] = useState(false);
  const [locationError, setLocationError] = useState("");
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [expandedLocations, setExpandedLocations] = useState<Set<string>>(new Set());

  const queryClient = useQueryClient();
  const COMPREHENSIVE_SERVERS_CACHE_KEY = "netronome-comprehensive-servers";
  const LOCATION_SERVERS_CACHE_KEY = "netronome-location-servers";

  interface LocationServerData {
    locations: Record<string, Server[]>;
    totalServers: number;
    lastUpdated: string;
  }

  // Function to get cached comprehensive server data
  const getCachedComprehensiveData = (): ComprehensiveServerData | null => {
    try {
      const cached = localStorage.getItem(COMPREHENSIVE_SERVERS_CACHE_KEY);
      if (!cached) return null;
      
      const parsed = JSON.parse(cached);
      return parsed.data;
    } catch (error) {
      console.error("Error reading comprehensive servers cache:", error);
      localStorage.removeItem(COMPREHENSIVE_SERVERS_CACHE_KEY);
      return null;
    }
  };

  // Functions for location-based server caching
  const getCachedLocationData = (): LocationServerData | null => {
    try {
      const cached = localStorage.getItem(LOCATION_SERVERS_CACHE_KEY);
      if (!cached) return null;
      return JSON.parse(cached);
    } catch (error) {
      console.error("Error reading location server cache:", error);
      return null;
    }
  };

  const setCachedLocationData = (data: LocationServerData) => {
    try {
      localStorage.setItem(LOCATION_SERVERS_CACHE_KEY, JSON.stringify(data));
    } catch (error) {
      console.error("Error caching location servers:", error);
    }
  };

  const cachedData = getCachedComprehensiveData();

  const handleAddLocation = async (force: boolean = false) => {
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

    console.log('🌍 Adding servers for location:', customLocation.trim(), force ? '(FORCE)' : '(normal)');

    try {
      // If force mode, clear existing location cache first  
      if (force) {
        console.log('🧹 Force mode: clearing existing location cache');
        localStorage.removeItem(LOCATION_SERVERS_CACHE_KEY);
      }

      const response = await fetch(getApiUrl(`/servers?testType=speedtest&location=${encodeURIComponent(customLocation.trim())}`));
      if (!response.ok) {
        throw new Error('Failed to fetch servers for location');
      }
      
      const locationServers = await response.json();
      console.log('Received servers from API:', {
        count: locationServers.length,
        sampleServer: locationServers[0]?.name || 'No servers',
        location: customLocation.trim(),
        timestamp: new Date().toISOString(),
        serverDetails: locationServers.slice(0, 5).map((s: any) => ({
          id: s.id,
          name: s.name,
          distance: s.distance?.toFixed(2) + 'km'
        }))
      });
      
      if (locationServers.length === 0) {
        throw new Error('No servers found for this location');
      }

      // Always use location cache mode for coordinate-based queries
      console.log('📍 Location cache mode: Using separate location-based cache');
      
      const existingLocationData = getCachedLocationData() || { 
        locations: {}, 
        totalServers: 0, 
        lastUpdated: new Date().toISOString() 
      };
      
      console.log('📦 Existing location data:', {
        exists: !!existingLocationData,
        totalServers: existingLocationData.totalServers,
        locations: Object.keys(existingLocationData.locations),
        lastUpdated: existingLocationData.lastUpdated
      });

      // Add servers to location-based cache
      const updatedLocationData: LocationServerData = {
        ...existingLocationData,
        locations: {
          ...existingLocationData.locations,
          [customLocation.trim()]: locationServers
        },
        totalServers: Object.values({
          ...existingLocationData.locations,
          [customLocation.trim()]: locationServers
        }).reduce((total, servers) => total + servers.length, 0),
        lastUpdated: new Date().toISOString()
      };

      setCachedLocationData(updatedLocationData);
      console.log('✅ Updated location cache:', {
        totalServers: updatedLocationData.totalServers,
        locations: Object.keys(updatedLocationData.locations),
        newLocationAdded: customLocation.trim()
      });

      showToast(`Added ${locationServers.length} servers for ${customLocation.trim()} to location cache!`, "success");
      
      setCustomLocation("");
      // Force re-render by triggering React Query invalidation
      queryClient.invalidateQueries({ queryKey: ["servers", "comprehensive", "speedtest"] });
      
    } catch (error) {
      console.error('Error adding location servers:', error);
      setLocationError(`Failed to add servers for location: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setIsAddingLocation(false);
    }
  };

  const handleRefreshServers = async () => {
    setIsRefreshing(true);
    try {
      localStorage.removeItem(COMPREHENSIVE_SERVERS_CACHE_KEY);
      queryClient.invalidateQueries({ queryKey: ["servers", "comprehensive", "speedtest"] });
      showToast("Server cache cleared and refreshed", "success");
    } catch (error) {
      showToast("Failed to refresh servers", "error");
    } finally {
      setIsRefreshing(false);
    }
  };

  const handleClearCache = () => {
    localStorage.removeItem(COMPREHENSIVE_SERVERS_CACHE_KEY);
    localStorage.removeItem(LOCATION_SERVERS_CACHE_KEY);
    queryClient.invalidateQueries({ queryKey: ["servers", "comprehensive", "speedtest"] });
    showToast("Both comprehensive and location caches cleared! App will use default local servers.", "info");
  };

  const toggleLocationExpansion = (location: string) => {
    const newExpanded = new Set(expandedLocations);
    if (newExpanded.has(location)) {
      newExpanded.delete(location);
    } else {
      newExpanded.add(location);
    }
    setExpandedLocations(newExpanded);
  };

  const formatServerInfo = (server: Server) => {
    const parts = [];
    if (server.name) parts.push(server.name);
    if (server.host) parts.push(`Host: ${server.host}`);
    if (server.country) parts.push(`Country: ${server.country}`);
    if (server.distance !== undefined) parts.push(`Distance: ${server.distance.toFixed(1)}km`);
    return parts.join(" • ");
  };

  return (
    <div className="space-y-6">
      {/* Debug Information */}
      {process.env.NODE_ENV === 'development' && (
        <Card className="border-yellow-200 dark:border-yellow-800">
          <CardHeader>
            <CardTitle className="text-sm text-yellow-700 dark:text-yellow-300 flex items-center gap-2">
              🐛 Debug Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xs font-mono space-y-2 text-card-foreground">
              <div className="mb-3 p-2 bg-blue-50 dark:bg-blue-900/20 rounded border border-blue-200 dark:border-blue-800">
                <p className="font-semibold text-blue-800 dark:text-blue-300 mb-1">ℹ️ How it works:</p>
                <p className="text-blue-700 dark:text-blue-400 text-xs">
                  When you add servers by location coordinates, they're stored in a separate location-based cache. 
                  The app uses servers from both the comprehensive cache and location cache for speedtest selection. 
                  Location-based servers are fetched using precise coordinates for optimal geographic accuracy.
                </p>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="font-semibold">Cache Info:</p>
                  <p>Cache Key: <span className="text-blue-600 dark:text-blue-400">{COMPREHENSIVE_SERVERS_CACHE_KEY}</span></p>
                  <p>Has Cached Data: <span className="text-green-600 dark:text-green-400">{cachedData ? 'Yes' : 'No'}</span></p>
                  <p>Total Servers: <span className="text-purple-600 dark:text-purple-400">{cachedData?.totalServers || 0}</span></p>
                  <p>Locations: <span className="text-orange-600 dark:text-orange-400">{cachedData?.locations?.length || 0}</span></p>
                  <p>Last Updated: <span className="text-cyan-600 dark:text-cyan-400">
                    {cachedData?.lastUpdated ? new Date(cachedData.lastUpdated).toLocaleString() : 'Never'}
                  </span></p>
                </div>
                <div>
                  <p className="font-semibold">Storage Keys:</p>
                  <div className="max-h-20 overflow-y-auto">
                    {Object.keys(localStorage).filter(k => k.includes('server')).map(key => (
                      <p key={key} className="text-xs text-muted-foreground truncate">{key}</p>
                    ))}
                  </div>
                </div>
              </div>
              
              {cachedData && (
                <details className="mt-3">
                  <summary className="cursor-pointer text-blue-600 dark:text-blue-400 hover:underline">
                    📁 View Cache Structure ({Object.keys(cachedData.servers || {}).length} locations)
                  </summary>
                  <div className="mt-2 p-3 bg-muted/20 dark:bg-muted/30 rounded border">
                    <div className="space-y-2">
                      <p className="font-medium">Locations in cache:</p>
                      {Object.keys(cachedData.servers || {}).map(location => (
                        <div key={location} className="flex justify-between text-xs">
                          <span className="text-cyan-600 dark:text-cyan-400">{location}</span>
                          <span className="text-gray-600 dark:text-gray-400">
                            {cachedData.servers?.[location]?.length || 0} servers
                          </span>
                        </div>
                      ))}
                    </div>
                    <details className="mt-2">
                      <summary className="cursor-pointer text-xs text-gray-600 dark:text-gray-400">Raw JSON</summary>
                      <pre className="mt-1 p-2 bg-black/10 dark:bg-white/10 rounded text-xs overflow-auto max-h-32 text-card-foreground">
                        {JSON.stringify(cachedData, null, 2)}
                      </pre>
                    </details>
                  </div>
                </details>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Server Cache Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Speedtest Server Cache
          </CardTitle>
          <CardDescription>
            Manage your comprehensive speedtest server list and add custom locations.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Dual Cache Status Display */}
          <div className="grid gap-4">
            {/* Comprehensive Cache */}
            <div className="flex items-center justify-between p-4 bg-blue-50 dark:bg-blue-950/20 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="flex-1">
                <p className="font-medium text-blue-800 dark:text-blue-200 flex items-center gap-2">
                  🌐 Comprehensive Cache
                </p>
                <p className="text-sm text-blue-600 dark:text-blue-300">
                  {cachedData ? 
                    `${cachedData.totalServers || 0} servers from ${cachedData.locations?.length || 0} locations` :
                    "Empty - no global server data"
                  }
                </p>
                {cachedData?.lastUpdated && (
                  <p className="text-xs text-blue-500 dark:text-blue-400 mt-1">
                    Updated: {new Date(cachedData.lastUpdated).toLocaleString()}
                  </p>
                )}
              </div>
            </div>

            {/* Location Cache */}
            <div className="flex items-center justify-between p-4 bg-green-50 dark:bg-green-950/20 rounded-lg border border-green-200 dark:border-green-800">
              <div className="flex-1">
                <p className="font-medium text-green-800 dark:text-green-200 flex items-center gap-2">
                  📍 Location Cache
                </p>
                {(() => {
                  const locationData = getCachedLocationData();
                  return (
                    <>
                      <p className="text-sm text-green-600 dark:text-green-300">
                        {locationData ? 
                          `${locationData.totalServers || 0} servers from ${Object.keys(locationData.locations || {}).length} coordinate locations` :
                          "Empty - no coordinate-based servers"
                        }
                      </p>
                      {locationData?.lastUpdated && (
                        <p className="text-xs text-green-500 dark:text-green-400 mt-1">
                          Updated: {new Date(locationData.lastUpdated).toLocaleString()}
                        </p>
                      )}
                      {locationData && Object.keys(locationData.locations || {}).length > 0 && (
                        <p className="text-xs text-green-500 dark:text-green-400 mt-1">
                          Locations: {Object.keys(locationData.locations).join(', ')}
                        </p>
                      )}
                    </>
                  );
                })()}
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-2 justify-end pt-4 border-t border-border/50">
            <Button
              variant="outline"
              onClick={handleRefreshServers}
              disabled={isRefreshing}
              size="sm"
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
              {isRefreshing ? "Refreshing..." : "Refresh"}
            </Button>
            <Button
              variant="outline"
              onClick={handleClearCache}
              disabled={isRefreshing}
              size="sm"
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Clear Both Caches
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Add Custom Location */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-4 w-4" />
            Add Servers by Location
          </CardTitle>
          <CardDescription>
            Add speedtest servers from a specific geographic location using coordinates.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
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

            <div className="flex gap-2">
              <Button
                onClick={() => handleAddLocation(false)}
                disabled={isAddingLocation || !customLocation.trim()}
                className="flex-1 sm:flex-none sm:w-auto"
              >
                <MapPin className={`h-4 w-4 mr-2 ${isAddingLocation ? "animate-pulse" : ""}`} />
                {isAddingLocation ? "Adding Servers..." : "Add Servers for Location"}
              </Button>
            </div>
          </div>

          <div className="p-3 bg-blue-50 dark:bg-blue-950/20 rounded-lg border border-blue-200 dark:border-blue-800">
            <p className="text-sm text-blue-700 dark:text-blue-300">
              <strong>Tip:</strong> You can find coordinates using Google Maps. Right-click on a location 
              and select "What's here?" to get the latitude and longitude.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Cached Servers Explorer */}
      {(() => {
        const hasComprehensiveData = cachedData && cachedData.locations && cachedData.locations.length > 0;
        const locationData = getCachedLocationData();
        const hasLocationData = locationData && Object.keys(locationData.locations || {}).length > 0;
        
        if (!hasComprehensiveData && !hasLocationData) return null;
        
        return (
          <Card>
            <CardHeader>
              <CardTitle>Cached Servers Explorer</CardTitle>
              <CardDescription>
                Browse and explore your cached speedtest servers from both comprehensive and location caches.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Comprehensive Cache Servers */}
              {hasComprehensiveData && cachedData.locations.map((location) => {
                const servers = cachedData.servers?.[location] || [];
                const isExpanded = expandedLocations.has(location);
                
                return (
                  <div key={`comp-${location}`} className="border border-border/50 dark:border-border rounded-lg bg-card/50">
                    <button
                      onClick={() => toggleLocationExpansion(location)}
                      className="w-full flex items-center justify-between p-3 text-left hover:bg-muted/30 dark:hover:bg-muted/50 transition-colors rounded-t-lg"
                    >
                      <div>
                        <p className="font-medium text-gray-900 dark:text-white flex items-center gap-2">
                          🌐 {location}
                          <span className="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded">
                            Comprehensive
                          </span>
                        </p>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {servers.length} server{servers.length !== 1 ? 's' : ''}
                        </p>
                      </div>
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                      )}
                    </button>
                    
                    {isExpanded && (
                      <div className="border-t border-border/30 bg-muted/10 dark:bg-muted/20">
                        <div className="p-3 space-y-3">
                          {servers.map((server: Server, index: number) => (
                            <div
                              key={server.id || index}
                              className="p-3 bg-card dark:bg-card/80 border border-border/30 rounded-md text-sm shadow-sm hover:shadow-md transition-shadow"
                            >
                              <p className="font-mono text-xs text-gray-600 dark:text-gray-400 mb-2 break-all select-all">
                                ID: {server.id}
                              </p>
                              <p className="text-gray-900 dark:text-white leading-relaxed">
                                {formatServerInfo(server)}
                              </p>
                              {server.distance !== undefined && (
                                <div className="mt-2 flex items-center text-xs text-gray-600 dark:text-gray-400">
                                  <span className="inline-block w-2 h-2 bg-green-500 dark:bg-green-400 rounded-full mr-2"></span>
                                  {server.distance.toFixed(1)}km away
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
              
              {/* Location Cache Servers */}
              {hasLocationData && (() => {
                // Group servers by country instead of location
                const serversByCountry: Record<string, Server[]> = {};
                Object.entries(locationData.locations).forEach(([, servers]) => {
                  servers.forEach(server => {
                    const country = server.country || 'Unknown Country';
                    if (!serversByCountry[country]) {
                      serversByCountry[country] = [];
                    }
                    serversByCountry[country].push(server);
                  });
                });

                return Object.entries(serversByCountry).map(([country, servers]) => {
                  const isExpanded = expandedLocations.has(`country-${country}`);
                  
                  return (
                    <div key={`country-${country}`} className="border border-border/50 dark:border-border rounded-lg bg-card/50">
                      <button
                        onClick={() => toggleLocationExpansion(`country-${country}`)}
                        className="w-full flex items-center justify-between p-3 text-left hover:bg-muted/30 dark:hover:bg-muted/50 transition-colors rounded-t-lg"
                      >
                        <div>
                          <p className="font-medium text-gray-900 dark:text-white flex items-center gap-2">
                            📍 {country}
                            <span className="text-xs bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 px-2 py-0.5 rounded">
                              Location-based
                            </span>
                          </p>
                          <p className="text-sm text-gray-600 dark:text-gray-400">
                            {servers.length} server{servers.length !== 1 ? 's' : ''}
                          </p>
                        </div>
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                        )}
                      </button>
                      
                      {isExpanded && (
                        <div className="border-t border-border/30 bg-muted/10 dark:bg-muted/20">
                          <div className="p-3 space-y-3">
                            {servers.map((server: Server, index: number) => (
                              <div
                                key={server.id || index}
                                className="p-3 bg-card dark:bg-card/80 border border-border/30 rounded-md text-sm shadow-sm hover:shadow-md transition-shadow"
                              >
                                <p className="font-mono text-xs text-gray-600 dark:text-gray-400 mb-2 break-all select-all">
                                  ID: {server.id}
                                </p>
                                <div className="space-y-1">
                                  <p className="font-medium text-gray-900 dark:text-white">
                                    {server.name || server.sponsor}
                                  </p>
                                  <p className="text-xs text-gray-600 dark:text-gray-400 break-all" title={server.host}>
                                    {server.host}
                                  </p>
                                  <p className="text-sm text-gray-700 dark:text-gray-300">
                                    {server.country} - {server.distance !== undefined ? `${server.distance.toFixed(0)} km` : 'Distance unknown'}
                                  </p>
                                </div>
                                {server.distance !== undefined && (
                                  <div className="mt-2 flex items-center text-xs text-gray-600 dark:text-gray-400">
                                    <span className="inline-block w-2 h-2 bg-green-500 dark:bg-green-400 rounded-full mr-2"></span>
                                    {server.distance.toFixed(1)}km away
                                  </div>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  );
                });
              })()}
            </CardContent>
          </Card>
        );
      })()}
    </div>
  );
}
