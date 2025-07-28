/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

"use client";

import * as React from "react";
import * as TabsPrimitive from "@radix-ui/react-tabs";

import { cn } from "@/lib/utils";

function Tabs({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      className={cn("flex flex-col gap-4", className)}
      {...props}
    />
  );
}

function TabsList({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      className={cn(
        "inline-flex items-center justify-start",
        "border-b border-gray-200 dark:border-gray-800",
        "gap-6 sm:gap-8",
        className
      )}
      {...props}
    />
  );
}

function TabsTrigger({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      className={cn(
        "relative inline-flex items-center justify-center whitespace-nowrap",
        "pb-3 px-1",
        "text-sm font-medium transition-all duration-200",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2",
        "disabled:pointer-events-none disabled:opacity-50",
        // Bottom border indicator
        "border-b-2 -mb-[2px]",
        // Active state - blue text with blue underline
        "data-[state=active]:text-blue-600 data-[state=active]:dark:text-blue-400",
        "data-[state=active]:border-blue-600 data-[state=active]:dark:border-blue-400",
        // Inactive state - gray text with transparent border
        "data-[state=inactive]:text-gray-500 data-[state=inactive]:dark:text-gray-400",
        "data-[state=inactive]:border-transparent",
        "data-[state=inactive]:hover:text-gray-700 data-[state=inactive]:dark:hover:text-gray-300",
        "data-[state=inactive]:hover:border-gray-300 data-[state=inactive]:dark:hover:border-gray-700",
        className
      )}
      {...props}
    />
  );
}

function TabsContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cn(
        "flex-1 outline-none",
        "focus-visible:outline-none",
        className
      )}
      {...props}
    />
  );
}

export { Tabs, TabsList, TabsTrigger, TabsContent };
