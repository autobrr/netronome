import React, { useState, useMemo } from "react";
import { motion } from "motion/react";
import { Switch, Field, Label } from "@headlessui/react";
import { Server } from "../../types/types";

interface ServerListProps {
  servers: Server[];
  selectedServers: Server[];
  onSelect: (server: Server) => void;
  multiSelect: boolean;
  onMultiSelectChange: (enabled: boolean) => void;
}

export const ServerList: React.FC<ServerListProps> = ({
  servers,
  selectedServers,
  onSelect,
  multiSelect,
  onMultiSelectChange,
}) => {
  const [displayCount, setDisplayCount] = useState(6);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterCountry, setFilterCountry] = useState("");

  // Get unique countries for filter dropdown
  const countries = useMemo(() => {
    const uniqueCountries = new Set(servers.map((server) => server.country));
    return Array.from(uniqueCountries).sort();
  }, [servers]);

  // Filter and sort servers
  const filteredServers = useMemo(() => {
    return servers.filter((server) => {
      const matchesSearch =
        searchTerm === "" ||
        server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.sponsor.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.country.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesCountry =
        filterCountry === "" || server.country === filterCountry;

      return matchesSearch && matchesCountry;
    });
  }, [servers, searchTerm, filterCountry]);

  return (
    <motion.div
      className="mt-1 select-none pointer-events-none server-list-animate"
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{
        duration: 0.5,
        type: "spring",
        stiffness: 300,
        damping: 20,
      }}
      onAnimationComplete={() => {
        const element = document.querySelector(".server-list-animate");
        if (element) {
          element.classList.remove("select-none", "pointer-events-none");
        }
      }}
    >
      {/* Multi-select Toggle */}
      <div className="flex items-center justify-end mb-4">
        <Field>
          <div className="flex items-center">
            <Label className="mr-3 text-sm text-gray-400">Multi-select</Label>
            <Switch
              checked={multiSelect}
              onChange={onMultiSelectChange}
              className={`${
                multiSelect ? "bg-blue-500" : "bg-gray-700"
              } relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none`}
            >
              <span
                className={`${
                  multiSelect ? "translate-x-6" : "translate-x-1"
                } inline-block h-4 w-4 transform rounded-full bg-white transition-transform`}
              />
            </Switch>
          </div>
        </Field>
      </div>

      <div className="flex flex-col md:flex-row gap-4 mb-4">
        {/* Search Input */}
        <div className="flex-1">
          <input
            type="text"
            placeholder="Search servers..."
            className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>

        {/* Country Filter */}
        <select
          className="px-4 py-2 bg-gray-800/50 border border-gray-900 rounded-lg focus:outline-none text-gray-300"
          value={filterCountry}
          onChange={(e) => setFilterCountry(e.target.value)}
        >
          <option value="">All Countries</option>
          {countries.map((country) => (
            <option key={country} value={country}>
              {country}
            </option>
          ))}
        </select>
      </div>

      {/* Server Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredServers.slice(0, displayCount).map((server) => (
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
                  ? "bg-blue-500/20 border-blue-500"
                  : "bg-gray-800/50 border-gray-900 hover:bg-gray-800"
              } border`}
            >
              <div className="flex flex-col gap-1">
                <span className="text-blue-400 font-medium">
                  {server.sponsor}
                </span>
                <span className="text-gray-400 text-sm">{server.name}</span>
                <span className="text-gray-400 text-sm">{server.country}</span>
              </div>
            </button>
          </motion.div>
        ))}
      </div>

      {/* Load More Button */}
      {filteredServers.length > displayCount && (
        <div className="flex justify-center mt-4">
          <button
            onClick={() => setDisplayCount((prev) => prev + 6)}
            className="px-4 py-2 mb-4 bg-gray-800/50 text-white rounded-lg hover:bg-gray-800 transition-colors"
          >
            Load More
          </button>
        </div>
      )}
    </motion.div>
  );
};
