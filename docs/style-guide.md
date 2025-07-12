# Frontend Style Guide

This document outlines the established patterns and conventions for frontend development in the Netronome project. It serves as the definitive reference for maintaining consistency across the codebase.

## Table of Contents

1. [Component Architecture](#component-architecture)
2. [Styling Conventions](#styling-conventions)
3. [Animation Patterns](#animation-patterns)
4. [Responsive Design](#responsive-design)
5. [Color System](#color-system)
6. [Typography](#typography)
7. [State Management](#state-management)
8. [Accessibility](#accessibility)
9. [Performance Optimizations](#performance-optimizations)
10. [File Organization](#file-organization)
11. [Quick Reference](#quick-reference)

---

## Component Architecture

### React Patterns

**Use functional components with TypeScript interfaces:**

```typescript
interface ComponentProps {
  title: string;
  isActive?: boolean;
  onAction?: () => void;
}

export const Component: React.FC<ComponentProps> = ({
  title,
  isActive = false,
  onAction,
}) => {
  // Component implementation
};
```

**Extend native HTML props when creating UI components:**

```typescript
interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  isLoading?: boolean;
  variant?: "primary" | "secondary";
}

export const Button: React.FC<ButtonProps> = ({
  isLoading = false,
  variant = "primary",
  children,
  className,
  ...props
}) => {
  // Filter motion-conflicting props
  const { whileHover, whileTap, ...buttonProps } = props;

  return (
    <motion.button
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      className={cn(baseStyles, variantStyles[variant], className)}
      {...buttonProps}
    >
      {isLoading ? <LoadingSpinner /> : children}
    </motion.button>
  );
};
```

### Component Composition

**Create reusable sub-components:**

```typescript
const MetricCard: React.FC<{
  icon: React.ReactNode;
  title: string;
  value: string;
  unit: string;
  average?: string;
}> = ({ icon, title, value, unit, average }) => (
  <div className="bg-gray-50/95 dark:bg-gray-850/95 p-4 rounded-xl border border-gray-200 dark:border-gray-900 shadow-lg">
    <div className="flex items-center gap-3 mb-2">
      <div className="text-gray-600 dark:text-gray-400">{icon}</div>
      <h3 className="text-gray-700 dark:text-gray-300 font-medium">{title}</h3>
    </div>
    <div className="flex items-baseline gap-2">
      <span className="text-2xl font-bold text-gray-900 dark:text-white">
        {value}
      </span>
      <span className="text-gray-600 dark:text-gray-400">{unit}</span>
    </div>
    {average && (
      <div className="mt-1 text-sm text-gray-600 dark:text-gray-400">
        Average: {average} {unit}
      </div>
    )}
  </div>
);
```

---

## Styling Conventions

### Tailwind CSS Usage

**Use utility classes with consistent patterns:**

```typescript
// Background patterns
className = "bg-gray-50/95 dark:bg-gray-850/95";

// Border patterns
className = "border border-gray-200 dark:border-gray-900";

// Text patterns
className = "text-gray-700 dark:text-gray-300";

// Interactive states
className = "hover:bg-gray-300/50 dark:hover:bg-gray-800 transition-colors";
```

**Use the `cn()` utility for conditional classes:**

```typescript
import { cn } from "@/lib/utils";

const Component = ({ isActive, className }) => (
  <div className={cn("base-styles", isActive && "active-styles", className)}>
    Content
  </div>
);
```

### Dark Mode Support

**Always provide dark mode alternatives:**

```typescript
// Standard pattern
className = "bg-white dark:bg-gray-800 text-gray-900 dark:text-white";

// With opacity
className = "bg-gray-50/95 dark:bg-gray-850/95";

// For borders
className = "border-gray-200 dark:border-gray-800";
```

### Shadow and Backdrop Effects

**Use consistent shadow patterns:**

```typescript
// Standard shadow
className = "shadow-lg";

// With backdrop blur
className = "backdrop-blur-sm bg-blue-500/10 border border-blue-500/30";
```

---

## Animation Patterns

### Motion/React Integration

**Define animation configurations outside components for performance:**

```typescript
// Animation configuration moved outside component to prevent re-creation
const SPRING_TRANSITION = {
  type: "spring" as const,
  stiffness: 500,
  damping: 30,
} as const;

const Component = () => (
  <motion.div
    initial={{ opacity: 0, y: 20 }}
    animate={{ opacity: 1, y: 0 }}
    transition={SPRING_TRANSITION}
  >
    Content
  </motion.div>
);
```

**Standard animation patterns:**

```typescript
// Page/section entrance
<motion.div
  initial={{ opacity: 0, y: 20 }}
  animate={{ opacity: 1, y: 0 }}
  exit={{ opacity: 0, y: 20 }}
  transition={{ duration: 0.5 }}
>

// List item entrance
<motion.tr
  initial={{ opacity: 0, y: 20 }}
  animate={{ opacity: 1, y: 0 }}
  transition={{ duration: 0.3 }}
>

// Button interactions
<motion.button
  whileHover={{ scale: 1.02 }}
  whileTap={{ scale: 0.98 }}
  transition={{ duration: 0.2 }}
>

// Layout transitions
<motion.div
  layoutId="activeTab"
  transition={{ type: "spring", stiffness: 500, damping: 30 }}
>
```

### Animation Timing

**Use consistent timing values:**

- **Quick interactions**: `0.2s`
- **Component transitions**: `0.3s`
- **Page transitions**: `0.5s`
- **Feedback delays**: `2s`

---

## Responsive Design

### Mobile-First Approach

**Use responsive prefixes consistently:**

```typescript
// Mobile-first breakpoints
className = "px-2 sm:px-4 md:px-6 lg:px-8";

// Component visibility
className = "hidden md:block"; // Desktop only
className = "md:hidden"; // Mobile only

// Grid layouts
className = "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4";
```

### Responsive Data Display

**Use table/card responsive patterns:**

```typescript
// Desktop table view
<div className="hidden md:block">
  <table className="w-full text-sm">
    {/* Table content */}
  </table>
</div>

// Mobile card view
<div className="md:hidden space-y-3">
  {items.map(item => (
    <div className="bg-gray-200/50 dark:bg-gray-800/50 rounded-lg p-4">
      {/* Card content */}
    </div>
  ))}
</div>
```

### Text Responsiveness

**Use responsive text sizing:**

```typescript
// Responsive text
className = "text-sm sm:text-base";

// Responsive spacing
className = "space-x-1 sm:space-x-2";

// Responsive padding
className = "px-2 sm:px-6 py-2 sm:py-3";
```

---

## Color System

### Semantic Color Usage

**Primary Actions (Blue):**

```typescript
className = "text-blue-600 dark:text-blue-400";
className = "bg-blue-500/10 border border-blue-500/30";
```

**Success/Positive (Emerald/Green):**

```typescript
className = "text-emerald-600 dark:text-emerald-400";
className = "text-green-600 dark:text-green-400"; // For data display
```

**Warning/Attention (Amber/Yellow):**

```typescript
className = "text-amber-600 dark:text-amber-400";
className = "text-yellow-600 dark:text-yellow-400"; // For data display
```

**Error/Danger (Red):**

```typescript
className = "text-red-600 dark:text-red-400";
className = "bg-red-500/10 border border-red-500/30";
```

**Special/Accent (Purple):**

```typescript
className = "text-purple-600 dark:text-purple-400";
className = "bg-purple-500/10 text-purple-600 dark:text-purple-400";
```

### Service Type Colors

**Speed Test Types:**

```typescript
// iperf3
className = "bg-purple-500/10 text-purple-600 dark:text-purple-400";

// LibreSpeed
className = "bg-blue-500/10 text-blue-600 dark:text-blue-400";

// Speedtest.net
className =
  "bg-emerald-200/50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400";
```

---

## Typography

### Font Patterns

**Standard text:**

```typescript
className = "font-normal"; // Default
className = "font-medium"; // Headers, labels
className = "font-semibold"; // Section titles
className = "font-bold"; // Emphasis, values
```

**Monospace for data:**

```typescript
className = "font-mono"; // Numbers, technical data
```

### Text Sizing

**Standard hierarchy:**

```typescript
className = "text-xs"; // 12px - Small labels, metadata
className = "text-sm"; // 14px - Body text, descriptions
className = "text-base"; // 16px - Standard body text
className = "text-lg"; // 18px - Large body text
className = "text-xl"; // 20px - Section headings
className = "text-2xl"; // 24px - Page titles, large values
```

### Text Colors

**Standard text colors:**

```typescript
className = "text-gray-900 dark:text-white"; // Primary text
className = "text-gray-700 dark:text-gray-300"; // Secondary text
className = "text-gray-600 dark:text-gray-400"; // Muted text
className = "text-gray-500 dark:text-gray-500"; // Disabled/placeholder
```

---

## State Management

### Local State Patterns

**Use useState for component state:**

```typescript
const [isOpen, setIsOpen] = useState(false);
const [displayCount, setDisplayCount] = useState(5);
```

**localStorage for user preferences:**

```typescript
const [isRecentTestsOpen] = useState(() => {
  const saved = localStorage.getItem("recent-tests-open");
  return saved === null ? true : saved === "true";
});

useEffect(() => {
  localStorage.setItem("recent-tests-open", open.toString());
}, [open]);
```

### TanStack Query Patterns

**Query with caching:**

```typescript
const { data: results } = useQuery<TracerouteResult | null>({
  queryKey: ["traceroute", "results"],
  queryFn: () => null,
  enabled: false,
  staleTime: Infinity,
  initialData: null,
});
```

**Mutation with optimistic updates:**

```typescript
const tracerouteMutation = useMutation({
  mutationFn: runTraceroute,
  onMutate: () => {
    // Clear previous results and error state
    queryClient.setQueryData(["traceroute", "results"], null);
    setError(null);
  },
  onSuccess: (data) => {
    queryClient.setQueryData(["traceroute", "results"], data);
  },
  onError: (error) => {
    setError(error.message);
  },
});
```

---

## Accessibility

### ARIA Attributes

**Use proper ARIA labels:**

```typescript
<button
  aria-label="Share public speed test page"
  aria-pressed={isActive}
  role="tab"
  aria-selected={isActive}
>
```

**Navigation patterns:**

```typescript
<nav role="tablist">
  <button role="tab" aria-selected={isActive} aria-pressed={isActive}>
    Tab Content
  </button>
</nav>
```

### Keyboard Navigation

**Support keyboard interactions:**

```typescript
<input
  onKeyDown={(e) => {
    if (e.key === "Enter" && !isLoading) {
      handleAction();
    }
  }}
/>
```

### Focus Management

**Provide focus indicators:**

```typescript
className =
  "focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50";
```

---

## Performance Optimizations

### Animation Performance

**Move configurations outside components:**

```typescript
// Good - defined outside component
const SPRING_TRANSITION = {
  type: "spring" as const,
  stiffness: 500,
  damping: 30,
} as const;

const Component = () => (
  <motion.div transition={SPRING_TRANSITION}>Content</motion.div>
);
```

### Memoization Patterns

**Use useMemo for expensive calculations:**

```typescript
const filteredServers = useMemo(() => {
  return allServers.filter((server) => {
    const matchesSearch =
      searchTerm === "" ||
      server.name.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesSearch;
  });
}, [allServers, searchTerm]);
```

### Component Optimization

**Use React.FC for consistency:**

```typescript
export const Component: React.FC<Props> = ({ prop1, prop2 }) => {
  // Component implementation
};
```

---

## File Organization

### Directory Structure

```
src/
├── components/
│   ├── auth/           # Authentication components
│   ├── common/         # Shared components
│   ├── speedtest/      # Feature-specific components
│   └── ui/             # Base UI components
├── utils/              # Utility functions
├── types/              # Type definitions
├── api/                # API layer
└── context/            # React contexts
```

### Naming Conventions

**Files and Components:**

- Components: `PascalCase.tsx`
- Utilities: `camelCase.ts`
- Types: `types.ts`
- Constants: `SCREAMING_SNAKE_CASE`

**Imports:**

```typescript
// Standard import order
import React from "react";
import { motion } from "motion/react";
import { useQuery } from "@tanstack/react-query";

// Local imports
import { Component } from "@/components/ui/Component";
import { utility } from "@/utils/utility";
import { Type } from "@/types/types";
```

### License Headers

**All source files must include:**

```typescript
/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */
```

---

## Quick Reference

### Common Class Combinations

**Card/Panel:**

```typescript
className =
  "bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800";
```

**Button (Primary):**

```typescript
className =
  "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700";
```

**Button (Secondary):**

```typescript
className =
  "bg-gray-200/50 dark:bg-gray-800/50 border-gray-300 dark:border-gray-800 hover:bg-gray-300/50 dark:hover:bg-gray-800";
```

**Input Field:**

```typescript
className =
  "px-4 py-2 bg-gray-200/50 dark:bg-gray-800/50 border border-gray-300 dark:border-gray-900 text-gray-700 dark:text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50";
```

**Status Badge:**

```typescript
className =
  "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-500/10 text-blue-600 dark:text-blue-400";
```

### Animation Shortcuts

**Standard entrance:**

```typescript
initial={{ opacity: 0, y: 20 }}
animate={{ opacity: 1, y: 0 }}
transition={{ duration: 0.5 }}
```

**Button interaction:**

```typescript
whileHover={{ scale: 1.02 }}
whileTap={{ scale: 0.98 }}
```

**Layout transition:**

```typescript
layoutId="uniqueId"
transition={{ type: "spring", stiffness: 500, damping: 30 }}
```

### Color Quick Reference

| Purpose        | Light              | Dark               |
| -------------- | ------------------ | ------------------ |
| Primary Text   | `text-gray-900`    | `text-white`       |
| Secondary Text | `text-gray-700`    | `text-gray-300`    |
| Muted Text     | `text-gray-600`    | `text-gray-400`    |
| Primary Action | `text-blue-600`    | `text-blue-400`    |
| Success        | `text-emerald-600` | `text-emerald-400` |
| Warning        | `text-amber-600`   | `text-amber-400`   |
| Error          | `text-red-600`     | `text-red-400`     |
| Special        | `text-purple-600`  | `text-purple-400`  |

---

This style guide should be updated as new patterns emerge and the codebase evolves. When adding new components or patterns, ensure they follow these established conventions for consistency across the application.
