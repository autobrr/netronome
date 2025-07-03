/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useMemo, useEffect, Fragment } from "react";
import { motion } from "motion/react";
import {
  Radio,
  RadioGroup,
  Label,
  Disclosure,
  DisclosureButton,
  Listbox,
  Transition,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
  Dialog,
} from "@headlessui/react";
import { SavedIperfServer, Server } from "@/types/types";
import {
  ChevronDownIcon,
  ChevronUpDownIcon,
  XMarkIcon,
} from "@heroicons/react/20/solid";
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
  testType: "speedtest" | "iperf" | "librespeed";
  onTestTypeChange: (testType: "speedtest" | "iperf" | "librespeed") => void;
}

export const ServerList: React.FC<ServerListProps> = ({
  servers,
  selectedServers,
  onSelect,
  // multiSelect,
  // onMultiSelectChange,
  onRunTest,
  isLoading,
  testType,
  onTestTypeChange,
}) => {
  const [displayCount, setDisplayCount] = useState(3);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterCountry, setFilterCountry] = useState("");
  const [iperfSearchTerm, setIperfSearchTerm] = useState("");
  const [addServerModalOpen, setAddServerModalOpen] = useState(false);
  const [iperfDisplayCount, setIperfDisplayCount] = useState(3);
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

  // Filter saved iperf servers
  const filteredIperfServers = useMemo(() => {
    return savedIperfServers.filter((server) => {
      const matchesSearch =
        iperfSearchTerm === "" ||
        server.name.toLowerCase().includes(iperfSearchTerm.toLowerCase()) ||
        server.host.toLowerCase().includes(iperfSearchTerm.toLowerCase());
      return matchesSearch;
    });
  }, [savedIperfServers, iperfSearchTerm]);

  const handleServerSelect = (server: Server) => {
    onSelect(server);
  };

  // Load the saved test type when component mounts
  useEffect(() => {
    const savedTestType = localStorage.getItem("testType") as
      | "speedtest"
      | "iperf"
      | "librespeed"
      | null;
    if (savedTestType && testType !== savedTestType) {
      onTestTypeChange(savedTestType);
    }
  }, [onTestTypeChange, testType]);

  // Handle test type change
  const handleTestTypeChange = (
    newTestType: "speedtest" | "iperf" | "librespeed"
  ) => {
    // Clear selected servers when toggling
    selectedServers.forEach((server) => onSelect(server));
    // Save the new state to localStorage
    localStorage.setItem("testType", newTestType);
    onTestTypeChange(newTestType);
  };

  const fetchSavedIperfServers = async () => {
    const response = await fetch(getApiUrl("/iperf/servers"));
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(
        errorData.message || "Failed to fetch saved iperf servers"
      );
    }
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

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || "Failed to save iperf server");
      }

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

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || "Failed to delete iperf server");
      }

      // Refresh the list of saved servers
      await fetchSavedIperfServers();
    } catch (error) {
      console.error("Failed to delete server:", error);
    }
  };

  // Fetch saved iperf servers when component mounts or iperf mode changes
  useEffect(() => {
    if (testType === "iperf") {
      fetchSavedIperfServers().catch((error) => {
        console.error("Failed to fetch iperf servers:", error);
      });
    }
  }, [testType]); // Re-run when useIperf changes

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

                      {/* Test Type Radio Group */}
                      <RadioGroup
                        value={testType}
                        onChange={handleTestTypeChange}
                        className="flex items-center gap-4"
                      >
                        <RadioOption value="speedtest">Speedtest</RadioOption>
                        <RadioOption value="iperf">iperf3</RadioOption>
                        <RadioOption value="librespeed">Librespeed</RadioOption>
                      </RadioGroup>
                    </div>

                    {/* Run Test Button */}
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
                      Run
                    </button>
                  </div>

                  {testType === "iperf" && (
                    <div className="flex flex-col gap-4 mb-4">
                      {/* Search Input and Add Button for iperf3 servers */}
                      <div className="flex flex-col md:flex-row gap-4">
                        <div className="flex-1">
                          <input
                            type="text"
                            placeholder="Search saved servers..."
                            className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                            value={iperfSearchTerm}
                            onChange={(e) => setIperfSearchTerm(e.target.value)}
                          />
                        </div>
                        <button
                          onClick={() => setAddServerModalOpen(true)}
                          className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors border border-blue-600 hover:border-blue-700 shadow-md"
                        >
                          Add Server
                        </button>
                      </div>
                    </div>
                  )}

                  {/* Server Search and Filter Controls */}
                  {(testType === "speedtest" || testType === "librespeed") && (
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
                  {testType === "iperf" ? (
                    <>
                      {filteredIperfServers.length === 0 ? (
                        <div className="flex flex-col items-center justify-center py-12 px-4">
                          <div className="text-center max-w-md">
                            <div className="text-gray-400 text-lg mb-2">🔧</div>
                            <h3 className="text-lg font-medium text-gray-300 mb-2">
                              No iperf3 servers found
                            </h3>
                            <p className="text-gray-400 text-sm mb-4">
                              Add your first iperf3 server using the input
                              above. Enter the server address and port (e.g.,
                              iperf.example.com:5201)
                            </p>
                          </div>
                        </div>
                      ) : (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                          {filteredIperfServers
                            .slice(0, iperfDisplayCount)
                            .map((server) => {
                              const iperfServer: Server = {
                                id: `iperf3-${server.host}:${server.port}`,
                                name: server.name,
                                host: `${server.host}:${server.port}`,
                                location: "Saved",
                                distance: 0,
                                country: "Saved",
                                sponsor: "iperf3",
                                latitude: 0,
                                longitude: 0,
                                isIperf: true,
                              };
                              return (
                                <motion.div
                                  key={server.id}
                                  initial={{ opacity: 0, y: 20 }}
                                  animate={{ opacity: 1, y: 0 }}
                                  transition={{ duration: 0.3 }}
                                >
                                  <button
                                    onClick={() => {
                                      selectedServers.forEach((s) =>
                                        onSelect(s)
                                      );
                                      onSelect(iperfServer);
                                    }}
                                    className={`w-full p-4 rounded-lg text-left transition-colors relative ${
                                      selectedServers.some(
                                        (s) => s.id === iperfServer.id
                                      )
                                        ? "bg-blue-500/10 border-blue-400/50 shadow-lg"
                                        : "bg-gray-800/50 border-gray-900 hover:bg-gray-800 shadow-lg"
                                    } border`}
                                  >
                                    <div className="flex flex-col gap-1 pr-8">
                                      <span className="text-blue-300 font-medium truncate">
                                        {server.name}
                                      </span>
                                      <span className="text-gray-400 text-sm">
                                        iperf3 Server
                                        <span
                                          className="block truncate text-xs"
                                          title={`${server.host}:${server.port}`}
                                        >
                                          {server.host}:{server.port}
                                        </span>
                                      </span>
                                      <span className="text-gray-400 text-sm mt-1">
                                        Custom Server
                                      </span>
                                    </div>
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        setServerToDelete(server.id);
                                        setDeleteModalOpen(true);
                                      }}
                                      className="absolute top-2 right-2 text-red-500 p-1 bg-red-900/50 border border-gray-900 rounded-md hover:bg-red-900/70 hover:text-red-400 transition-colors"
                                      title="Delete server"
                                    >
                                      <XMarkIcon className="h-4 w-4" />
                                    </button>
                                  </button>
                                </motion.div>
                              );
                            })}
                        </div>
                      )}

                      {/* Load More Button for iperf3 */}
                      {filteredIperfServers.length > iperfDisplayCount && (
                        <div className="flex justify-center mt-6">
                          <button
                            onClick={() =>
                              setIperfDisplayCount((prev) => prev + 6)
                            }
                            className="px-4 py-2 bg-gray-800/50 border border-gray-900/80 text-gray-300/50 hover:text-gray-300 rounded-lg hover:bg-gray-800 transition-colors"
                          >
                            Load More
                          </button>
                        </div>
                      )}
                    </>
                  ) : (
                    <>
                      {filteredServers.length === 0 &&
                      testType === "librespeed" ? (
                        <div className="flex flex-col items-center justify-center py-12 px-4">
                          <div className="text-center max-w-md">
                            <div className="text-gray-400 text-lg mb-2">📡</div>
                            <h3 className="text-lg font-medium text-gray-300 mb-2">
                              No Librespeed servers found
                            </h3>
                            <p className="text-gray-400 text-sm mb-4">
                              The librespeed-servers.json file was not found.
                              Please create this file in the same directory as
                              your config.toml to run librespeed tests.
                            </p>
                            <div className="text-xs text-gray-500 bg-gray-800/30 rounded-lg p-3 border border-gray-900">
                              <p className="mb-2">
                                Example librespeed-servers.json:
                              </p>
                              <pre className="text-left">
                                {`[
  {
    "id": 1,
    "name": "Example Server",
    "server": "https://example.com/backend",
    "dlURL": "garbage.php",
    "ulURL": "empty.php",
    "pingURL": "empty.php",
    "getIpURL": "getIP.php"
  }
]`}
                              </pre>
                            </div>
                          </div>
                        </div>
                      ) : (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                          {filteredServers
                            .slice(0, displayCount)
                            .map((server) => {
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
                                      selectedServers.some(
                                        (s) => s.id === server.id
                                      )
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
                                        {server.country} -{" "}
                                        {Math.floor(distance)} km
                                      </span>
                                    </div>
                                  </button>
                                </motion.div>
                              );
                            })}
                        </div>
                      )}
                    </>
                  )}

                  {/* Load More Button */}
                  {testType !== "iperf" &&
                    filteredServers.length > displayCount && (
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
                }
              }}
              title="Save Server"
              message="Enter a name for this iperf server"
              confirmText="Save"
              serverDetails={newServerDetails}
            />

            <AddServerModal
              isOpen={addServerModalOpen}
              onClose={() => setAddServerModalOpen(false)}
              onConfirm={(name, host, port) => {
                saveIperfServer(name, host, parseInt(port));
              }}
            />
          </div>
        );
      }}
    </Disclosure>
  );
};

