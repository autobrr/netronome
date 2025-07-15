/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React, { useMemo, useRef, useEffect, useState } from "react";
import { motion } from "motion/react";

interface SparklineProps {
  data: number[];
  color: "blue" | "green";
  height?: number;
  strokeWidth?: number;
  showArea?: boolean;
}

export const MiniSparkline: React.FC<SparklineProps> = ({
  data,
  color,
  height = 32,
  strokeWidth = 1.5,
  showArea = true,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(100);

  useEffect(() => {
    const updateWidth = () => {
      if (containerRef.current) {
        setWidth(containerRef.current.offsetWidth);
      }
    };

    updateWidth();
    const resizeObserver = new ResizeObserver(updateWidth);
    if (containerRef.current) {
      resizeObserver.observe(containerRef.current);
    }

    return () => resizeObserver.disconnect();
  }, []);
  const { path, areaPath } = useMemo(() => {
    if (!data || data.length < 2) {
      return { path: "", areaPath: "" };
    }

    const min = Math.min(...data);
    const max = Math.max(...data);
    const range = max - min || 1;
    const padding = 2;

    const xStep = (width - padding * 2) / (data.length - 1);
    const yScale = (height - padding * 2) / range;

    const points = data.map((value, i) => ({
      x: padding + i * xStep,
      y: padding + (max - value) * yScale,
    }));

    const pathData = points.reduce((acc, point, i) => {
      if (i === 0) return `M ${point.x} ${point.y}`;
      return `${acc} L ${point.x} ${point.y}`;
    }, "");

    const areaData = `${pathData} L ${points[points.length - 1].x} ${
      height - padding
    } L ${padding} ${height - padding} Z`;

    return { path: pathData, areaPath: areaData };
  }, [data, height, width]);

  const strokeColor = color === "blue" ? "#3B82F6" : "#10B981";
  const fillColor = color === "blue" ? "#3B82F6" : "#10B981";

  if (!data || data.length < 2) {
    return (
      <div
        ref={containerRef}
        className="flex items-center justify-center text-xs text-gray-400 w-full"
        style={{ height }}
      >
        No data
      </div>
    );
  }

  return (
    <div ref={containerRef} className="w-full" style={{ height }}>
      <svg width={width} height={height} className="overflow-visible">
        {showArea && (
          <path
            d={areaPath}
            fill={fillColor}
            fillOpacity={0.1}
            className="transition-opacity duration-200"
          />
        )}
        <motion.path
          d={path}
          fill="none"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeLinejoin="round"
          initial={{ pathLength: 0 }}
          animate={{ pathLength: 1 }}
          transition={{ duration: 0.5, ease: "easeOut" }}
        />
      </svg>
    </div>
  );
};
