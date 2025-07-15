# Vnstat Monitoring Implementation Progress

## Overview

This document tracks the implementation of vnstat bandwidth monitoring in Netronome, which allows monitoring network bandwidth from remote servers running the vnstat agent.

## Architecture

- **Agent Mode**: `netronome agent` runs on remote servers, broadcasting vnstat data via SSE
- **Server Mode**: Main Netronome server connects to agents as SSE clients and stores/displays data
- **Real-time Updates**: Live bandwidth data streamed via Server-Sent Events

## Implementation Progress

### ✅ Completed Tasks

#### 1. Agent Command CLI Structure

**File: `cmd/netronome/main.go`**

- Added `agentCmd` to the CLI command structure
- Implemented `runAgent` function to start the agent service
- Added command flags: `--port` and `--interface`

#### 2. Agent Service Implementation

**File: `internal/agent/agent.go`** (NEW)

- Created SSE server using Gin framework
- Implements vnstat live monitoring with JSON output parsing
- Broadcasts bandwidth data to connected clients
- Supports CORS for cross-origin access

#### 3. Configuration Support

**File: `internal/config/config.go`**

- Added `AgentConfig` struct with port and interface settings
- Added `VnstatConfig` struct for server-side vnstat settings
- Updated `New()` function to include default values
- Added environment variable loading for both configs
- Updated `WriteToml()` to include agent and vnstat sections

#### 4. Database Migrations

**Files:**

- `internal/database/migrations/sqlite/011_add_vnstat_monitoring.sql` (NEW)
- `internal/database/migrations/postgres/011_add_vnstat_monitoring_postgres.sql` (NEW)

Created tables:

- `vnstat_agents`: Stores agent configurations
- `vnstat_bandwidth`: Stores historical bandwidth data

#### 5. Type Definitions

**File: `internal/types/types.go`**
Added types:

- `VnstatAgent`: Agent configuration
- `VnstatBandwidth`: Bandwidth data point
- `VnstatLiveData`: Live data from vnstat
- `VnstatUpdate`: Real-time update broadcast

#### 6. Vnstat SSE Client Service

**File: `internal/vnstat/client.go`** (NEW)

- SSE client implementation for connecting to agents
- Automatic reconnection with exponential backoff
- Data parsing and storage
- Real-time broadcast support

#### 7. Database Methods

**Files:**

- `internal/database/database.go`: Added vnstat method signatures to Service interface
- `internal/database/vnstat.go` (NEW): Implemented all vnstat database operations

Methods implemented:

- `CreateVnstatAgent`
- `GetVnstatAgent`
- `GetVnstatAgents`
- `UpdateVnstatAgent`
- `DeleteVnstatAgent`
- `SaveVnstatBandwidth`
- `GetVnstatBandwidthHistory`
- `CleanupOldVnstatData`

#### 8. API Handlers

**File: `internal/handlers/vnstat.go`** (NEW)

Implemented all vnstat API endpoints:

- `GET /api/vnstat/agents` - List all agents
- `POST /api/vnstat/agents` - Create new agent
- `GET /api/vnstat/agents/:id` - Get agent details
- `PUT /api/vnstat/agents/:id` - Update agent
- `DELETE /api/vnstat/agents/:id` - Delete agent
- `GET /api/vnstat/agents/:id/status` - Get agent connection status
- `GET /api/vnstat/agents/:id/bandwidth` - Get bandwidth history
- `POST /api/vnstat/agents/:id/start` - Start agent monitoring
- `POST /api/vnstat/agents/:id/stop` - Stop agent monitoring

#### 9. Server Integration

**File: `internal/server/server.go`**

- Added vnstat service to Server struct
- Updated NewServer to accept vnstatService parameter
- Added BroadcastVnstatUpdate method
- Added SetVnstatService method
- Registered all vnstat API routes in RegisterRoutes

**File: `cmd/netronome/main.go`**

- Added vnstat service initialization in runServer
- Start vnstat service when enabled
- Stop vnstat service on shutdown

#### 10. Frontend Components

**Files created in `web/src/components/vnstat/`:**

- `VnstatTab.tsx` - Main tab component with agent management
- `VnstatAgentList.tsx` - List view showing all agents with connection status
- `VnstatAgentForm.tsx` - Modal form for adding/editing agents
- `VnstatBandwidthChart.tsx` - Recharts-based bandwidth history visualization
- `VnstatLiveMonitor.tsx` - Real-time bandwidth display cards
- `VnstatAgentDetails.tsx` - Detailed view for selected agent with controls

#### 11. Frontend API Client

**File: `web/src/api/vnstat.ts`** (NEW)

Implemented all API functions:

- `getVnstatAgents` - Fetch all agents
- `getVnstatAgent` - Fetch single agent
- `createVnstatAgent` - Create new agent
- `updateVnstatAgent` - Update agent
- `deleteVnstatAgent` - Delete agent
- `getVnstatAgentStatus` - Get connection status
- `getVnstatAgentBandwidth` - Get bandwidth history
- `startVnstatAgent` - Start monitoring
- `stopVnstatAgent` - Stop monitoring

#### 12. Tab Integration

**File: `web/src/components/speedtest/SpeedTest.tsx`**

- Added ServerIcon import
- Imported VnstatTab component
- Added "Bandwidth" tab to tab configuration
- Added tab content rendering for vnstat

### ❌ Remaining Tasks

None! The vnstat monitoring feature is now fully implemented.

## Testing the Agent

The agent can already be tested:

```bash
# Run the agent
netronome agent --port 8200 --interface eth0

# Test the SSE endpoint
curl http://localhost:8200/events?stream=live-data
```

## Next Steps

1. Implement API handlers for agent management
2. Integrate vnstat service into the main server
3. Build frontend components for agent management
4. Add real-time bandwidth monitoring to dashboard
5. Test end-to-end functionality with multiple agents
