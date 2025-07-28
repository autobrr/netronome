/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

"use client";

import React, { useState, useEffect } from "react";
import { motion } from "motion/react";
import { SpeedTestResult, TimeRange } from "@/types/types";
import { SpeedHistoryChart } from "./SpeedHistoryChart";
import { MetricCard } from "@/components/common/MetricCard";
import { FeaturedMonitorWidget } from "@/components/monitor/FeaturedMonitorWidget";
import { FaWaveSquare, FaShare, FaArrowDown, FaArrowUp, FaGripVertical } from "react-icons/fa";
import { IoIosPulse } from "react-icons/io";
import { ChevronDownIcon } from "@heroicons/react/20/solid";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { DataTable } from "@/components/ui/data-table";
import { speedTestColumns, speedTestMobileColumns } from "./columns";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

interface DashboardTabProps {
  latestTest: SpeedTestResult | null;
  tests: SpeedTestResult[];
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
  isPublic?: boolean;
  hasAnyTests?: boolean;
  onShareClick?: () => void;
  onNavigateToSpeedTest?: () => void;
  onNavigateToVnstat?: (agentId?: number) => void;
}

interface SortableItemProps {
  id: string;
  children: React.ReactNode;
  dragHandleClassName?: string;
}

const SortableItem: React.FC<SortableItemProps> = ({ id, children, dragHandleClassName = "drag-handle" }) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
    setActivatorNodeRef,
  } = useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      {React.cloneElement(children as React.ReactElement, {
        dragHandleRef: setActivatorNodeRef,
        dragHandleListeners: listeners,
        dragHandleClassName,
      })}
    </div>
  );
};

// Wrapper component for SpeedHistoryChart with drag handle
interface DraggableSpeedHistoryChartProps {
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
  isPublic?: boolean;
  hasAnyTests?: boolean;
  hasCurrentRangeTests?: boolean;
  dragHandleRef?: (node: HTMLElement | null) => void;
  dragHandleListeners?: any;
}

const DraggableSpeedHistoryChart: React.FC<DraggableSpeedHistoryChartProps> = ({
  dragHandleRef,
  dragHandleListeners,
  ...props
}) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 20 }}
      transition={{ duration: 0.5 }}
    >
      <SpeedHistoryChart
        {...props}
        showDragHandle={true}
        dragHandleRef={dragHandleRef}
        dragHandleListeners={dragHandleListeners}
      />
    </motion.div>
  );
};

export const DashboardTab: React.FC<DashboardTabProps> = ({
  latestTest,
  tests,
  timeRange,
  onTimeRangeChange,
  isPublic = false,
  hasAnyTests = false,
  onShareClick,
  onNavigateToSpeedTest,
  onNavigateToVnstat,
}) => {
  const [displayCount, setDisplayCount] = useState(5);
  const [isRecentTestsOpen, setIsRecentTestsOpen] = useState(() => {
    const saved = localStorage.getItem("recent-tests-open");
    return saved === null ? true : saved === "true";
  });

  // Initialize section order from localStorage or default
  const [sectionOrder, setSectionOrder] = useState<string[]>(() => {
    const saved = localStorage.getItem("dashboard-section-order");
    return saved ? JSON.parse(saved) : ["history", "recent"];
  });

  // Initialize drag sensors
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Persist recent tests open state to localStorage
  useEffect(() => {
    localStorage.setItem("recent-tests-open", isRecentTestsOpen.toString());
  }, [isRecentTestsOpen]);

  // Persist section order to localStorage
  useEffect(() => {
    localStorage.setItem("dashboard-section-order", JSON.stringify(sectionOrder));
  }, [sectionOrder]);

  const displayedTests = tests.slice(0, displayCount);

  const calculateAverage = (field: keyof SpeedTestResult): string => {
    if (tests.length === 0) return "N/A";

    const validValues = tests
      .map((test) => {
        const value = test[field];
        if (typeof value === "string") {
          return parseFloat(value.replace("ms", ""));
        }
        return Number(value);
      })
      .filter((value) => !isNaN(value));

    if (validValues.length === 0) return "N/A";

    const avg =
      validValues.reduce((sum, value) => sum + value, 0) / validValues.length;
    return avg.toFixed(2);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      setSectionOrder((items) => {
        const oldIndex = items.indexOf(active.id as string);
        const newIndex = items.indexOf(over.id as string);
        return arrayMove(items, oldIndex, newIndex);
      });
    }
  };

  return (
    <div className="space-y-6">
      {/* No History Available message */}
      {!hasAnyTests && (
        <div className="max-w-xl mx-auto bg-gray-50/95 dark:bg-gray-850/95 p-4 sm:p-6 rounded-xl shadow-lg border border-gray-200 dark:border-gray-900">
          <div className="text-center space-y-3 sm:space-y-4">
            <div>
              <h2 className="text-gray-900 dark:text-white text-lg sm:text-xl font-semibold mb-1 sm:mb-2">
                No History Available
              </h2>
              <p className="text-gray-600 dark:text-gray-400 text-sm sm:text-base">
                Start monitoring your network performance
              </p>
            </div>

            {/* Empty Chart Visualization - more compact on mobile */}
            <div className="my-4 sm:my-6">
              <div className="flex items-end justify-center gap-1 sm:gap-2 h-16 sm:h-24">
                {[40, 60, 35, 70, 45, 55, 65].map((height, i) => (
                  <div
                    key={i}
                    className="w-6 sm:w-8 bg-gray-300/50 dark:bg-gray-700/50 rounded-t-sm transition-all duration-500"
                    style={{ height: `${height}%`, opacity: 0.3 + i * 0.1 }}
                  />
                ))}
              </div>
              <div className="border-t border-gray-300/50 dark:border-gray-700/50 mt-2" />
            </div>

            <div className="max-w-md mx-auto">
              <p className="text-gray-600 dark:text-gray-400 text-sm">
                Go to the{" "}
                <button
                  onClick={onNavigateToSpeedTest}
                  className="inline-flex items-center mx-1 px-3 py-2 sm:px-2 sm:py-1 min-h-[44px] sm:min-h-0 rounded-lg transition-colors text-xs bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 hover:bg-emerald-500/20 touch-manipulation"
                  disabled={!onNavigateToSpeedTest}
                >
                  Speed Test tab
                </button>{" "}
                to run manual tests or set up automated schedules
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Featured Vnstat Widget - only show if callback is provided and not public */}
      {onNavigateToVnstat && !isPublic && (
        <FeaturedMonitorWidget onNavigateToMonitor={onNavigateToVnstat} />
      )}

      {/* Latest Results */}
      {hasAnyTests && latestTest && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 20 }}
          transition={{ duration: 0.5 }}
          className="mb-6"
        >
          <h2 className="text-gray-900 dark:text-white text-xl ml-1 font-semibold">
            Latest Run
          </h2>
          <div className="flex justify-between ml-1 items-center text-gray-600 dark:text-gray-400 text-sm mb-4">
            <div>
              Last test run:{" "}
              {latestTest?.createdAt
                ? new Date(latestTest.createdAt).toLocaleString(undefined, {
                    dateStyle: "short",
                    timeStyle: "short",
                  })
                : "N/A"}
            </div>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 mb-6 cursor-default relative">
            <MetricCard
              icon={<IoIosPulse className="w-5 h-5 text-amber-500" />}
              title="Latency"
              value={parseFloat(latestTest.latency).toFixed(2)}
              unit="ms"
              average={calculateAverage("latency")}
            />
            <MetricCard
              icon={<FaArrowDown className="w-5 h-5 text-blue-500" />}
              title="Download"
              value={latestTest.downloadSpeed.toFixed(2)}
              unit="Mbps"
              average={calculateAverage("downloadSpeed")}
            />
            <MetricCard
              icon={<FaArrowUp className="w-5 h-5 text-emerald-500" />}
              title="Upload"
              value={latestTest.uploadSpeed.toFixed(2)}
              unit="Mbps"
              average={calculateAverage("uploadSpeed")}
            />
            <MetricCard
              icon={<FaWaveSquare className="w-5 h-5 text-purple-400" />}
              title="Jitter"
              value={latestTest.jitter?.toFixed(2) ?? "N/A"}
              unit="ms"
              average={
                latestTest.jitter ? calculateAverage("jitter") : undefined
              }
            />

            {/* Floating Share Button positioned on the grid */}
            {!isPublic && onShareClick && (
              <motion.button
                onClick={onShareClick}
                className="absolute top-2 right-2 sm:top-3 sm:right-3 p-3 sm:p-2 min-w-[44px] min-h-[44px] sm:min-w-0 sm:min-h-0 bg-blue-500/20 hover:bg-blue-500/30 border border-blue-500/30 hover:border-blue-500/50 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 rounded-lg transition-all duration-200 backdrop-blur-sm z-10 opacity-80 hover:opacity-100 touch-manipulation flex items-center justify-center"
                aria-label="Share public speed test page"
              >
                <FaShare className="w-3.5 h-3.5 sm:w-2.5 sm:h-2.5" />
              </motion.button>
            )}
          </div>
        </motion.div>
      )}

      {/* Draggable Sections */}
      {hasAnyTests && (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={sectionOrder}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-6">
              {sectionOrder.map((sectionId) => {
                if (sectionId === "history") {
                  return (
                    <SortableItem key="history" id="history">
                      <DraggableSpeedHistoryChart
                        timeRange={timeRange}
                        onTimeRangeChange={onTimeRangeChange}
                        isPublic={isPublic}
                        hasAnyTests={hasAnyTests}
                        hasCurrentRangeTests={tests.length > 0}
                      />
                    </SortableItem>
                  );
                } else if (sectionId === "recent" && tests.length > 0) {
                  return (
                    <SortableItem key="recent" id="recent">
                      <DraggableRecentSpeedtests
                        tests={tests}
                        displayedTests={displayedTests}
                        displayCount={displayCount}
                        setDisplayCount={setDisplayCount}
                        isRecentTestsOpen={isRecentTestsOpen}
                        setIsRecentTestsOpen={setIsRecentTestsOpen}
                      />
                    </SortableItem>
                  );
                }
                return null;
              })}
            </div>
          </SortableContext>
        </DndContext>
      )}
    </div>
  );
};

