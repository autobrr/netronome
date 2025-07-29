/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useState, useMemo, useEffect } from "react";
import { motion } from "motion/react";
import { SavedIperfServer, Server } from "@/types/types";
import {
  ChevronDownIcon,
  XMarkIcon,
} from "@heroicons/react/20/solid";
import { IperfServerModal } from "./IperfServerModal";
import { getApiUrl } from "@/utils/baseUrl";
import { showToast } from "@/components/common/Toast";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/input";

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
  const getInitialDisplayCount = () => {
    if (typeof window !== "undefined") {
      return window.innerWidth >= 1024 ? 6 : 3;
    }
    return 3;
  };

  const [displayCount, setDisplayCount] = useState(getInitialDisplayCount);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterCountry, setFilterCountry] = useState("");
  const [iperfSearchTerm, setIperfSearchTerm] = useState("");
  const [addServerModalOpen, setAddServerModalOpen] = useState(false);
  const [iperfDisplayCount, setIperfDisplayCount] = useState(
    getInitialDisplayCount
  );
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
  const [isOpen, setIsOpen] = useState(() => {
    const saved = localStorage.getItem("server-list-open");
    return saved === null ? true : saved === "true";
  });

  // Persist server list open state to localStorage
  useEffect(() => {
    localStorage.setItem("server-list-open", isOpen.toString());
  }, [isOpen]);

  // Handle window resize for responsive display counts
  useEffect(() => {
    const handleResize = () => {
      const newDisplayCount = window.innerWidth >= 1024 ? 6 : 3;
      setDisplayCount(newDisplayCount);
      setIperfDisplayCount(newDisplayCount);
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  // Get unique countries for filter dropdown
  const countries = useMemo(() => {
    const uniqueCountries = new Set(servers.map((server) => server.country));
    return Array.from(uniqueCountries).sort();
  }, [servers]);


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
      showToast(`Server "${name}" added successfully`, "success");
    } catch (error) {
      console.error("Failed to save server:", error);
      showToast(
        error instanceof Error ? error.message : "Failed to save iperf server",
        "error"
      );
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
      showToast("Server deleted successfully", "success");
    } catch (error) {
      console.error("Failed to delete server:", error);
      showToast(
        error instanceof Error ? error.message : "Failed to delete iperf server",
        "error"
      );
    }
  };

  // Fetch saved iperf servers when component mounts or iperf mode changes
  useEffect(() => {
    if (testType === "iperf") {
      fetchSavedIperfServers().catch((error) => {
        console.error("Failed to fetch iperf servers:", error);
        showToast("Failed to load iperf servers", "error", {
          description: error instanceof Error ? error.message : "Unknown error",
        });
      });
    }
  }, [testType]); // Re-run when useIperf changes

  // Update filterCountry logic to handle select component values
  const filteredServersWithSelect = useMemo(() => {
    const filtered = servers.filter((server) => {
      const matchesSearch =
        searchTerm === "" ||
        server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.sponsor.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.country.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesCountry =
        filterCountry === "" || filterCountry === "all-countries" || server.country === filterCountry;

      return matchesSearch && matchesCountry;
    });

    return filtered.sort((a, b) => a.distance - b.distance);
  }, [servers, searchTerm, filterCountry]);

  return (
    <Collapsible
      defaultOpen={isOpen}
      open={isOpen}
      onOpenChange={setIsOpen}
    >
      <div className="flex flex-col h-full">
        <CollapsibleTrigger
          className={`flex justify-between items-center w-full px-4 py-2 bg-gray-50/95 dark:bg-gray-850/95 ${
            isOpen ? "rounded-t-xl" : "rounded-xl"
          } shadow-lg border border-gray-200 dark:border-gray-800 ${
            isOpen ? "border-b-0" : ""
          } text-left`}
        >
          <div className="flex flex-col">
            <h2 className="text-gray-900 dark:text-white text-xl font-semibold p-1 select-none">
              Server Selection
            </h2>
            <p className="text-gray-600 dark:text-gray-400 text-sm pl-1 pb-1">
              Choose between speedtest.net, iperf3 or librespeed servers
            </p>
          </div>
          <div className="flex items-center gap-2">
            <ChevronDownIcon
              className={`${
                isOpen ? "transform rotate-180" : ""
              } w-5 h-5 text-gray-600 dark:text-gray-400 transition-transform duration-200`}
            />
          </div>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="bg-gray-50/95 dark:bg-gray-850/95 px-4 pt-2 rounded-b-xl shadow-lg flex-1 border border-t-0 border-gray-200 dark:border-gray-800">
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
                  <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4 mb-4">
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
                        onValueChange={(value) => handleTestTypeChange(value as "speedtest" | "iperf" | "librespeed")}
                        className="flex items-center gap-2 sm:gap-4"
                      >
                        <RadioOption value="speedtest">Speedtest</RadioOption>
                        <RadioOption value="iperf">iperf3</RadioOption>
                        <RadioOption value="librespeed">Librespeed</RadioOption>
                      </RadioGroup>
                    </div>

                    {/* Run Test Button */}
                    <Button
                      onClick={onRunTest}
                      disabled={isLoading || selectedServers.length === 0}
                      className="w-full sm:w-auto"
                    >
                      Run
                    </Button>
                  </div>

                  {testType === "iperf" && (
                    <div className="flex flex-col gap-4 mb-4">
                      {/* Search Input and Add Button for iperf3 servers */}
                      <div className="flex flex-col gap-4">
                        <div className="flex gap-2">
                          <div className="flex-1">
                            <Input
                              type="text"
                              placeholder="Search saved servers..."
                              value={iperfSearchTerm}
                              onChange={(e) =>
                                setIperfSearchTerm(e.target.value)
                              }
                            />
                          </div>
                          <button
                            onClick={() => setAddServerModalOpen(true)}
                            className="px-3 py-2 bg-gray-100 dark:bg-gray-800/50 hover:bg-gray-200 dark:hover:bg-gray-700/50 text-gray-700 dark:text-gray-300 rounded-lg transition-colors border border-gray-300 dark:border-gray-900 hover:border-gray-400 dark:hover:border-gray-700 shadow-md text-sm"
                            title="Add new iperf3 server"
                          >
                            + Add
                          </button>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Server Search and Filter Controls */}
                  {(testType === "speedtest" || testType === "librespeed") && (
                    <div className="flex flex-col md:flex-row gap-4 mb-4">
                      {/* Search Input */}
                      <div className="flex-1">
                        <Input
                          type="text"
                          placeholder="Search servers..."
                          value={searchTerm}
                          onChange={(e) => setSearchTerm(e.target.value)}
                        />
                      </div>

                      {/* Country Filter */}
                      <Select
                        value={filterCountry || "all-countries"}
                        onValueChange={(value) => setFilterCountry(value === "all-countries" ? "" : value)}
                      >
                        <SelectTrigger className="min-w-[160px]">
                          <SelectValue placeholder="All Countries" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all-countries">
                            All Countries
                          </SelectItem>
                          {countries.map((country) => (
                            <SelectItem key={country} value={country}>
                              {country}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  {/* Server Grid */}
                  {testType === "iperf" ? (
                    <>
                      {filteredIperfServers.length === 0 ? (
                        <div className="flex flex-col items-center justify-center py-12 px-4">
                          <div className="text-center max-w-md">
                            <div className="text-gray-600 dark:text-gray-400 text-lg mb-2">🔧</div>
                            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-300 mb-2">
                              No iperf3 servers found
                            </h3>
                            <p className="text-gray-600 dark:text-gray-400 text-sm mb-4">
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
                                  <div
                                    onClick={() =>
                                      handleServerSelect(iperfServer)
                                    }
                                    className={`w-full p-4 rounded-lg text-left transition-colors relative cursor-pointer ${
                                      selectedServers.some(
                                        (s) => s.id === iperfServer.id
                                      )
                                        ? "bg-blue-100/50 dark:bg-blue-500/10 border-blue-400/50 shadow-lg"
                                        : "bg-gray-100/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900 hover:bg-gray-200/50 dark:hover:bg-gray-800 shadow-lg"
                                    } border`}
                                  >
                                    <div className="flex flex-col gap-1 pr-8">
                                      <span className="text-blue-600 dark:text-blue-300 font-medium truncate">
                                        {server.name}
                                      </span>
                                      <span className="text-gray-600 dark:text-gray-400 text-sm">
                                        iperf3 Server
                                        <span
                                          className="block truncate text-xs text-gray-500 dark:text-gray-500"
                                          title={`${server.host}:${server.port}`}
                                        >
                                          {server.host}:{server.port}
                                        </span>
                                      </span>
                                      <span className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                                        Custom Server
                                      </span>
                                    </div>
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        setServerToDelete(server.id);
                                        setDeleteModalOpen(true);
                                      }}
                                      className="absolute top-2 right-2 text-gray-600 dark:text-gray-400 p-1 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 rounded-md hover:bg-red-100/50 dark:hover:bg-red-900/50 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                                      title="Delete server"
                                    >
                                      <XMarkIcon className="h-4 w-4" />
                                    </button>
                                  </div>
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
                            className="px-4 py-2 bg-gray-200/30 dark:bg-gray-800/30 border border-gray-300/50 dark:border-gray-900/50 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800/50 transition-colors"
                          >
                            Load More
                          </button>
                        </div>
                      )}
                    </>
                  ) : (
                    <>
                      {filteredServersWithSelect.length === 0 &&
                      testType === "librespeed" ? (
                        <div className="flex flex-col items-center justify-center py-12 px-4">
                          <div className="text-center max-w-md">
                            <div className="text-gray-600 dark:text-gray-400 text-lg mb-2">📡</div>
                            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-300 mb-2">
                              No Librespeed servers found
                            </h3>
                            <p className="text-gray-600 dark:text-gray-400 text-sm mb-4">
                              The librespeed-servers.json file was not found.
                              Please create this file in the same directory as
                              your config.toml to run librespeed tests.
                            </p>
                            <div className="text-xs text-gray-600 dark:text-gray-500 bg-gray-100/30 dark:bg-gray-800/30 rounded-lg p-3 border border-gray-300 dark:border-gray-900">
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
                          {filteredServersWithSelect
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
                                        ? "bg-blue-100/50 dark:bg-blue-500/10 border-blue-400/50 shadow-lg"
                                        : "bg-gray-100/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-900 hover:bg-gray-200/50 dark:hover:bg-gray-800 shadow-lg"
                                    } border`}
                                  >
                                    <div className="flex flex-col gap-1">
                                      <span className="text-blue-600 dark:text-blue-300 font-medium truncate">
                                        {server.sponsor}
                                      </span>
                                      <span className="text-gray-600 dark:text-gray-400 text-sm">
                                        {server.name}
                                        <span
                                          className="block truncate text-xs text-gray-500 dark:text-gray-500"
                                          title={server.host}
                                        >
                                          {server.host}
                                        </span>
                                      </span>
                                      <span className="text-gray-600 dark:text-gray-400 text-sm mt-1">
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
                    filteredServersWithSelect.length > displayCount && (
                      <div className="flex justify-center mt-6">
                        <button
                          onClick={() => setDisplayCount((prev) => prev + 6)}
                          className="px-4 py-2 bg-gray-200/30 dark:bg-gray-800/30 border border-gray-300/50 dark:border-gray-900/50 text-gray-600/50 dark:text-gray-300/50 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-300/50 dark:hover:bg-gray-800/50 transition-colors"
                        >
                          Load More
                        </button>
                      </div>
                    )}
                </motion.div>
          </div>
        </CollapsibleContent>
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
    </Collapsible>
  );
};

const RadioOption: React.FC<{
  value: "speedtest" | "iperf" | "librespeed";
  children: React.ReactNode;
}> = ({ value, children }) => (
  <div className="flex items-center gap-2">
    <RadioGroupItem value={value} id={value} />
    <Label
      htmlFor={value}
      className="cursor-pointer px-3 py-1 rounded-lg transition-colors hover:bg-gray-200 dark:hover:bg-gray-800 text-xs sm:text-sm text-gray-600 dark:text-gray-400"
    >
      {children}
    </Label>
  </div>
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
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="w-full max-w-md">
        <DialogHeader>
          <DialogTitle>Add iperf3 Server</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="serverName">
              Server Name
            </Label>
            <Input
              type="text"
              id="serverName"
              value={serverName}
              onChange={(e) => setServerName(e.target.value)}
              placeholder="Enter a name for this server"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="serverHost">
              Server Host
            </Label>
            <Input
              type="text"
              id="serverHost"
              value={serverHost}
              onChange={(e) => setServerHost(e.target.value)}
              placeholder="iperf.example.com"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="serverPort">
              Server Port
            </Label>
            <Input
              type="number"
              id="serverPort"
              value={serverPort}
              onChange={(e) => setServerPort(e.target.value)}
              placeholder="5201"
            />
          </div>
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button
            type="button"
            variant="ghost"
            onClick={handleClose}
          >
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!serverName.trim() || !serverHost.trim()}
            onClick={handleConfirm}
          >
            Add Server
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
