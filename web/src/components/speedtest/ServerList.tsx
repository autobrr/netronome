/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useMemo } from "react";
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
import { Server } from "@/types/types";
import { ChevronDownIcon, ChevronUpDownIcon } from "@heroicons/react/20/solid";

interface ServerListProps {
  servers: Server[];
  selectedServers: Server[];
  onSelect: (server: Server) => void;
  multiSelect: boolean;
  onMultiSelectChange: (enabled: boolean) => void;
  onRunTest: () => Promise<void>;
  isLoading: boolean;
}

export const ServerList: React.FC<ServerListProps> = ({
  servers,
  selectedServers,
  onSelect,
  multiSelect,
  onMultiSelectChange,
  onRunTest,
  isLoading,
}) => {
  const [displayCount, setDisplayCount] = useState(3);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterCountry, setFilterCountry] = useState("");

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

    // Sort servers by distance using the distance property
    return filtered.sort((a, b) => a.distance - b.distance); // Ascending order
  }, [servers, searchTerm, filterCountry]);

  return (
    <Disclosure defaultOpen={true}>
      {({ open }) => (
        <div className="flex flex-col h-full">
          <DisclosureButton
            className={`flex justify-between items-center w-full px-4 py-2 bg-gray-850/95 ${
              open ? "rounded-t-xl border-b-0" : "rounded-xl"
            } shadow-lg border-b-0 border-gray-900 text-left`}
          >
            <div className="flex flex-col">
              <h2 className="text-white text-xl font-semibold p-1 select-none">
                Server Selection
              </h2>
            </div>
            <div className="flex items-center gap-2">
              {selectedServers.length > 0 && (
                <span className="text-gray-400">
                  {selectedServers.length} server
                  {selectedServers.length !== 1 ? "s" : ""} selected
                </span>
              )}
              <ChevronDownIcon
                className={`${
                  open ? "transform rotate-180" : ""
                } w-5 h-5 text-gray-400 transition-transform duration-200`}
              />
            </div>
          </DisclosureButton>

          {open && (
            <div className="bg-gray-850/95 px-4 rounded-b-xl shadow-lg flex-1">
              <div className="flex flex-col pl-1">
                <p className="text-gray-400 text-sm select-none pointer-events-none">
                  Select one or more servers to test
                </p>
              </div>
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
                  {/* Multi-select Toggle */}
                  <Field className="flex items-center gap-3">
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

                  {/* Run Test Button */}
                  <div className="relative inline-block group">
                    <button
                      onClick={onRunTest}
                      disabled={isLoading || selectedServers.length === 0}
                      className={`
                        px-4 py-2 
                        rounded-lg 
                        shadow-md
                        transition-colors
                        border
                        ${
                          isLoading || selectedServers.length === 0
                            ? "bg-gray-700 text-gray-400 cursor-not-allowed border-gray-900"
                            : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                        }
                      `}
                    >
                      {isLoading
                        ? "Running Test..."
                        : selectedServers.length === 0
                        ? "Select a server"
                        : "Run Test"}
                    </button>
                    {selectedServers.length === 0 && (
                      <div className="absolute bottom-full border border-gray-900 left-1/2 transform -translate-x-1/2 mb-2 px-3 py-1 text-sm text-white bg-gray-800 rounded-md invisible group-hover:visible transition-all duration-200 whitespace-nowrap">
                        Pick a server first
                      </div>
                    )}
                  </div>
                </div>

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
                  <Listbox value={filterCountry} onChange={setFilterCountry}>
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

                {/* Server Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {filteredServers.slice(0, displayCount).map((server) => {
                    const distance = server.distance; // Use the distance from the server response
                    return (
                      <motion.div
                        key={server.id}
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.3 }}
                      >
                        <button
                          onClick={() => onSelect(server)}
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

                {/* Load More Button */}
                {filteredServers.length > displayCount && (
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
        </div>
      )}
    </Disclosure>
  );
};
