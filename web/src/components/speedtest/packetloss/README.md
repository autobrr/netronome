# Packet Loss Tab Refactoring

This directory contains the refactored Packet Loss Tab component, split from a single 1842-line file into modular, maintainable components.

## Structure

```
packetloss/
├── PacketLossTab.tsx              # Main container component (~260 lines)
├── PacketLossMonitorList.tsx      # Monitor list with status display
├── PacketLossMonitorForm.tsx      # Add/Edit monitor form modal
├── PacketLossMonitorDetails.tsx   # Monitor details panel
├── components/
│   ├── CountryFlag.tsx           # Reusable country flag component
│   ├── EmptyStatePlaceholder.tsx # Placeholder when no monitor selected
│   ├── MonitorStatusCard.tsx     # Current status display
│   ├── MonitorPerformanceChart.tsx # Performance trends chart
│   ├── MonitorResultsTable.tsx   # Historical results table
│   └── MTRResultsDisplay.tsx     # MTR hop-by-hop analysis
├── hooks/
│   └── usePacketLossMonitorStatus.ts # Status polling logic
├── utils/
│   └── packetLossUtils.ts        # Shared utility functions
├── constants/
│   └── packetLossConstants.ts    # Configuration constants
└── types/
    └── monitorStatus.ts          # TypeScript type definitions
```

## Key Improvements

1. **Reduced Complexity**: Main component reduced from 1842 lines to ~260 lines
2. **Better Organization**: Clear separation of concerns with dedicated components
3. **Type Safety**: Proper TypeScript types instead of `any`
4. **Reusability**: Components can be used independently
5. **Maintainability**: Easier to locate and fix issues
6. **Performance**: Smaller components with focused re-renders

## Component Responsibilities

- **PacketLossTab**: Main orchestrator handling state and data fetching
- **PacketLossMonitorList**: Displays monitors with inline status updates
- **PacketLossMonitorForm**: Handles monitor creation/editing
- **PacketLossMonitorDetails**: Composes status, chart, and results
- **MTRResultsDisplay**: Specialized component for MTR data visualization
- **usePacketLossMonitorStatus**: Custom hook for polling monitor statuses

## MTR vs Regular Packet Loss

The refactoring maintains clear separation between:

- Regular ICMP ping monitoring (direct ping tests)
- MTR monitoring (hop-by-hop analysis with network path details)

MTR-specific code is isolated in `MTRResultsDisplay.tsx` and conditionally rendered based on the `usedMtr` flag.
