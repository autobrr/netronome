/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex items-center px-3 py-1 rounded-full text-xs font-medium transition-colors",
  {
    variants: {
      variant: {
        default:
          "bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30",
        secondary:
          "bg-gray-200/50 dark:bg-gray-850/95 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-900",
        destructive:
          "bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30",
        success:
          "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30",
        warning:
          "bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30",
        purple:
          "bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/30",
        outline:
          "border border-gray-200 dark:border-gray-900 text-gray-700 dark:text-gray-300",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "span"

  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }