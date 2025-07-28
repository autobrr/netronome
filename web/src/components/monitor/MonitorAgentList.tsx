/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import {
  PencilIcon,
  TrashIcon,
  SparklesIcon,
  ArrowDownIcon,
  ArrowUpIcon,
  LockClosedIcon,
  LockOpenIcon,
} from "@heroicons/react/24/solid";
import { SparklesIcon as SparklesIconSolid } from "@heroicons/react/24/solid";
import { MonitorAgent } from "@/api/monitor";
import { AgentIcon } from "@/utils/agentIcons";
import { useMonitorAgent } from "@/hooks/useMonitorAgent";
import { TailscaleLogo } from "../icons/TailscaleLogo";
import { DeleteConfirmationDialog } from "@/components/common/DeleteConfirmationDialog";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { DataTable, createSortableHeader } from "@/components/ui/data-table";
import { cn } from "@/lib/utils";

interface MonitorAgentListProps {
  agents: MonitorAgent[];
  selectedAgent: MonitorAgent | null;
  onSelectAgent: (agent: MonitorAgent) => void;
  onEditAgent: (agent: MonitorAgent) => void;
  onDeleteAgent: (id: number) => void;
  isLoading: boolean;
}

// Extended interface for table data
interface AgentTableData extends MonitorAgent {
  status?: {
    connected: boolean;
    liveData?: {
      rx: { ratestring: string };
      tx: { ratestring: string };
    };
  };
  isFeatured: boolean;
}

