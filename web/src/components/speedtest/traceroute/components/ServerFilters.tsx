/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { ChevronUpDownIcon } from "@heroicons/react/24/solid";
import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
  Transition,
} from "@headlessui/react";
import { STYLES } from "../constants/tracerouteConstants";

interface ServerTypeOption {
  value: string;
  label: string;
}

interface ServerFiltersProps {
  searchTerm: string;
  filterType: string;
  serverTypeOptions: ServerTypeOption[];
  onSearchChange: (value: string) => void;
  onFilterTypeChange: (value: string) => void;
  disabled?: boolean;
}

export const ServerFilters: React.FC<ServerFiltersProps> = ({
  searchTerm,
  filterType,
  serverTypeOptions,
  onSearchChange,
  onFilterTypeChange,
  disabled = false,
}) => {
  return (
    <div className="flex flex-col md:flex-row gap-4 mb-4">
      {/* Search Input */}
      <div className="flex-1">
        <input
          type="text"
          placeholder="Search servers..."
          className={STYLES.input}
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
          disabled={disabled}
        />
      </div>

      {/* Server Type Filter */}
      <Listbox
        value={filterType}
        onChange={onFilterTypeChange}
        disabled={disabled}
      >
        <div className="relative min-w-[160px]">
          <ListboxButton className={`relative w-full ${STYLES.input}`}>
            <span className="block truncate">
              {serverTypeOptions.find((type) => type.value === filterType)
                ?.label || "All Types"}
            </span>
            <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
              <ChevronUpDownIcon
                className="h-5 w-5 text-gray-600 dark:text-gray-400"
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
            <ListboxOptions className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-800 py-1 shadow-lg focus:outline-none">
              {serverTypeOptions.map((type) => (
                <ListboxOption
                  key={type.value}
                  value={type.value}
                  className={({ focus }) =>
                    `relative cursor-pointer select-none py-2 px-4 ${
                      focus
                        ? "bg-blue-500/10 text-blue-600 dark:text-blue-200"
                        : "text-gray-700 dark:text-gray-300"
                    }`
                  }
                >
                  {type.label}
                </ListboxOption>
              ))}
            </ListboxOptions>
          </Transition>
        </div>
      </Listbox>
    </div>
  );
};
