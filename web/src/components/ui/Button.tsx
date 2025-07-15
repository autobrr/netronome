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
}

export const Button = ({
  className,
  children,
  isLoading,
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
        "px-6 py-2 rounded-lg font-medium transition-colors border shadow-md flex items-center gap-2 min-w-[100px] justify-center",
        className
      )}
      {...buttonProps}
    >
      <div className="flex items-center gap-2">
        <Loader />
        <span>{children}</span>
      </div>
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
