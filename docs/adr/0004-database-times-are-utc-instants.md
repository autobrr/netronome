# ADR-0004: Database times are UTC instants

Date: 2026-11-12
Status: accepted
Issue: #355

The database stores each time as a UTC instant, and the connection applies the rule. On SQLite, the connection converts each time to UTC before it writes or compares the time. On Postgres, each time column has the type `TIMESTAMP WITH TIME ZONE`, and the session uses UTC. A query that compares a column with a Go time is correct on both databases, with no special function. A test fails when a Postgres time column has no time zone.

One migration rewrote the result times on SQLite to UTC, because the stored text holds the offset. Old Postgres values keep the time that they had. The database does not hold the zone that wrote them, so a conversion is a guess. On Postgres with a Netronome server time zone that is not UTC, packet loss results from before this change stay shifted by the UTC offset.

We also considered a module of predicate functions that each reader calls. A reader that does not call them repeats the fault. We considered Unix seconds as integers, which changes each column, each index, and the export. For the old Postgres rows, we considered a guess from the zone of the server and a command that the operator runs with a zone. The fault had no user reports, so the command was more code than the fault justified.
