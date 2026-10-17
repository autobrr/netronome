# ADR-0003: Export columns are a contract separate from the API

Date: 2026-10-17
Status: accepted
Issue: #333

The export of results (CSV and JSON) uses one flat list of snake_case columns for each dataset, with the unit in the column name. The JSON export does not reuse the API types, which use camelCase, nest some fields, and keep latency as provider text. User scripts and spreadsheets depend on the export, so the export must not change when the API changes. Adding a column is not a breaking change. Renaming a column, removing a column, or changing a unit or the row order (oldest first) is a breaking change and needs a release note.

Considered option: JSON export as the API types. It needs less code. But every API change breaks user scripts, and tools cannot chart the latency text.
