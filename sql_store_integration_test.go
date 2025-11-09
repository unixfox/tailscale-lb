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

//go:build integration
// +build integration

package main

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// To run integration tests:
// go test -tags=integration -v

func TestSQLStore_PostgreSQL_Integration(t *testing.T) {
	dsn := "postgres://tailscale:testpassword123@localhost:5432/tailscale_lb?sslmode=disable"
	store, err := NewSQLStore("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	t.Log("PostgreSQL store created successfully")
	testStateStore(t, store)
}

func TestSQLStore_MySQL_Integration(t *testing.T) {
	dsn := "tailscale:testpassword123@tcp(localhost:3306)/tailscale_lb?parseTime=true"
	store, err := NewSQLStore("mysql", dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer store.Close()

	t.Log("MySQL store created successfully")
	testStateStore(t, store)
}

func TestSQLStore_SQLite_Integration(t *testing.T) {
	dsn := "file:/tmp/tailscale-lb-integration-test.db?cache=shared&mode=rwc"
	store, err := NewSQLStore("sqlite3", dsn)
	if err != nil {
		t.Fatalf("SQLite failed: %v", err)
	}
	defer store.Close()

	t.Log("SQLite store created successfully")
	testStateStore(t, store)
}
