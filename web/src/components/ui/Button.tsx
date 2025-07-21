/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

"use client";
import { cn } from "@/lib/utils";
import React from "react";
import { motion, useAnimate } from "motion/react";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  className?: string;
  children: React.ReactNode;
  isLoading?: boolean;
  size?: "sm" | "md" | "lg";
  variant?: "primary" | "secondary" | "danger";
}

export const Button = ({
  className,
  children,
  isLoading,
  size = "md",
  variant = "primary",
  ...props
}: ButtonProps) => {
  const [scope, animate] = useAnimate();

  React.useEffect(() => {
    if (isLoading) {
      animate(
        ".loader",
        {
          width: "16px",
          scale: 1,
          display: "block",
        },
        {
          duration: 0.2,
        }
      );
    } else {
      animate(
        ".loader",
        {
          width: "0px",
          scale: 0,
          display: "none",
        },
        {
          duration: 0.2,
        }
      );
    }
  }, [isLoading, animate]);

  // Filter out motion-conflicting props
  const {
    onAnimationStart,
    onAnimationEnd,
    onDrag,
    onDragStart,
    onDragEnd,
    ...buttonProps
  } = props;

  return (
    <motion.button
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      ref={scope}
      className={cn(
        "rounded-lg font-medium transition-colors border shadow-md flex items-center justify-center gap-2",
        // Size variants
        size === "sm" && "px-3 py-1 text-sm min-w-[80px]",
        size === "md" && "px-6 py-2 min-w-[100px]",
        size === "lg" && "px-8 py-3 text-lg min-w-[120px]",
        // Color variants
        variant === "primary" && "bg-blue-500 hover:bg-blue-600 text-white border-blue-600",
        variant === "secondary" && "bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-white border-gray-300 dark:border-gray-600",
        variant === "danger" && "bg-red-500 hover:bg-red-600 text-white border-red-600",
        className
      )}
      {...buttonProps}
    >
      <Loader />
      {children}
    </motion.button>
  );
};

const Loader = () => {
  return (
    <motion.svg
      animate={{
        rotate: [0, 360],
      }}
      initial={{
        scale: 0,
        width: 0,
        display: "none",
      }}
      style={{
        scale: 0.5,
        display: "none",
      }}
      transition={{
        duration: 0.8,
        repeat: Infinity,
        ease: "linear",
      }}
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="loader"
    >
      <path stroke="none" d="M0 0h24v24H0z" fill="none" />
      <path d="M12 3a9 9 0 1 0 9 9" />
    </motion.svg>
  );
};