// Wrapper component for Recent Speedtests with drag handle
interface DraggableRecentSpeedtestsProps {
  tests: SpeedTestResult[];
  displayedTests: SpeedTestResult[];
  displayCount: number;
  setDisplayCount: (count: number | ((prev: number) => number)) => void;
  isRecentTestsOpen: boolean;
  setIsRecentTestsOpen: (open: boolean) => void;
  dragHandleRef?: (node: HTMLElement | null) => void;
  dragHandleListeners?: any;
}

const DraggableRecentSpeedtests: React.FC<DraggableRecentSpeedtestsProps> = ({
  tests,
  displayedTests,
  displayCount,
  setDisplayCount,
  isRecentTestsOpen,
  setIsRecentTestsOpen,
  dragHandleRef,
  dragHandleListeners,
}) => {
  return (
    <Collapsible
      open={isRecentTestsOpen}
      onOpenChange={setIsRecentTestsOpen}
      className="flex flex-col h-full"
    >
      <CollapsibleTrigger
        className={cn(
          "flex justify-between items-center w-full px-4 py-3 sm:py-2 min-h-[44px] sm:min-h-0 bg-gray-50/95 dark:bg-gray-850/95",
          isRecentTestsOpen ? "rounded-t-xl" : "rounded-xl",
          "shadow-lg border border-gray-200 dark:border-gray-800",
          isRecentTestsOpen ? "border-b-0" : "",
          "text-left transition-all duration-200 hover:bg-gray-100/95 dark:hover:bg-gray-800/95 touch-manipulation"
        )}
      >
        <div className="flex items-center gap-2">
          <div
            ref={dragHandleRef}
            {...dragHandleListeners}
            className="cursor-grab active:cursor-grabbing touch-none p-1 -m-1"
          >
            <FaGripVertical className="w-4 h-4 text-gray-400 dark:text-gray-600" />
          </div>
          <h2 className="text-gray-900 dark:text-white text-lg sm:text-xl font-semibold p-1 select-none">
            Recent Speedtests
          </h2>
        </div>
        <div className="p-1 -m-1">
          <ChevronDownIcon
            className={cn(
              isRecentTestsOpen ? "transform rotate-180" : "",
              "w-5 h-5 text-gray-600 dark:text-gray-400 transition-transform duration-200"
            )}
          />
        </div>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{
            duration: 0.5,
            type: "spring",
            stiffness: 300,
            damping: 20,
          }}
          className="bg-gray-50/95 dark:bg-gray-850/95 px-3 sm:px-4 pt-3 pb-6 rounded-b-xl shadow-lg flex-1 border border-t-0 border-gray-200 dark:border-gray-800"
        >
          {/* Desktop Table View */}
          <div className="hidden md:block">
            <DataTable
              columns={speedTestColumns}
              data={displayedTests}
              showPagination={false}
              showColumnVisibility={true}
              showRowSelection={false}
              filterColumn="serverName"
              filterPlaceholder="Filter by server..."
              className="-mt-4"
            />
          </div>

          {/* Mobile Card View */}
          <div className="md:hidden">
            <DataTable
              columns={speedTestMobileColumns}
              data={displayedTests}
              showPagination={false}
              showColumnVisibility={false}
              showRowSelection={false}
              showHeaders={false}
              className="-mt-4"
              tableClassName="border-0"
            />
          </div>
          {/* Test Count and Load More */}
          {tests.length > 5 && (
            <div className="mt-4 space-y-3">
              {/* Test Count */}
              <div className="text-center">
                <span className="text-gray-500 dark:text-gray-500 text-xs sm:text-sm">
                  Showing {displayedTests.length} of {tests.length} tests
                </span>
              </div>

              {/* Load More / Show Less Buttons */}
              <div className="flex flex-col sm:flex-row items-center justify-center gap-2 sm:gap-3">
                {tests.length > displayCount && (
                  <button
                    onClick={() => setDisplayCount((prev) => prev + 5)}
                    className="inline-flex items-center justify-center w-full sm:w-auto px-4 py-3 sm:py-2 min-h-[44px] sm:min-h-0 bg-blue-600/10 hover:bg-blue-600/20 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 rounded-lg transition-colors duration-200 text-sm font-medium touch-manipulation"
                  >
                    Load {Math.min(5, tests.length - displayCount)} more
                    <span className="ml-2">↓</span>
                  </button>
                )}

                {displayCount > 5 && (
                  <button
                    onClick={() => setDisplayCount(5)}
                    className="inline-flex items-center justify-center w-full sm:w-auto px-4 py-3 sm:py-2 min-h-[44px] sm:min-h-0 bg-gray-600/10 hover:bg-gray-600/20 text-gray-600 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg transition-colors duration-200 text-sm font-medium touch-manipulation"
                  >
                    Show less
                    <span className="ml-2">↑</span>
                  </button>
                )}
              </div>
            </div>
          )}
        </motion.div>
      </CollapsibleContent>
    </Collapsible>
  );
};
