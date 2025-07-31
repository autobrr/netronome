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

### Component Variant Architecture (CVA)

**Use class-variance-authority for complex component variants:**

```typescript
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-blue-500 text-white hover:bg-blue-600 dark:bg-blue-600 dark:hover:bg-blue-700 shadow-lg",
        destructive: "bg-red-500 text-white hover:bg-red-600 dark:bg-red-600 dark:hover:bg-red-700 shadow-lg",
        outline: "border border-gray-300 bg-white text-gray-900 hover:bg-gray-50 hover:text-gray-900 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:hover:bg-gray-800 dark:hover:text-gray-100 shadow-lg",
        secondary: "bg-gray-200/50 text-gray-900 hover:bg-gray-300/50 dark:bg-gray-800/50 dark:text-gray-100 dark:hover:bg-gray-800 border border-gray-300 dark:border-gray-800 shadow-lg",
        ghost: "text-gray-900 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-100 dark:hover:bg-gray-800 dark:hover:text-gray-100",
        link: "text-blue-500 underline-offset-4 hover:underline dark:text-blue-400",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-8",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  isLoading?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, isLoading = false, children, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      >
        {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        {children}
      </Comp>
    );
  }
);
Button.displayName = "Button";
```

### Radix UI Integration Patterns

**Use Radix UI primitives with proper composition:**

```typescript
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { Slot } from "@radix-ui/react-slot";

// Compound component pattern
function Dialog(props: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />;
}

function DialogContent({
  className,
  children,
  showCloseButton = true,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content> & {
  showCloseButton?: boolean;
}) {
  return (
    <DialogPortal>
      <DialogOverlay />
      <DialogPrimitive.Content
        data-slot="dialog-content"
        className={cn(
          "bg-gray-50/95 dark:bg-gray-850/95 fixed top-[50%] left-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 rounded-xl border p-6 shadow-xl",
          className
        )}
        {...props}
      >
        {children}
        {showCloseButton && (
          <DialogPrimitive.Close className="absolute top-4 right-4">
            <XMarkIcon />
            <span className="sr-only">Close</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </DialogPortal>
  );
}

// Export all sub-components
export { Dialog, DialogContent, DialogHeader, DialogFooter, DialogTitle };
```

### Component Composition

**Create reusable sub-components with status-based styling:**

```typescript
const MetricCard: React.FC<{
  icon: React.ReactNode;
  title: string;
  value: string;
  unit: string;
  average?: string;
  status?: "normal" | "warning" | "error" | "success";
}> = ({ icon, title, value, unit, average, status = "normal" }) => {
  const statusColors = {
    normal: "",
    success: "ring-1 ring-emerald-500/20 bg-emerald-500/5",
    warning: "ring-1 ring-amber-500/20 bg-amber-500/5",
    error: "ring-1 ring-red-500/20 bg-red-500/5",
  };

  const valueColors = {
    normal: "text-gray-900 dark:text-white",
    success: "text-emerald-600 dark:text-emerald-400",
    warning: "text-amber-600 dark:text-amber-400",
    error: "text-red-600 dark:text-red-400",
  };

  return (
    <div className={`bg-gray-50/95 dark:bg-gray-850/95 p-3 sm:p-4 rounded-xl border border-gray-200 dark:border-gray-800 shadow-lg ${statusColors[status]}`}>
      <div className="flex items-center gap-2 sm:gap-3 mb-2">
        <div className="text-gray-600 dark:text-gray-400 flex-shrink-0">{icon}</div>
        <h3 className="text-gray-700 dark:text-gray-300 font-medium text-sm sm:text-base truncate">
          {title}
        </h3>
      </div>
      <div className="flex items-baseline gap-1 sm:gap-2">
        <span className={`text-xl sm:text-2xl font-bold ${valueColors[status]}`}>
          {value}
        </span>
        <span className="text-gray-600 dark:text-gray-400 text-sm sm:text-base">{unit}</span>
      </div>
      {average && (
        <div className="mt-1 text-xs sm:text-sm text-gray-600 dark:text-gray-400 truncate">
          <span className="hidden sm:inline">Average: </span>
          <span className="sm:hidden">Avg: </span>
          {average} {unit}
        </div>
      )}
    </div>
  );
};
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

**Always provide dark mode alternatives with explicit text colors:**

```typescript
// Standard pattern - always include text colors for accessibility
className = "bg-white dark:bg-gray-800 text-gray-900 dark:text-white";

