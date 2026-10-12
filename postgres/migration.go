// Package postgres provides atomic server-time fenced leases through a borrowed
// pgx pool. Apply SchemaMigration/SchemaMigrationFor yourself before New; no
// schema setup, pool closure, retry or replication policy is owned here.
// Inactive rows retain counters and require deployment-managed storage policy.
package postgres

//lint:file-ignore SA1019 This file intentionally exercises or implements the retained compatibility contract.

import (
	"fmt"

	"github.com/faustbrian/go-scheduler/v2/lease" //nolint:staticcheck // Backend implements the retained domain port.
	"github.com/jackc/pgx/v5"
)

// Migration contains the versioned lease-table schema transition.
type Migration struct {
	// Version identifies the schema transition; it is not applied automatically.
	Version uint
	// Name is the stable application migration label.
	Name string
	// Up is executable creation SQL for caller-controlled migration execution.
	Up string
	// Down is destructive rollback SQL; callers own its authorization and sequencing.
	Down string
}

// SchemaMigration returns the PostgreSQL lease-table migration.
func SchemaMigration() Migration {
	return Migration{
		Version: 1,
		Name:    "create_scheduler_leases",
		Up: `CREATE TABLE scheduler_leases (
    lease_key text PRIMARY KEY,
    owner text NOT NULL,
    fencing_token bigint NOT NULL CHECK (fencing_token > 0),
    acquired_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    active boolean NOT NULL DEFAULT true
);
CREATE INDEX scheduler_leases_active_expiry_idx
    ON scheduler_leases (expires_at) WHERE active;`,
		Down: `DROP TABLE scheduler_leases;`,
	}
}

// SchemaMigrationFor returns the lease-table migration for a caller-owned
// schema. The schema itself must already exist.
func SchemaMigrationFor(schema string) (Migration, error) {
	if !schemaPattern.MatchString(schema) {
		return Migration{}, lease.ErrInvalid
	}
	table := pgx.Identifier{schema, "scheduler_leases"}.Sanitize()
	index := pgx.Identifier{
		"scheduler_leases_active_expiry_idx",
	}.Sanitize()

	return Migration{
		Version: 1,
		Name:    "create_scheduler_leases",
		Up: fmt.Sprintf(`CREATE TABLE %s (
    lease_key text PRIMARY KEY,
    owner text NOT NULL,
    fencing_token bigint NOT NULL CHECK (fencing_token > 0),
    acquired_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    active boolean NOT NULL DEFAULT true
);
CREATE INDEX %s
    ON %s (expires_at) WHERE active;`, table, index, table),
		Down: fmt.Sprintf("DROP TABLE %s;", table),
	}, nil
}