export const MonitorAgentList: React.FC<MonitorAgentListProps> = ({
  agents,
  selectedAgent: _selectedAgent,
  onSelectAgent,
  onEditAgent,
  onDeleteAgent,
  isLoading,
}) => {
  // Force re-render all items when featured agents change
  const [updateKey, setUpdateKey] = React.useState(0);
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
  const [agentToDelete, setAgentToDelete] = React.useState<MonitorAgent | null>(
    null
  );

  React.useEffect(() => {
    const handleStorageChange = () => {
      setUpdateKey((prev) => prev + 1);
    };

    window.addEventListener("storage", handleStorageChange);
    window.addEventListener("featured-agents-changed", handleStorageChange);
    return () => {
      window.removeEventListener("storage", handleStorageChange);
      window.removeEventListener(
        "featured-agents-changed",
        handleStorageChange
      );
    };
  }, []);

  // Get featured agent IDs
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-monitor-agents");
      const parsed = stored ? JSON.parse(stored) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  };

  const featuredAgentIds = getFeaturedAgentIds();

  // Transform agents to table data (with updateKey to force re-render on featured changes)
  const tableData: AgentTableData[] = React.useMemo(
    () =>
      agents.map((agent) => ({
        ...agent,
        isFeatured: featuredAgentIds.includes(agent.id),
      })),
    [agents, featuredAgentIds, updateKey]
  );

  // Column definitions
  const columns: ColumnDef<AgentTableData>[] = [
    {
      accessorKey: "name",
      header: createSortableHeader("Agent"),
      cell: ({ row }) => {
        const agent = row.original;
        return (
          <div className="flex items-center gap-3">
            <AgentIcon
              name={agent.name}
              className="h-7 w-7 text-gray-400 flex-shrink-0"
            />
            <div className="font-medium text-gray-900 dark:text-white">
              {agent.name}
            </div>
          </div>
        );
      },
    },
    {
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const agent = row.original;
        return <AgentStatusCell agent={agent} />;
      },
    },
    {
      accessorKey: "url",
      header: "Connection",
      cell: ({ row }) => {
        const agent = row.original;
        return (
          <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
            {agent.isTailscale ? (
              <>
                <TailscaleLogo
                  className="h-4 w-4 flex-shrink-0"
                  title="Connected through Tailscale"
                />
                <LockClosedIcon
                  className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-500 flex-shrink-0"
                  title="Encrypted via Tailscale"
                />
              </>
            ) : agent.apiKey ? (
              <LockClosedIcon
                className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-500 flex-shrink-0"
                title="Authentication enabled"
              />
            ) : (
              <LockOpenIcon
                className="h-3.5 w-3.5 text-amber-600 dark:text-amber-500 flex-shrink-0"
                title="No authentication"
              />
            )}
            <span className="truncate max-w-xs">
              {agent.url.replace(/\/events\?stream=live-data$/, "")}
            </span>
          </div>
        );
      },
    },
    {
      id: "bandwidth",
      header: "Current Bandwidth",
      cell: ({ row }) => {
        const agent = row.original;
        return <AgentBandwidthCell agent={agent} />;
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const agent = row.original;
        return (
          <AgentActionsCell
            agent={agent}
            agents={agents}
            onEdit={() => onEditAgent(agent)}
            onDelete={() => {
              setAgentToDelete(agent);
              setDeleteDialogOpen(true);
            }}
          />
        );
      },
    },
  ];

  const [filterValue, setFilterValue] = React.useState("");

  // Filter the data based on the filter value
  const filteredData = React.useMemo(() => {
    if (!filterValue) return tableData;
    return tableData.filter((agent) =>
      agent.name.toLowerCase().includes(filterValue.toLowerCase())
    );
  }, [tableData, filterValue]);

  return (
    <>
      <Card>
        <CardHeader className="py-3 px-4">
          <div className="flex items-center justify-between gap-4">
            <CardTitle className="text-base">Netronome Agents</CardTitle>
            {!isLoading && agents.length > 0 && (
              <div className="relative w-72">
                <Input
                  placeholder="Filter agents..."
                  value={filterValue}
                  onChange={(e) => setFilterValue(e.target.value)}
                  className="h-9 pr-8"
                />
                {filterValue && (
                  <button
                    onClick={() => setFilterValue("")}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <svg
                      className="h-4 w-4"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                )}
              </div>
            )}
          </div>
        </CardHeader>

        <CardContent className="p-0 pt-0">
          {isLoading ? (
            <div className="p-8 text-center">
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Loading agents...
              </p>
            </div>
          ) : !agents || agents.length === 0 ? (
            <div className="p-8 text-center">
              <AgentIcon
                name="Server"
                className="mx-auto h-10 w-10 text-gray-400"
              />
              <p className="mt-1.5 text-sm text-gray-500 dark:text-gray-400">
                No agents configured
              </p>
              <p className="text-xs text-gray-400 dark:text-gray-500">
                Add an agent to start monitoring bandwidth
              </p>
            </div>
          ) : (
            <DataTable
              columns={columns}
              data={filteredData}
              showPagination={filteredData.length > 10}
              showColumnVisibility={false}
              pageSize={10}
              className="px-4 pb-4"
              tableClassName=""
              noDataMessage="No agents found."
              onRowClick={(agent) => onSelectAgent(agent)}
              filterColumn={undefined} // Disable built-in filter since we moved it to header
            />
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmationDialog
        isOpen={deleteDialogOpen}
        onClose={() => {
          setDeleteDialogOpen(false);
          setAgentToDelete(null);
        }}
        onConfirm={() => {
          if (agentToDelete) {
            onDeleteAgent(agentToDelete.id);
            setDeleteDialogOpen(false);
            setAgentToDelete(null);
          }
        }}
        itemName={agentToDelete?.name || ""}
      />
    </>
  );
};

// Cell component for agent status
const AgentStatusCell: React.FC<{ agent: AgentTableData }> = ({ agent }) => {
  const { status } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeHardwareStats: false,
  });

  if (!agent.enabled) {
    return (
      <Badge variant="secondary" className="gap-1.5">
        <span className="inline-flex rounded-full h-2 w-2 bg-gray-500"></span>
        Disabled
      </Badge>
    );
  }

  if (status?.connected) {
    return (
      <Badge variant="success" className="gap-1.5">
        <span className="relative inline-flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
        </span>
        Online
      </Badge>
    );
  }

  return (
    <Badge variant="destructive" className="gap-1.5">
      <span className="inline-flex rounded-full h-2 w-2 bg-red-500"></span>
      Disconnected
    </Badge>
  );
};

// Cell component for bandwidth display
const AgentBandwidthCell: React.FC<{ agent: AgentTableData }> = ({ agent }) => {
  const { status } = useMonitorAgent({
    agent,
    includeNativeData: true,
    includeHardwareStats: false,
  });

  if (!agent.enabled || !status?.connected || !status.liveData) {
    return <span className="text-sm text-gray-400">-</span>;
  }

  return (
    <div className="flex items-center gap-4">
      <div className="flex items-center gap-1">
        <ArrowDownIcon className="h-3.5 w-3.5 text-blue-500" />
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {status.liveData.rx.ratestring}
        </span>
      </div>
      <div className="flex items-center gap-1">
        <ArrowUpIcon className="h-3.5 w-3.5 text-green-500" />
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {status.liveData.tx.ratestring}
        </span>
      </div>
    </div>
  );
};

