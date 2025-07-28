/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

"use client"

import { ColumnDef } from "@tanstack/react-table"
import { SpeedTestResult } from "@/types/types"
import { createSortableHeader, createRightAlignedSortableHeader } from "@/components/ui/data-table"
import { cn } from "@/lib/utils"

// Helper function to format speed
const formatSpeed = (speed: number) => {
  if (speed >= 1000) {
    return `${(speed / 1000).toFixed(1)} Gbps`
  }
  return `${speed.toFixed(0)} Mbps`
}

// Helper function to get test type badge styles
const getTestTypeBadgeClass = (testType: string) => {
  switch (testType) {
    case "iperf3":
      return "bg-purple-500/10 text-purple-600 dark:text-purple-400"
    case "librespeed":
      return "bg-blue-500/10 text-blue-600 dark:text-blue-400"
    default:
      return "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
  }
}

// Helper function to get test type display name
const getTestTypeDisplayName = (testType: string) => {
  switch (testType) {
    case "iperf3":
      return "iperf3"
    case "librespeed":
      return "LibreSpeed"
    default:
      return "Speedtest.net"
  }
}

export const speedTestColumns: ColumnDef<SpeedTestResult>[] = [
  {
    accessorKey: "createdAt",
    header: createSortableHeader("Date"),
    cell: ({ row }) => {
      const date = new Date(row.getValue("createdAt"))
      return (
        <span className="text-gray-700 dark:text-gray-300">
          {date.toLocaleString(undefined, {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </span>
      )
    },
    enableHiding: false, // Always show date
  },
  {
    accessorKey: "serverName",
    header: "Server",
    cell: ({ row }) => (
      <span 
        className="text-gray-700 dark:text-gray-300 truncate block max-w-[180px] font-medium"
        title={row.getValue("serverName")}
      >
        {row.getValue("serverName")}
      </span>
    ),
    enableHiding: false, // Always show server
  },
  {
    accessorKey: "testType",
    header: "Type",
    cell: ({ row }) => {
      const testType = row.getValue("testType") as string
      return (
        <span
          className={cn(
            "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium",
            getTestTypeBadgeClass(testType)
          )}
        >
          {getTestTypeDisplayName(testType)}
        </span>
      )
    },
    meta: {
      displayName: "Test Type"
    },
  },
  {
    accessorKey: "latency",
    header: createRightAlignedSortableHeader("Latency"),
    cell: ({ row }) => {
      const latency = parseFloat(row.getValue("latency"))
      return (
        <div className="text-right text-amber-600 dark:text-amber-400 font-mono font-medium">
          {latency.toFixed(1)}ms
        </div>
      )
    },
  },
  {
    accessorKey: "jitter",
    header: createRightAlignedSortableHeader("Jitter"),
    cell: ({ row }) => {
      const jitter = row.getValue("jitter") as number | null
      return (
        <div className="text-right text-purple-600 dark:text-purple-400 font-mono font-medium">
          {jitter ? `${jitter.toFixed(1)}ms` : "—"}
        </div>
      )
    },
  },
  {
    accessorKey: "downloadSpeed",
    header: createRightAlignedSortableHeader("Download"),
    cell: ({ row }) => {
      const speed = row.getValue("downloadSpeed") as number
      return (
        <div className="text-right text-blue-600 dark:text-blue-400 font-mono font-medium">
          {formatSpeed(speed)}
        </div>
      )
    },
  },
  {
    accessorKey: "uploadSpeed",
    header: createRightAlignedSortableHeader("Upload"),
    cell: ({ row }) => {
      const speed = row.getValue("uploadSpeed") as number
      return (
        <div className="text-right text-emerald-600 dark:text-emerald-400 font-mono font-medium">
          {formatSpeed(speed)}
        </div>
      )
    },
  },
]

// Mobile-friendly columns with fewer fields
export const speedTestMobileColumns: ColumnDef<SpeedTestResult>[] = [
  {
    id: "summary",
    header: "Test Summary",
    cell: ({ row }) => {
      const test = row.original
      const testType = test.testType
      
      return (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <div className="text-gray-700 dark:text-gray-300 text-sm font-medium truncate flex-1 mr-2">
              {test.serverName}
            </div>
            <span
              className={cn(
                "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium flex-shrink-0",
                getTestTypeBadgeClass(testType)
              )}
            >
              {getTestTypeDisplayName(testType)}
            </span>
          </div>
          <div className="text-gray-600 dark:text-gray-400 text-xs">
            {new Date(test.createdAt).toLocaleString(undefined, {
              month: "short",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            })}
          </div>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Latency:</span>
              <span className="text-amber-600 dark:text-amber-400 font-mono font-medium">
                {parseFloat(test.latency).toFixed(1)}ms
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Jitter:</span>
              <span className="text-purple-600 dark:text-purple-400 font-mono font-medium">
                {test.jitter ? `${test.jitter.toFixed(1)}ms` : "—"}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Download:</span>
              <span className="text-blue-600 dark:text-blue-400 font-mono font-medium">
                {formatSpeed(test.downloadSpeed)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Upload:</span>
              <span className="text-emerald-600 dark:text-emerald-400 font-mono font-medium">
                {formatSpeed(test.uploadSpeed)}
              </span>
            </div>
          </div>
        </div>
      )
    },
  },
]