// Button variants must specify text colors explicitly
className = "bg-white text-gray-900 hover:bg-gray-50 dark:bg-gray-900 dark:text-gray-100 dark:hover:bg-gray-800";

// With opacity
className = "bg-gray-50/95 dark:bg-gray-850/95";

// For borders
className = "border-gray-200 dark:border-gray-800";
```

**Important**: Always specify explicit text colors, especially for interactive elements like buttons. Relying on inherited text colors can cause accessibility issues in dark mode.

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

### Primary Color Palette

**Core -400 Variants (Charts, Data Visualization):**

```typescript
// Blue-400: #60a5fa - Primary actions, download metrics
className = "text-blue-400";
className = "stroke-blue-400";

// Emerald-400: #34d399 - Success states, upload metrics
className = "text-emerald-400";
className = "stroke-emerald-400";

// Amber-400: #fbbf24 - Warning states, latency metrics
className = "text-amber-400";
className = "stroke-amber-400";

// Purple-400: #c084fc - Special actions, jitter metrics
className = "text-purple-400";
className = "stroke-purple-400";
```

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

**Advanced query configuration with caching strategies:**

```typescript
const statusQuery = useQuery<MonitorStatus>({
  queryKey: ["monitor-agent-status", agent.id],
  queryFn: () => getMonitorAgentStatus(agent.id),
  refetchInterval: agent.enabled ? MONITOR_REFRESH_INTERVALS.STATUS : false,
  staleTime: MONITOR_REFRESH_INTERVALS.STATUS / 2, // Consider data fresh for half the refetch interval
  gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes even when unused
  enabled: agent.enabled && includeData,
});
```

**Conditional queries based on multiple conditions:**

```typescript
const hardwareStatsQuery = useQuery<HardwareStats>({
  queryKey: ["monitor-agent-hardware", agent.id],
  queryFn: () => getMonitorAgentHardwareStats(agent.id),
  refetchInterval: agent.enabled ? MONITOR_REFRESH_INTERVALS.HARDWARE_STATS : false,
  staleTime: MONITOR_REFRESH_INTERVALS.HARDWARE_STATS,
  enabled: agent.enabled && includeHardwareStats && statusQuery.data?.connected,
});
```

**Direct cache manipulation with invalidation:**

```typescript
// Direct cache update followed by invalidation
queryClient.setQueryData(["packetloss", "history", monitorId], freshHistory);

// Force React Query to notify all subscribers
queryClient.invalidateQueries({
  queryKey: ["packetloss", "history", monitorId],
  exact: true,
});
```

**Mutation with toast notifications:**

```typescript
const startMutation = useMutation({
  mutationFn: () => startMonitorAgent(agent.id),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["monitor-agents"] });
    queryClient.invalidateQueries({ queryKey: ["monitor-agent-status", agent.id] });
    showToast("Agent started", "success", {
      description: `${agent.name} is now active`,
    });
  },
  onError: (error: Error) => {
    showToast("Failed to start agent", "error", {
      description: error.message || "Unable to start the monitoring agent",
    });
  },
});
```

### Custom Hook Patterns

**Hook with polling and cleanup:**

```typescript
export const usePacketLossMonitorStatus = (
  monitors: PacketLossMonitor[],
  selectedMonitorId?: number,
) => {
  const queryClient = useQueryClient();
  const [monitorStatuses, setMonitorStatuses] = useState<Map<number, MonitorStatus>>(new Map());

  useEffect(() => {
    const enabledMonitors = monitors.filter((m) => m.enabled);
    if (enabledMonitors.length === 0) return;

    const pollInterval = setInterval(async () => {
      const statusPromises = enabledMonitors.map(async (monitor) => {
        try {
          const status = await getPacketLossMonitorStatus(monitor.id);
          return { monitorId: monitor.id, status };
        } catch (error) {
          console.error(`Failed to get status for monitor ${monitor.id}:`, error);
          return null;
        }
      });

      const results = await Promise.all(statusPromises);
      
      // Update state and handle completion logic
      setMonitorStatuses((prev) => {
        const newStatuses = new Map(prev);
        results.forEach((result) => {
          if (result) newStatuses.set(result.monitorId, result.status);
        });
        return newStatuses;
      });
    }, 2000);

    return () => clearInterval(pollInterval);
  }, [monitors, queryClient, selectedMonitorId]);

  return monitorStatuses;
};
```

**Hook with complex options and multiple data sources:**

```typescript
interface UseMonitorAgentOptions {
  agent: MonitorAgent;
  includeNativeData?: boolean;
  includeSystemInfo?: boolean;
  includePeakStats?: boolean;
  includeHardwareStats?: boolean;
}

