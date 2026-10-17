// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/config"
)

// ExportDataset names a set of results that ExportResults can write.
type ExportDataset string

const (
	ExportSpeedTests ExportDataset = "speedtests"
	ExportPacketLoss ExportDataset = "packetloss"
	ExportDNS        ExportDataset = "dns"
)

// ExportFormat is the file format that ExportResults writes.
type ExportFormat string

const (
	ExportCSV  ExportFormat = "csv"
	ExportJSON ExportFormat = "json"
)

// Valid reports whether ExportResults knows the dataset.
func (d ExportDataset) Valid() bool {
	_, ok := exportDatasets[d]
	return ok
}

// Valid reports whether ExportResults can write the format.
func (f ExportFormat) Valid() bool {
	return f == ExportCSV || f == ExportJSON
}

// exportKind tells how to scan a column and how to write its value.
type exportKind int

const (
	exportText exportKind = iota
	exportInt
	exportFloat
	exportBool
	exportTime
	exportLatency // provider latency text, written as milliseconds
)

type exportColumn struct {
	name string // the column name in the file, part of the contract in ADR-0003
	expr string // the SQL expression; "r" is the results table
	kind exportKind
}

type exportDataset struct {
	from    string // the results table as "r", with its joins
	columns []exportColumn
}

var exportDatasets = map[ExportDataset]exportDataset{
	ExportSpeedTests: {
		from: "speed_tests r",
		columns: []exportColumn{
			{"id", "r.id", exportInt},
			{"created_at", "r.created_at", exportTime},
			{"test_type", "r.test_type", exportText},
			{"server_name", "r.server_name", exportText},
			{"server_id", "r.server_id", exportText},
			{"server_host", "r.server_host", exportText},
			{"server_city", "r.server_city", exportText},
			{"download_mbps", "r.download_speed", exportFloat},
			{"upload_mbps", "r.upload_speed", exportFloat},
			{"latency_ms", "r.latency", exportLatency},
			{"jitter_ms", "r.jitter", exportFloat},
			{"is_scheduled", "r.is_scheduled", exportBool},
		},
	},
	// Deleting a monitor deletes its results, so the join keeps every row.
	ExportPacketLoss: {
		from: "packet_loss_results r JOIN packet_loss_monitors m ON m.id = r.monitor_id",
		columns: []exportColumn{
			{"id", "r.id", exportInt},
			{"created_at", "r.created_at", exportTime},
			{"monitor_id", "r.monitor_id", exportInt},
			{"monitor_name", "m.name", exportText},
			{"monitor_host", "m.host", exportText},
			{"packet_loss_percent", "r.packet_loss", exportFloat},
			{"min_rtt_ms", "r.min_rtt", exportFloat},
			{"avg_rtt_ms", "r.avg_rtt", exportFloat},
			{"max_rtt_ms", "r.max_rtt", exportFloat},
			{"std_dev_rtt_ms", "r.std_dev_rtt", exportFloat},
			{"packets_sent", "r.packets_sent", exportInt},
			{"packets_received", "r.packets_recv", exportInt},
			{"used_mtr", "r.used_mtr", exportBool},
			{"hop_count", "r.hop_count", exportInt},
		},
	},
	ExportDNS: {
		from: "dns_results r JOIN dns_monitors m ON m.id = r.monitor_id",
		columns: []exportColumn{
			{"id", "r.id", exportInt},
			{"created_at", "r.created_at", exportTime},
			{"monitor_id", "r.monitor_id", exportInt},
			{"monitor_name", "m.name", exportText},
			{"resolver", "m.host", exportText},
			{"protocol", "m.protocol", exportText},
			{"query", "m.query", exportText},
			{"record_type", "m.record_type", exportText},
			{"response_time_ms", "r.response_time_ms", exportFloat},
			{"response_code", "r.response_code", exportText},
			{"success", "r.success", exportBool},
			{"error", "r.error", exportText},
		},
	},
}

