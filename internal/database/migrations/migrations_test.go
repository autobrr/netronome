// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"slices"
	"testing"
)

func TestSortMigrationFiles(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name:  "wider number sorts by value",
			files: []string{"sqlite/0027_x.sql", "sqlite/010_b.sql", "sqlite/009_a.sql"},
			want:  []string{"sqlite/009_a.sql", "sqlite/010_b.sql", "sqlite/0027_x.sql"},
		},
		{
			name:  "narrower number sorts by value",
			files: []string{"postgres/100_c_postgres.sql", "postgres/27_b_postgres.sql", "postgres/026_a_postgres.sql"},
			want:  []string{"postgres/026_a_postgres.sql", "postgres/27_b_postgres.sql", "postgres/100_c_postgres.sql"},
		},
		{
			name:  "same number keeps filename order",
			files: []string{"sqlite/025_a.sql", "sqlite/025_b.sql", "sqlite/026_c.sql"},
			want:  []string{"sqlite/025_a.sql", "sqlite/025_b.sql", "sqlite/026_c.sql"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := slices.Clone(tc.files)
			sortMigrationFiles(got)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("sortMigrationFiles(%v) = %v, want %v", tc.files, got, tc.want)
			}
		})
	}
}