export const useMonitorAgent = ({
  agent,
  includeNativeData = false,
  includeSystemInfo = false,
  includePeakStats = false,
  includeHardwareStats = false,
}: UseMonitorAgentOptions) => {
  // Multiple queries with different refresh rates and conditions
  // Return complex data structure with loading states
  return {
    status: statusQuery.data,
    nativeData: nativeDataQuery.data,
    systemInfo: systemInfoQuery.data,
    peakStats: peakStatsQuery.data,
    hardwareStats: hardwareStatsQuery.data,
    isLoadingStatus: statusQuery.isLoading,
    isLoadingNativeData: nativeDataQuery.isLoading,
    // ... other loading states
    startMutation,
    stopMutation,
  };
};
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

## API Layer Patterns

### Consistent Error Handling

**Use standardized error handling across all API functions:**

```typescript
export async function getServers(testType: string) {
  try {
    const response = await fetch(getApiUrl(`/servers?testType=${testType}`));
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch servers");
    }
    return await response.json();
  } catch (error) {
    console.error("Error fetching servers:", error);
    throw error;
  }
}
```

**POST requests with proper headers:**

```typescript
export async function runSpeedTest(options: SpeedTestOptions) {
  try {
    const response = await fetch(getApiUrl("/speedtest"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(options),
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to run speed test");
    }
    return await response.json();
  } catch (error) {
    console.error("Error running speed test:", error);
    throw error;
  }
}
```

**Cache control for real-time data:**

```typescript
export async function getSpeedTestStatus() {
  try {
    const response = await fetch(getApiUrl("/speedtest/status"), {
      headers: {
        "Cache-Control": "no-cache",
        Pragma: "no-cache",
      },
    });
    // ... error handling
  } catch (error) {
    console.error("Error getting speed test status:", error);
    throw error;
  }
}
```

## Constants and Configuration Patterns

### Type-Safe Constants

**Use `as const` for immutable configuration:**

```typescript
export const MONITOR_REFRESH_INTERVALS = {
  STATUS: 5000, // 5 seconds
  HARDWARE_STATS: 30000, // 30 seconds
  NATIVE_DATA: 60000, // 1 minute
  SYSTEM_INFO: 300000, // 5 minutes
} as const;

// Extract type from constants
export type MonitorRefreshInterval = typeof MONITOR_REFRESH_INTERVALS[keyof typeof MONITOR_REFRESH_INTERVALS];
```

**Interface + data pattern for form options:**

```typescript
export interface IntervalOption {
  value: string;
  label: string;
}

export const intervalOptions: IntervalOption[] = [
  { value: "10s", label: "Every 10 seconds" },
  { value: "30s", label: "Every 30 seconds" },
  { value: "1m", label: "Every 1 minute" },
  { value: "5m", label: "Every 5 minutes" },
  // ... more options
];

export const defaultFormData: MonitorFormData = {
  host: "",
  name: "",
  interval: "30m",
  scheduleType: "interval",
  exactTimes: [],
  packetCount: 10,
  threshold: 5.0,
  enabled: true,
};
```

