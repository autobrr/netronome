# Packet Loss History Cutover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep the current expandable MTR UI, but remove `mtrData` from the normal packet-loss history list so large MTR blobs no longer inflate history requests or server memory.

**Architecture:** Split packet-loss history into two shapes. The existing history endpoint becomes a paginated summary feed with no `mtrData`. A new detail endpoint returns a single result, including `mtrData`, only when a user expands an MTR row. Add a composite index so summary history queries stop building temp sort state for common monitor history reads.

**Tech Stack:** Go, Gin, SQLite/Postgres SQL via squirrel, React, TanStack Query, TypeScript

---

## File Map

- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/types/types.go`
  Summary packet-loss history type and detail type split.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/database.go`
  Database service interface additions for summary/detail history reads.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/packetloss.go`
  Summary query without `mtr_data`, single-result detail query with `mtr_data`.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/handlers/packetloss.go`
  Paginated summary history endpoint and new detail endpoint.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/server/server.go`
  Route registration for new detail endpoint.
- Add: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/migrations/sqlite/021_packetloss_history_summary_index.sql`
  Composite index for packet-loss history ordering.
- Add: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/migrations/postgres/021_packetloss_history_summary_index_postgres.sql`
  Postgres equivalent index.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/packetloss_integration_test.go`
  Database coverage for summary rows and detail fetch.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/types/types.ts`
  Split summary history row from full detail row.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/api/packetloss.ts`
  Paginated summary fetch and detail fetch.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/PacketLossMonitorDetails.tsx`
  Summary history query wiring.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/hooks/usePacketLossMonitorStatus.ts`
  Refetch summary history only.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/components/MonitorResultsTable.tsx`
  Lazy detail fetch per expanded row and pagination controls.
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/components/MTRResultsDisplay.tsx`
  Keep rendering contract aligned with lazy-loaded detail data.

### Task 1: Backend Summary/Detail Split

**Files:**
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/types/types.go`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/database.go`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/packetloss.go`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/handlers/packetloss.go`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/server/server.go`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/packetloss_integration_test.go`

- [ ] Add failing integration coverage for:
  - summary history returns rows without `mtr_data`
  - detail fetch returns `mtr_data` for one result
  - history endpoint pagination metadata is correct

- [ ] Introduce dedicated summary/detail result types in Go so the history list is not forced to carry the blob field.

- [ ] Replace `GetPacketLossResults(monitorID, limit)` with:
  - paginated summary read ordered by newest first
  - single-result detail read by result id + monitor id

- [ ] Update the handler layer:
  - existing `/packetloss/monitors/:id/history` becomes paginated summary-only
  - add `/packetloss/monitors/:id/history/:resultId` for full detail

- [ ] Update route registration and keep existing clients working only through the new response shape used by this branch.

- [ ] Run focused backend tests for packet-loss database/handler behavior.

### Task 2: Query/Index Hardening

**Files:**
- Add: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/migrations/sqlite/021_packetloss_history_summary_index.sql`
- Add: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/migrations/postgres/021_packetloss_history_summary_index_postgres.sql`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/internal/database/packetloss_integration_test.go`

- [ ] Add a composite history index on `(monitor_id, created_at DESC)` for both SQLite and Postgres.

- [ ] Extend integration coverage so history reads still return newest-first rows after migrations.

- [ ] Run focused migration/integration tests for packet-loss database access.

### Task 3: Frontend Summary History + Lazy MTR Detail

**Files:**
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/types/types.ts`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/api/packetloss.ts`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/PacketLossMonitorDetails.tsx`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/hooks/usePacketLossMonitorStatus.ts`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/components/MonitorResultsTable.tsx`
- Modify: `/Users/soup/.codex/worktrees/4fd8/netronome/web/src/components/speedtest/packetloss/components/MTRResultsDisplay.tsx`

- [ ] Split packet-loss summary row typing from packet-loss detail typing in TypeScript.

- [ ] Change the default history request to fetch paginated summary rows only.

- [ ] Keep the current expandable MTR interaction, but lazy-load full detail when a row is expanded instead of parsing `mtrData` from the summary list.

- [ ] Preserve current summary UX:
  - timestamps
  - RTT metrics
  - sent/recv
  - mode badge
  - hop count

- [ ] Add simple page/load-more behavior on top of the paginated summary response instead of relying on one large result payload.

- [ ] Run frontend lint after the TypeScript/API changes.

### Task 4: Verification And Review Gate

**Files:**
- Review only

- [ ] Run focused backend tests covering packet-loss history and migrations.

- [ ] Run `pnpm -C web lint`.

- [ ] Run one end-to-end manual smoke check:
  - open packet-loss history
  - confirm summary list renders without raw `mtrData`
  - expand an MTR row
  - confirm the hop-by-hop UI still renders from the new detail endpoint

- [ ] Dispatch two parallel review subagents:
  - Reviewer A: `gpt-5.4` with `high` reasoning, focused on backend/API/query correctness
  - Reviewer B: `gpt-5.4` with `high` reasoning, focused on frontend behavior/regression risk

- [ ] Resolve reviewer findings before final handoff.

## Self-Review

- Spec coverage:
  - preserve current MTR UI behavior: covered in Task 3
  - hard cutover away from blob-heavy history list: covered in Task 1
  - reduce query memory pressure: covered in Task 2
  - review against plan with two subagents: covered in Task 4

- Placeholder scan:
  - no `TODO`/`TBD`
  - every task maps to concrete files and outcomes

- Type consistency:
  - plan uses `summary` vs `detail` terminology consistently across backend and frontend