// ExportResults writes the results of one dataset to w, oldest first. A zero
// from or to leaves the range open on that side. from is inclusive, to is
// exclusive. It reads and writes one row at a time.
func (s *service) ExportResults(ctx context.Context, w io.Writer, dataset ExportDataset, format ExportFormat, from, to time.Time) error {
	spec, ok := exportDatasets[dataset]
	if !ok || !format.Valid() {
		return ErrInvalidInput
	}

	exprs := make([]string, len(spec.columns))
	for i, col := range spec.columns {
		exprs[i] = col.expr
	}
	query := s.sqlBuilder.Select(exprs...).From(spec.from)

	// SQLite compares timestamps as text, and some rows hold a local offset.
	// julianday compares the instants. Migration 026 indexes this expression,
	// so keep the two the same.
	createdAt := "r.created_at"
	param := "?"
	if s.config.Type == config.SQLite {
		createdAt = "julianday(r.created_at)"
		param = "julianday(?)"
	}
	if !from.IsZero() {
		query = query.Where(sq.Expr(createdAt+" >= "+param, from.UTC()))
	}
	if !to.IsZero() {
		query = query.Where(sq.Expr(createdAt+" < "+param, to.UTC()))
	}
	query = query.OrderBy(createdAt, "r.id")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to query %s export: %w", dataset, err)
	}
	defer rows.Close()

	names := make([]string, len(spec.columns))
	for i, col := range spec.columns {
		names[i] = col.name
	}
	out, err := newExportWriter(w, format, names)
	if err != nil {
		return err
	}

	dest := make([]any, len(spec.columns))
	for i, col := range spec.columns {
		dest[i] = newExportScanTarget(col.kind)
	}
	values := make([]any, len(spec.columns))
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("failed to scan %s export row: %w", dataset, err)
		}
		for i, col := range spec.columns {
			values[i] = exportValue(col.kind, dest[i])
		}
		if err := out.row(values); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read %s export rows: %w", dataset, err)
	}
	return out.close()
}

// exportWriter writes the rows of one file in CSV or JSON.
type exportWriter struct {
	names []string
	csv   *csv.Writer
	json  *jsontext.Encoder
}

// newExportWriter writes the CSV header or opens the JSON array.
func newExportWriter(w io.Writer, format ExportFormat, names []string) (*exportWriter, error) {
	if format == ExportJSON {
		// Invalid UTF-8 in stored text becomes U+FFFD, so that one row does not stop the export.
		out := &exportWriter{names: names, json: jsontext.NewEncoder(w, jsontext.AllowInvalidUTF8(true))}
		return out, out.json.WriteToken(jsontext.BeginArray)
	}
	out := &exportWriter{names: names, csv: csv.NewWriter(w)}
	return out, out.csv.Write(names)
}

func (e *exportWriter) row(values []any) error {
	if e.csv != nil {
		record := make([]string, len(values))
		for i, v := range values {
			record[i] = csvCell(v)
		}
		return e.csv.Write(record)
	}

	// Write the tokens one by one, because a map would sort the keys.
	if err := e.json.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for i, v := range values {
		if err := e.json.WriteToken(jsontext.String(e.names[i])); err != nil {
			return err
		}
		if err := e.json.WriteToken(jsonToken(v)); err != nil {
			return err
		}
	}
	return e.json.WriteToken(jsontext.EndObject)
}

func (e *exportWriter) close() error {
	if e.csv != nil {
		e.csv.Flush()
		return e.csv.Error()
	}
	return e.json.WriteToken(jsontext.EndArray)
}

func jsonToken(value any) jsontext.Token {
	switch v := value.(type) {
	case string:
		return jsontext.String(v)
	case int64:
		return jsontext.Int(v)
	case float64:
		return jsontext.Float(v)
	case bool:
		return jsontext.Bool(v)
	}
	return jsontext.Null
}

func newExportScanTarget(kind exportKind) any {
	switch kind {
	case exportInt:
		return new(sql.NullInt64)
	case exportFloat:
		return new(sql.NullFloat64)
	case exportBool:
		return new(sql.NullBool)
	case exportTime:
		return new(sql.NullTime)
	case exportText, exportLatency:
		return new(sql.NullString)
	default:
		return new(sql.NullString)
	}
}

// exportValue returns nil, string, int64, float64, or bool.
func exportValue(kind exportKind, target any) any {
	switch v := target.(type) {
	case *sql.NullInt64:
		if v.Valid {
			return v.Int64
		}
	case *sql.NullFloat64:
		if v.Valid {
			return v.Float64
		}
	case *sql.NullBool:
		if v.Valid {
			return v.Bool
		}
	case *sql.NullTime:
		if v.Valid {
			return v.Time.UTC().Format(time.RFC3339)
		}
	case *sql.NullString:
		if !v.Valid {
			return nil
		}
		if kind == exportLatency {
			return latencyMs(v.String)
		}
		return v.String
	}
	return nil
}

// csvCell writes a value as CSV text. A text cell that a spreadsheet can read
// as a formula gets a "'" prefix (OWASP CSV injection guidance).
func csvCell(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
			return "'" + v
		}
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	return fmt.Sprint(value)
}

// latencyMs converts stored latency text to milliseconds. speedtest.net stores
// a Go duration ("12.345ms"). iperf3 stores milliseconds with a space
// ("15.20 ms"). LibreSpeed stores plain milliseconds ("12.35"). It returns nil
// when the text has none of these forms.
func latencyMs(latency string) any {
	if d, err := time.ParseDuration(strings.ReplaceAll(latency, " ", "")); err == nil {
		return float64(d) / float64(time.Millisecond)
	}
	ms, err := strconv.ParseFloat(latency, 64)
	if err != nil || math.IsNaN(ms) || math.IsInf(ms, 0) {
		return nil
	}
	return ms
}