## Routing Patterns

### TanStack Router Configuration

**Route composition with authentication:**

```typescript
import { createRouter, createRoute, createRootRoute, Outlet } from "@tanstack/react-router";

// Protected route wrapper
function ProtectedRoute() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = router.navigate;

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate({ to: "/login" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading) {
    return <LoadingSpinner />;
  }

  return isAuthenticated ? <Outlet /> : null;
}

// Route tree structure
const rootRoute = createRootRoute({
  component: () => <App />,
});

const protectedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "protected",
  component: ProtectedRoute,
});

const indexRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: "/",
  component: Main,
});

const routeTree = rootRoute.addChildren([
  protectedRoute.addChildren([indexRoute]),
  authRoute.addChildren([loginRoute, registerRoute]),
]);

export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
  basepath: window.__BASE_URL__ || "/",
});
```

## Type Definition Patterns

### Complex Interface Hierarchies

**Define comprehensive type structures:**

```typescript
export interface PacketLossMonitor {
  id: number;
  host: string;
  name?: string;
  interval: string; // Duration string, not number
  packetCount: number;
  enabled: boolean;
  threshold: number;
  lastRun?: string;
  nextRun?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PacketLossResult {
  id: number;
  monitorId: number;
  packetLoss: number;
  minRtt: number;
  maxRtt: number;
  avgRtt: number;
  stdDevRtt: number;
  packetsSent: number;
  packetsRecv: number;
  usedMtr?: boolean;
  hopCount?: number;
  mtrData?: string; // JSON string containing MTRData
  privilegedMode?: boolean;
  createdAt: string;
}
```

**Union types for specific values:**

```typescript
export type TimeRange = "1d" | "3d" | "1w" | "1m" | "all";
export type TestType = "speedtest" | "iperf" | "librespeed";
```

**Generic interfaces for API responses:**

```typescript
export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  limit: number;
  total?: number;
}
```

---

## File Organization

### Directory Structure

```
src/
├── components/
│   ├── auth/           # Authentication components
│   ├── common/         # Shared components
│   ├── monitor/        # Monitor feature components
│   ├── settings/       # Settings and configuration components
│   ├── speedtest/      # Speed test features with deep nesting:
│   │   ├── packetloss/ # Feature-specific organization:
│   │   │   ├── components/    # Sub-feature components
│   │   │   ├── hooks/         # Feature-specific hooks
│   │   │   ├── types/         # Feature-specific types
│   │   │   ├── utils/         # Feature-specific utilities
│   │   │   └── constants/     # Feature-specific constants
│   │   └── traceroute/ # Similar deep structure
│   ├── icons/          # Custom icon components
│   └── ui/             # Base UI components (shadcn/ui)
├── api/                # API layer functions
├── constants/          # Global constants
├── context/            # React contexts
├── hooks/              # Global custom hooks
├── lib/                # Library utilities (utils.ts)
├── types/              # Global type definitions
├── utils/              # Global utility functions
├── routes.tsx          # TanStack Router configuration
└── main.tsx           # Application entry point
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

### Component Architecture Shortcuts

**CVA Button Variant:**

```typescript
const buttonVariants = cva(
  "base-classes",
  {
    variants: {
      variant: { default: "classes", destructive: "classes" },
      size: { default: "classes", sm: "classes", lg: "classes" },
    },
    defaultVariants: { variant: "default", size: "default" },
  }
);

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />;
  }
);
```

**Status-based Component:**

```typescript
const statusColors = {
  normal: "",
  success: "ring-1 ring-emerald-500/20 bg-emerald-500/5",
  warning: "ring-1 ring-amber-500/20 bg-amber-500/5",
  error: "ring-1 ring-red-500/20 bg-red-500/5",
};
```

### TanStack Query Shortcuts

**Complex Query Configuration:**

```typescript
const query = useQuery({
  queryKey: ["key", id],
  queryFn: () => fetchData(id),
  refetchInterval: enabled ? INTERVALS.STATUS : false,
  staleTime: INTERVALS.STATUS / 2,
  gcTime: 5 * 60 * 1000,
  enabled: enabled && hasData,
});
```

**Cache Manipulation:**

```typescript
// Direct update + invalidation pattern
queryClient.setQueryData(["key", id], newData);
queryClient.invalidateQueries({ queryKey: ["key", id], exact: true });
```

### API Pattern Shortcuts

**Standard API Function:**

```typescript
export async function apiFunction(param: string) {
  try {
    const response = await fetch(getApiUrl(`/endpoint?param=${param}`));
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || "Failed to fetch data");
    }
    return await response.json();
  } catch (error) {
    console.error("Error:", error);
    throw error;
  }
}
```

### Constants Pattern Shortcuts

**Type-safe Constants:**

```typescript
export const CONFIG = {
  VALUE1: 1000,
  VALUE2: 2000,
} as const;