const RadioOption: React.FC<{
  value: "speedtest" | "iperf" | "librespeed";
  children: React.ReactNode;
}> = ({ value, children }) => (
  <Radio value={value} as="div" className="flex items-center gap-2">
    {({ checked }) => (
      <div
        className={`cursor-pointer flex items-center gap-2 px-3 py-1 rounded-lg transition-colors ${
          checked
            ? "bg-blue-500/10 text-blue-200 border-blue-500/50"
            : "text-gray-400 hover:bg-gray-800 border-transparent"
        } border`}
      >
        <div
          className={`w-4 h-4 rounded-full border-2 ${
            checked ? "border-blue-400 bg-blue-500" : "border-gray-600"
          } transition-colors`}
        />
        <Label className="text-xs sm:text-sm cursor-pointer">{children}</Label>
      </div>
    )}
  </Radio>
);

interface AddServerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (name: string, host: string, port: string) => void;
}

const AddServerModal: React.FC<AddServerModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
}) => {
  const [serverName, setServerName] = useState("");
  const [serverHost, setServerHost] = useState("");
  const [serverPort, setServerPort] = useState("5201");

  const handleClose = () => {
    setServerName("");
    setServerHost("");
    setServerPort("5201");
    onClose();
  };

  const handleConfirm = () => {
    if (serverName.trim() && serverHost.trim()) {
      onConfirm(serverName.trim(), serverHost.trim(), serverPort.trim());
      handleClose();
    }
  };

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={handleClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md transform overflow-hidden rounded-xl bg-gray-850/95 border border-gray-900 p-6 shadow-xl transition-all">
                <Dialog.Title
                  as="h2"
                  className="text-xl font-semibold text-white mb-4"
                >
                  Add iperf3 Server
                </Dialog.Title>

                <div className="space-y-4">
                  <div className="space-y-2">
                    <label
                      htmlFor="serverName"
                      className="block text-sm font-medium text-gray-400"
                    >
                      Server Name
                    </label>
                    <input
                      type="text"
                      id="serverName"
                      value={serverName}
                      onChange={(e) => setServerName(e.target.value)}
                      placeholder="Enter a name for this server"
                      className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                      autoFocus
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      htmlFor="serverHost"
                      className="block text-sm font-medium text-gray-400"
                    >
                      Server Host
                    </label>
                    <input
                      type="text"
                      id="serverHost"
                      value={serverHost}
                      onChange={(e) => setServerHost(e.target.value)}
                      placeholder="iperf.example.com"
                      className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                    />
                  </div>
                  <div className="space-y-2">
                    <label
                      htmlFor="serverPort"
                      className="block text-sm font-medium text-gray-400"
                    >
                      Server Port
                    </label>
                    <input
                      type="number"
                      id="serverPort"
                      value={serverPort}
                      onChange={(e) => setServerPort(e.target.value)}
                      placeholder="5201"
                      className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                    />
                  </div>
                </div>

                <div className="mt-6 flex justify-end gap-3">
                  <button
                    type="button"
                    className="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 transition-colors"
                    onClick={handleClose}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={!serverName.trim() || !serverHost.trim()}
                    className={`px-4 py-2 rounded-lg shadow-md transition-colors border text-sm ${
                      !serverName.trim() || !serverHost.trim()
                        ? "bg-gray-700 text-gray-400 cursor-not-allowed border-gray-900"
                        : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                    }`}
                    onClick={handleConfirm}
                  >
                    Add Server
                  </button>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
};