// Cell component for actions
interface AgentActionsCellProps {
  agent: AgentTableData;
  agents: MonitorAgent[];
  onEdit: () => void;
  onDelete: () => void;
}

const AgentActionsCell: React.FC<AgentActionsCellProps> = ({
  agent,
  agents,
  onEdit,
  onDelete,
}) => {
  // Featured agents management with local state for instant feedback
  const getFeaturedAgentIds = (): number[] => {
    try {
      const stored = localStorage.getItem("netronome-featured-monitor-agents");
      const parsed = stored ? JSON.parse(stored) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  };

  const setFeaturedAgentIds = (ids: number[]) => {
    try {
      const validIds = Array.isArray(ids) ? ids : [];
      localStorage.setItem(
        "netronome-featured-monitor-agents",
        JSON.stringify(validIds)
      );
    } catch {
      console.error("Error saving featured agents to localStorage");
    }
  };

  const [isFeatured, setIsFeatured] = React.useState(() => {
    const featuredIds = getFeaturedAgentIds();
    return featuredIds.includes(agent.id);
  });

  React.useEffect(() => {
    const handleStorageChange = () => {
      const featuredIds = getFeaturedAgentIds();
      setIsFeatured(featuredIds.includes(agent.id));
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, [agent.id]);

  const handleToggleFeatured = (e: React.MouseEvent) => {
    e.stopPropagation();

    const currentFeatured = getFeaturedAgentIds();

    if (isFeatured) {
      const newFeatured = currentFeatured.filter((id) => id !== agent.id);
      setFeaturedAgentIds(newFeatured);
      setIsFeatured(false);
    } else {
      const validCurrentFeatured = Array.isArray(currentFeatured)
        ? currentFeatured.filter((id) => typeof id === "number")
        : [];

      const existingAgentIds = agents.map((a) => a.id);
      const existingFeatured = validCurrentFeatured.filter((id) =>
        existingAgentIds.includes(id)
      );

      if (existingFeatured.length !== validCurrentFeatured.length) {
        setFeaturedAgentIds(existingFeatured);
      }

      if (existingFeatured.length >= 3) {
        alert(
          "You can only feature up to 3 agents at a time. Please unfeature an agent first."
        );
        return;
      }

      if (existingFeatured.includes(agent.id)) {
        setIsFeatured(true);
        return;
      }

      const newFeatured = [...existingFeatured, agent.id];
      setFeaturedAgentIds(newFeatured);
      setIsFeatured(true);
    }

    window.dispatchEvent(new Event("storage"));
    window.dispatchEvent(new Event("featured-agents-changed"));
  };

  return (
    <div
      className="flex items-center justify-end gap-1"
      onClick={(e) => e.stopPropagation()}
    >
      <Button
        variant="secondary"
        size="sm"
        className={cn(
          "h-8 w-8 p-0",
          isFeatured
            ? "hover:bg-yellow-100 dark:hover:bg-yellow-900/30 text-yellow-500 hover:text-yellow-600 dark:text-yellow-400 dark:hover:text-yellow-500"
            : "hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        )}
        onClick={handleToggleFeatured}
        title={isFeatured ? "Remove from featured" : "Add to featured"}
      >
        {isFeatured ? (
          <SparklesIconSolid className="w-4 h-4" />
        ) : (
          <SparklesIcon className="w-4 h-4" />
        )}
      </Button>
      <Button
        variant="secondary"
        size="sm"
        className="h-8 w-8 p-0"
        onClick={(e) => {
          e.stopPropagation();
          onEdit();
        }}
        title="Edit Agent"
      >
        <PencilIcon className="w-4 h-4" />
      </Button>
      <Button
        variant="secondary"
        size="sm"
        className="h-8 w-8 p-0 hover:bg-red-100 dark:hover:bg-red-900/30 hover:text-red-600 dark:hover:text-red-400"
        onClick={(e) => {
          e.stopPropagation();
          onDelete();
        }}
        title="Delete Agent"
      >
        <TrashIcon className="w-4 h-4" />
      </Button>
    </div>
  );
};