export type ConfigValue = typeof CONFIG[keyof typeof CONFIG];
```

### Common Class Combinations

**Card/Panel:**

```typescript
className =
  "bg-gray-50/95 dark:bg-gray-850/95 rounded-xl p-6 shadow-lg border border-gray-200 dark:border-gray-800";
```

**Button (Primary):**

```typescript
className =
  "bg-blue-500 text-white hover:bg-blue-600 dark:bg-blue-600 dark:hover:bg-blue-700 shadow-lg";
```

**Button (Outline):**

```typescript
className =
  "border border-gray-300 bg-white text-gray-900 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:hover:bg-gray-800 shadow-lg";
```

**Button (Secondary):**

```typescript
className =
  "bg-gray-200/50 text-gray-900 hover:bg-gray-300/50 dark:bg-gray-800/50 dark:text-gray-100 dark:hover:bg-gray-800 border border-gray-300 dark:border-gray-800 shadow-lg";
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

#### Primary -400 Color Palette

| Color       | Hex Code  | Tailwind Class | Use Cases                 |
| ----------- | --------- | -------------- | ------------------------- |
| Blue-400    | `#60a5fa` | `blue-400`     | Download metrics, primary |
| Emerald-400 | `#34d399` | `emerald-400`  | Upload metrics, success   |
| Amber-400   | `#fbbf24` | `amber-400`    | Latency metrics, warnings |
| Purple-400  | `#c084fc` | `purple-400`   | Jitter metrics, special   |

#### Semantic Color Usage

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

## Recent Updates

This style guide was thoroughly updated based on analysis of the current codebase patterns. The following major additions were made:

### New Sections Added:
- **Component Variant Architecture (CVA)**: Modern variant management with `class-variance-authority`
- **Radix UI Integration Patterns**: Compound components and `asChild` prop patterns
- **Advanced TanStack Query Patterns**: Complex caching strategies, conditional queries, and cache manipulation
- **Custom Hook Patterns**: Polling, cleanup, and complex data management
- **API Layer Patterns**: Consistent error handling and request patterns
- **Constants and Configuration Patterns**: Type-safe configurations with `as const`
- **Routing Patterns**: TanStack Router composition and authentication
- **Type Definition Patterns**: Complex interfaces and type extraction

### Updated Sections:
- **Directory Structure**: Added actual deep nesting patterns used in the codebase
- **Component Composition**: Added status-based styling patterns
- **Quick Reference**: Added shortcuts for new patterns

### Latest Update: Button Component Patterns (January 2025)

- **Updated CVA Button Examples**: Now show complete variant definitions with proper dark mode support
- **Added Button (Outline) Pattern**: Documented the full outline variant with explicit text colors
- **Enhanced Dark Mode Guidelines**: Added emphasis on explicit text colors for accessibility
- **Fixed Quick Reference**: Updated button shortcuts to match actual implementation patterns

**Key Insight**: The actual Button component implementation was more comprehensive than the style guide examples, demonstrating the importance of keeping documentation aligned with production code.

This style guide should be updated as new patterns emerge and the codebase evolves. When adding new components or patterns, ensure they follow these established conventions for consistency across the application.
