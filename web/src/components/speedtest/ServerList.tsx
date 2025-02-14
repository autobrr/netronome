/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useMemo, useEffect } from "react";
import { motion } from "motion/react";
import {
  Switch,
  Field,
  Label,
  Disclosure,
  DisclosureButton,
  Listbox,
  Transition,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
} from "@headlessui/react";
import { SavedIperfServer, Server } from "@/types/types";
import { ChevronDownIcon, ChevronUpDownIcon } from "@heroicons/react/20/solid";
import { IperfServerModal } from "./IperfServerModal";
import { getApiUrl } from "@/utils/baseUrl";

interface ServerListProps {
  servers: Server[];
  selectedServers: Server[];
  onSelect: (server: Server) => void;
  multiSelect: boolean;
  onMultiSelectChange: (enabled: boolean) => void;
  onRunTest: () => Promise<void>;
  isLoading: boolean;
  useIperf: boolean;
  onIperfChange: (enabled: boolean) => void;
}

export const ServerList: React.FC<ServerListProps> = ({
  servers,
  selectedServers,
  onSelect,
  // multiSelect,
  // onMultiSelectChange,
  onRunTest,
  isLoading,
  useIperf,
  onIperfChange,
}) => {
  const [displayCount, setDisplayCount] = useState(3);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterCountry, setFilterCountry] = useState("");
  const [iperfHost, setIperfHost] = useState("");
  const [savedIperfServers, setSavedIperfServers] = useState<
    SavedIperfServer[]
  >([]);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [saveModalOpen, setSaveModalOpen] = useState(false);
  const [serverToDelete, setServerToDelete] = useState<number | null>(null);
  const [newServerDetails, setNewServerDetails] = useState<{
    host: string;
    port: string;
  }>({ host: "", port: "5201" });
  const [isOpen] = useState(() => {
    const saved = localStorage.getItem("server-list-open");
    return saved === null ? true : saved === "true";
  });

  // Get unique countries for filter dropdown
  const countries = useMemo(() => {
    const uniqueCountries = new Set(servers.map((server) => server.country));
    return Array.from(uniqueCountries).sort();
  }, [servers]);

  // Filter and sort servers
  const filteredServers = useMemo(() => {
    const filtered = servers.filter((server) => {
      const matchesSearch =
        searchTerm === "" ||
        server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.sponsor.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.country.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesCountry =
        filterCountry === "" || server.country === filterCountry;

      return matchesSearch && matchesCountry;
    });

    return filtered.sort((a, b) => a.distance - b.distance);
  }, [servers, searchTerm, filterCountry]);

  const handleServerSelect = (server: Server) => {
    onSelect(server);
  };

  // Load the saved iperf state when component mounts
  useEffect(() => {
    const savedIperfState = localStorage.getItem("useIperf");
    if (savedIperfState !== null && useIperf !== (savedIperfState === "true")) {
      onIperfChange(savedIperfState === "true");
    }
  }, [onIperfChange, useIperf]);

  // Handle iperf toggle change
  const handleIperfChange = (enabled: boolean) => {
    // Clear selected servers when toggling iperf (both enabling and disabling)
    selectedServers.forEach((server) => onSelect(server));
    // Save the new state to localStorage
    localStorage.setItem("useIperf", enabled.toString());
    onIperfChange(enabled);
  };

  const fetchSavedIperfServers = async () => {
    const response = await fetch(getApiUrl("/iperf/servers"));
    if (!response.ok) throw new Error("Failed to fetch saved iperf servers");
    const data = await response.json();
    setSavedIperfServers(data);
  };

  const saveIperfServer = async (name: string, host: string, port: number) => {
    try {
      const response = await fetch(getApiUrl("/iperf/servers"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ name, host, port }),
      });

      if (!response.ok) throw new Error("Failed to save iperf server");

      // Clear the input after successful save
      setIperfHost("");

      // Refresh the list of saved servers
      await fetchSavedIperfServers();
    } catch (error) {
      console.error("Failed to save server:", error);
    }
  };

  const deleteSavedServer = async (id: number) => {
    try {
      const response = await fetch(getApiUrl(`/iperf/servers/${id}`), {
        method: "DELETE",
      });

      if (!response.ok) throw new Error("Failed to delete iperf server");

      // Refresh the list of saved servers
      await fetchSavedIperfServers();
    } catch (error) {
      console.error("Failed to delete server:", error);
    }
  };

  // Fetch saved iperf servers when component mounts or iperf mode changes
  useEffect(() => {
    if (useIperf) {
      fetchSavedIperfServers().catch((error) => {
        console.error("Failed to fetch iperf servers:", error);
      });
    }
  }, [useIperf]); // Re-run when useIperf changes

  return (
    <Disclosure defaultOpen={isOpen}>
      {({ open }) => {
        useEffect(() => {
          localStorage.setItem("server-list-open", open.toString());
        }, [open]);

        return (
          <div className="flex flex-col h-full">
            <DisclosureButton
              className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
                open ? "rounded-t-xl" : "rounded-xl"
              } shadow-lg border-b-0 border-gray-900 text-left`}
            >
              <div className="flex flex-col">
                <h2 className="text-white text-xl font-semibold p-1 select-none">
                  Server Selection
                </h2>
                <p className="text-gray-400 text-sm pl-1 pb-1">
                  Choose between speedtest.net or iperf3 servers
                </p>
              </div>
              <div className="flex items-center gap-2">
                {/* {!useIperf && selectedServers.length > 0 && (
                  <span className="text-gray-400">
                    {selectedServers.length} server
                    {selectedServers.length !== 1 ? "s" : ""} selected
                  </span>
                )} */}
                <ChevronDownIcon
                  className={`${
                    open ? "transform rotate-180" : ""
                  } w-5 h-5 text-gray-400 transition-transform duration-200`}
                />
              </div>
            </DisclosureButton>

            {open && (
              <div className="bg-gray-850/95 px-4 pt-2 rounded-b-xl shadow-lg flex-1">
                <div className="flex flex-col pl-1"></div>
                <motion.div
                  className="mt-1 px-1 select-none pointer-events-none server-list-animate pb-4"
                  initial={{ opacity: 0, y: -20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    duration: 0.5,
                    type: "spring",
                    stiffness: 300,
                    damping: 20,
                  }}
                  onAnimationComplete={() => {
                    const element = document.querySelector(
                      ".server-list-animate"
                    );
                    if (element) {
                      element.classList.remove(
                        "select-none",
                        "pointer-events-none"
                      );
                    }
                  }}
                >
                  {/* Controls Header */}
                  <div className="flex justify-between items-center mb-4">
                    <div className="flex items-center gap-6">
                      {/* Multi-select Toggle */}
                      {/* TODO: backend needs some work for multiple server tests
                      <Field
                        className={`flex items-center gap-3 transition-opacity duration-200 ${
                          useIperf
                            ? "opacity-30 pointer-events-none"
                            : "opacity-100"
                        }`}
                      >
                        <Label className="text-sm text-gray-400">
                          Multi-select
                        </Label>
                        <Switch
                          checked={multiSelect}
                          onChange={onMultiSelectChange}
                          className={`${
                            multiSelect ? "bg-blue-500" : "bg-gray-700"
                          } relative inline-flex h-6 w-11 items-center rounded-full`}
                        >
                          <span
                            className={`${
                              multiSelect ? "translate-x-6" : "translate-x-1"
                            } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
                          />
                        </Switch>
                      </Field>
                      */}

                      {/* iperf3 Toggle */}
                      <Field className="flex items-center gap-2 sm:gap-3">
                        <Label className="text-xs sm:text-sm text-gray-400">
                          Use iperf3
                        </Label>
                        <Switch
                          checked={useIperf}
                          onChange={handleIperfChange}
                          className={`${
                            useIperf ? "bg-blue-500" : "bg-gray-700"
                          } relative inline-flex h-5 sm:h-6 w-9 sm:w-11 items-center rounded-full`}
                        >
                          <span
                            className={`${
                              useIperf
                                ? "translate-x-5 sm:translate-x-6"
                                : "translate-x-1"
                            } inline-block h-3 sm:h-4 w-3 sm:w-4 transform rounded-full bg-white transition-transform`}
                          />
                        </Switch>
                      </Field>
                    </div>

                    {/* Run Test Button */}
                    <button
                      onClick={onRunTest}
                      disabled={
                        isLoading ||
                        (!useIperf && selectedServers.length === 0) ||
                        (useIperf && selectedServers.length === 0)
                      }
                      className={`
                        px-4 py-2 
                        rounded-lg 
                        shadow-md
                        transition-colors
                        border
                        ${
                          isLoading ||
                          (!useIperf && selectedServers.length === 0) ||
                          (useIperf && selectedServers.length === 0)
                            ? "bg-gray-700 text-gray-400 cursor-not-allowed border-gray-900"
                            : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                        }
                      `}
                    >
                      Run
                    </button>
                  </div>

                  {useIperf && (
                    <div className="flex flex-col gap-4 mb-4">
                      {/* Section for saved servers */}
                      {savedIperfServers.length > 0 && (
                        <div className="flex flex-col gap-2">
                          <h3 className="text-sm font-medium text-gray-400">
                            Saved Servers
                          </h3>
                          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
                            <Listbox
                              value=""
                              onChange={(selectedServerId: string) => {
                                const server = savedIperfServers.find(
                                  (s) => s.id.toString() === selectedServerId
                                );
                                if (server) {
                                  const iperfServer: Server = {
                                    id: `iperf3-${server.host}:${server.port}`,
                                    name: server.name,
                                    host: `${server.host}:${server.port}`,
                                    location: "Saved",
                                    distance: 0,
                                    country: "Saved",
                                    sponsor: "Saved iperf3",
                                    latitude: 0,
                                    longitude: 0,
                                    isIperf: true,
                                  };
                                  selectedServers.forEach((s) => onSelect(s));
                                  onSelect(iperfServer);
                                }
                              }}
                            >
                              <div className="relative w-full sm:min-w-[200px] md:min-w-[300px] lg:min-w-[400px]">
                                <ListboxButton className="relative w-full px-4 py-2 bg-gray-800/50 border border-gray-900 rounded-lg text-left text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                                  <span className="block truncate">
                                    Select a server
                                  </span>
                                  <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                                    <ChevronUpDownIcon
                                      className="h-5 w-5 text-gray-400"
                                      aria-hidden="true"
                                    />
                                  </span>
                                </ListboxButton>
                                <Transition
                                  enter="transition duration-100 ease-out"
                                  enterFrom="transform scale-95 opacity-0"
                                  enterTo="transform scale-100 opacity-100"
                                  leave="transition duration-75 ease-out"
                                  leaveFrom="transform scale-100 opacity-100"
                                  leaveTo="transform scale-95 opacity-0"
                                >
                                  <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-gray-800 border border-gray-900 py-1 shadow-lg focus:outline-none">
                                    {savedIperfServers.map((server) => (
                                      <ListboxOption
                                        key={server.id}
                                        value={server.id.toString()}
                                        className={({ active }) =>
                                          `relative cursor-pointer select-none py-2 px-4 ${
                                            active
                                              ? "bg-blue-500/10 text-blue-200"
                                              : "text-gray-300"
                                          }`
                                        }
                                      >
                                        {({ selected }) => (
                                          <div className="flex items-center justify-between space-x-2">
                                            <div className="flex-1 min-w-0">
                                              <span
                                                className={`block truncate ${
                                                  selected
                                                    ? "font-medium"
                                                    : "font-normal"
                                                }`}
                                              >
                                                {server.name}
                                              </span>
                                              <span className="block truncate text-xs text-gray-400">
                                                {server.host}:{server.port}
                                              </span>
                                            </div>
                                            <button
                                              onClick={(e) => {
                                                e.stopPropagation();
                                                setServerToDelete(server.id);
                                                setDeleteModalOpen(true);
                                              }}
                                              className="px-2 py-1 text-sm text-red-400 hover:text-red-300 transition-colors"
                                            >
                                              Delete
                                            </button>
                                          </div>
                                        )}
                                      </ListboxOption>
                                    ))}
                                  </ListboxOptions>
                                </Transition>
                              </div>
                            </Listbox>

                            {selectedServers.length > 0 && (
                              <div className="flex items-center gap-2 w-full sm:w-[300px] md:w-[400px] min-w-0">
                                <div className="flex items-center gap-2 flex-1 min-w-0">
                                  <span className="text-gray-400 text-sm whitespace-nowrap">
                                    Selected:
                                  </span>
                                  <span
                                    className="text-gray-300 text-sm truncate"
                                    title={`${selectedServers[0].name} (${selectedServers[0].host})`}
                                  >
                                    {selectedServers[0].name}
                                  </span>
                                </div>
                                <button
                                  onClick={() =>
                                    selectedServers.forEach((s) => onSelect(s))
                                  }
                                  className="px-2 py-1 text-sm text-gray-400 hover:text-gray-200 transition-colors shrink-0"
                                >
                                  ✕
                                </button>
                              </div>
                            )}
                          </div>
                        </div>
                      )}

                      {/* Section for adding new servers */}
                      <div className="flex flex-col gap-2">
                        <h3 className="text-sm font-medium text-gray-400">
                          {savedIperfServers.length === 0
                            ? "Add iperf3 Server"
                            : "Add New Server"}
                        </h3>
                        <div className="flex gap-4">
                          <input
                            type="text"
                            value={iperfHost}
                            onChange={(e) => setIperfHost(e.target.value)}
                            placeholder={
                              window.innerWidth < 640
                                ? "iperf.example.com:5201"
                                : "Enter iperf3 server host (e.g., iperf.example.com:5201)"
                            }
                            className="flex-1 px-2 sm:px-4 py-1.5 sm:py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg text-xs sm:text-sm shadow-md"
                          />
                          <button
                            onClick={() => {
                              const [host, portStr] = iperfHost.split(":");
                              setNewServerDetails({
                                host,
                                port: portStr || "5201",
                              });
                              setSaveModalOpen(true);
                            }}
                            disabled={!iperfHost.trim()}
                            className={`px-4 py-2 rounded-lg transition-colors border shadow-md ${
                              !iperfHost.trim()
                                ? "bg-gray-700 text-gray-400 cursor-not-allowed border-gray-900"
                                : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                            }`}
                          >
                            Save
                          </button>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Server Search and Filter Controls */}
                  {!useIperf && (
                    <div className="flex flex-col md:flex-row gap-4 mb-4">
                      {/* Search Input */}
                      <div className="flex-1">
                        <input
                          type="text"
                          placeholder="Search servers..."
                          className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                          value={searchTerm}
                          onChange={(e) => setSearchTerm(e.target.value)}
                        />
                      </div>

                      {/* Country Filter */}
                      <Listbox
                        value={filterCountry}
                        onChange={setFilterCountry}
                      >
                        <div className="relative min-w-[160px]">
                          <ListboxButton className="relative w-full px-4 py-2 bg-gray-800/50 border border-gray-900 rounded-lg text-left text-gray-300 shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50">
                            <span className="block truncate">
                              {filterCountry || "All Countries"}
                            </span>
                            <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                              <ChevronUpDownIcon
                                className="h-5 w-5 text-gray-400"
                                aria-hidden="true"
                              />
                            </span>
                          </ListboxButton>
                          <Transition
                            enter="transition duration-100 ease-out"
                            enterFrom="transform scale-95 opacity-0"
                            enterTo="transform scale-100 opacity-100"
                            leave="transition duration-75 ease-out"
                            leaveFrom="transform scale-100 opacity-100"
                            leaveTo="transform scale-95 opacity-0"
                          >
                            <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-gray-800 border border-gray-900 py-1 shadow-lg focus:outline-none">
                              <ListboxOption
                                className={({ focus }) =>
                                  `relative cursor-pointer select-none py-2 px-4 ${
                                    focus
                                      ? "bg-blue-500/10 text-blue-200"
                                      : "text-gray-300"
                                  }`
                                }
                                value=""
                              >
                                All Countries
                              </ListboxOption>
                              {countries.map((country) => (
                                <ListboxOption
                                  key={country}
                                  value={country}
                                  className={({ focus }) =>
                                    `relative cursor-pointer select-none py-2 px-4 ${
                                      focus
                                        ? "bg-blue-500/10 text-blue-200"
                                        : "text-gray-300"
                                    }`
                                  }
                                >
                                  {country}
                                </ListboxOption>
                              ))}
                            </ListboxOptions>
                          </Transition>
                        </div>
                      </Listbox>
                    </div>
                  )}

                  {/* Server Grid */}
                  {!useIperf && (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {filteredServers.slice(0, displayCount).map((server) => {
                        const distance = server.distance;
                        return (
                          <motion.div
                            key={server.id}
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.3 }}
                          >
                            <button
                              onClick={() => handleServerSelect(server)}
                              className={`w-full p-4 rounded-lg text-left transition-colors ${
                                selectedServers.some((s) => s.id === server.id)
                                  ? "bg-blue-500/10 border-blue-400/50 shadow-lg"
                                  : "bg-gray-800/50 border-gray-900 hover:bg-gray-800 shadow-lg"
                              } border`}
                            >
                              <div className="flex flex-col gap-1">
                                <span className="text-blue-300 font-medium truncate">
                                  {server.sponsor}
                                </span>
                                <span className="text-gray-400 text-sm">
                                  {server.name}
                                  <span
                                    className="block truncate text-xs"
                                    title={server.host}
                                  >
                                    {server.host}
                                  </span>
                                </span>
                                <span className="text-gray-400 text-sm mt-1">
                                  {server.country} - {Math.floor(distance)} km
                                </span>
                              </div>
                            </button>
                          </motion.div>
                        );
                      })}
                    </div>
                  )}

                  {/* Load More Button */}
                  {!useIperf && filteredServers.length > displayCount && (
                    <div className="flex justify-center mt-6">
                      <button
                        onClick={() => setDisplayCount((prev) => prev + 6)}
                        className="px-4 py-2 bg-gray-800/50 border border-gray-900/80 text-gray-300/50 hover:text-gray-300 rounded-lg hover:bg-gray-800 transition-colors"
                      >
                        Load More
                      </button>
                    </div>
                  )}
                </motion.div>
              </div>
            )}
            <IperfServerModal
              isOpen={deleteModalOpen}
              onClose={() => {
                setDeleteModalOpen(false);
                setServerToDelete(null);
              }}
              onConfirm={() => {
                if (serverToDelete) {
                  deleteSavedServer(serverToDelete);
                }
              }}
              title="Delete Server"
              message="Are you sure you want to delete this server? This action cannot be undone."
              confirmText="Delete"
              confirmStyle="danger"
            />

            <IperfServerModal
              isOpen={saveModalOpen}
              onClose={() => {
                setSaveModalOpen(false);
                setNewServerDetails({ host: "", port: "5201" });
              }}
              onConfirm={(name) => {
                if (name && newServerDetails.host) {
                  saveIperfServer(
                    name,
                    newServerDetails.host,
                    parseInt(newServerDetails.port)
                  );
                  setIperfHost("");
                }
              }}
              title="Save Server"
              message="Enter a name for this iperf server"
              confirmText="Save"
              serverDetails={newServerDetails}
            />
          </div>
        );
      }}
    </Disclosure>
  );
};
