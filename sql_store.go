// Copyright 2022 Ross Light
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//		 https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"database/sql"
	"fmt"
	"time"

	"tailscale.com/ipn"
)

// SQLStore implements ipn.StateStore interface using a SQL database.
// It supports PostgreSQL, MySQL, and SQLite.
type SQLStore struct {
	db       *sql.DB
	driver   string
	table    string
	dsn      string
}

var _ ipn.StateStore = (*SQLStore)(nil)

// NewSQLStore creates a new SQL-backed state store.
// Supported drivers: "postgres" (alias for "pgx"), "pgx", "mysql", "sqlite3"
func NewSQLStore(driver, dsn string) (*SQLStore, error) {
	return NewSQLStoreWithTable(driver, dsn, "tailscale_state")
}

// NewSQLStoreWithTable creates a new SQL-backed state store with a custom table name.
func NewSQLStoreWithTable(driver, dsn, table string) (*SQLStore, error) {
	if driver == "" {
		return nil, fmt.Errorf("sql store: driver cannot be empty")
	}
	if dsn == "" {
		return nil, fmt.Errorf("sql store: dsn cannot be empty")
	}

	// For backwards compatibility, treat "postgres" as an alias for "pgx"
	actualDriver := driver
	if driver == "postgres" {
		actualDriver = "pgx"
	}

	db, err := sql.Open(actualDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql store: open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &SQLStore{
		db:     db,
		driver: driver,
		table:  table,
		dsn:    dsn,
	}

	if err := store.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sql store: initialize: %w", err)
	}

	return store, nil
}

// init creates the state table if it doesn't exist
func (s *SQLStore) init() error {
	var createTableSQL string

	switch s.driver {
	case "postgres", "pgx":
		createTableSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key TEXT PRIMARY KEY,
				value BYTEA NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`, s.table)
	case "mysql":
		createTableSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key VARCHAR(255) PRIMARY KEY,
				value BLOB NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`, s.table)
	case "sqlite3", "sqlite":
		createTableSQL = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key TEXT PRIMARY KEY,
				value BLOB NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`, s.table)
	default:
		return fmt.Errorf("unsupported database driver: %s", s.driver)
	}

	_, err := s.db.Exec(createTableSQL)
	return err
}

// ReadState retrieves the state associated with the given key.
// Returns (nil, ipn.ErrStateNotExist) if the key doesn't exist.
func (s *SQLStore) ReadState(key ipn.StateKey) ([]byte, error) {
	query := fmt.Sprintf("SELECT value FROM %s WHERE key = $1", s.table)
	
	// MySQL uses ? placeholders instead of $1
	if s.driver == "mysql" || s.driver == "sqlite3" || s.driver == "sqlite" {
		query = fmt.Sprintf("SELECT value FROM %s WHERE key = ?", s.table)
	}

	var value []byte
	err := s.db.QueryRow(query, string(key)).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, ipn.ErrStateNotExist
	}
	if err != nil {
		return nil, fmt.Errorf("sql store: read state: %w", err)
	}

	return value, nil
}

// WriteState saves the state associated with the given key.
func (s *SQLStore) WriteState(key ipn.StateKey, value []byte) error {
	var query string

	switch s.driver {
	case "postgres", "pgx":
		query = fmt.Sprintf(`
			INSERT INTO %s (key, value, updated_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (key)
			DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
			s.table)
		_, err := s.db.Exec(query, string(key), value, time.Now())
		return err

	case "mysql":
		query = fmt.Sprintf(`
			INSERT INTO %s (key, value, updated_at)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = VALUES(updated_at)`,
			s.table)
		_, err := s.db.Exec(query, string(key), value, time.Now())
		return err

	case "sqlite3", "sqlite":
		query = fmt.Sprintf(`
			INSERT INTO %s (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key)
			DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			s.table)
		_, err := s.db.Exec(query, string(key), value, time.Now())
		return err

	default:
		return fmt.Errorf("unsupported database driver: %s", s.driver)
	}
}

// Close closes the database connection
func (s *SQLStore) Close() error {
	return s.db.Close()
}

// Cleanup deletes state entries older than the specified duration.
// This is an optional maintenance operation.
func (s *SQLStore) Cleanup(olderThan time.Duration) error {
	var query string
	cutoff := time.Now().Add(-olderThan)

	switch s.driver {
	case "postgres", "pgx":
		query = fmt.Sprintf("DELETE FROM %s WHERE updated_at < $1", s.table)
		_, err := s.db.Exec(query, cutoff)
		return err
	case "mysql", "sqlite3", "sqlite":
		query = fmt.Sprintf("DELETE FROM %s WHERE updated_at < ?", s.table)
		_, err := s.db.Exec(query, cutoff)
		return err
	default:
		return fmt.Errorf("unsupported database driver: %s", s.driver)
	}
}
