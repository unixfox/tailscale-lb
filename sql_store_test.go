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
	"testing"
	"time"

	"tailscale.com/ipn"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLStore_SQLite(t *testing.T) {
	store, err := NewSQLStore("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQL store: %v", err)
	}
	defer store.Close()

	testStateStore(t, store)
}

func TestSQLStore_CustomTable(t *testing.T) {
	store, err := NewSQLStoreWithTable("sqlite3", ":memory:", "custom_state")
	if err != nil {
		t.Fatalf("Failed to create SQL store: %v", err)
	}
	defer store.Close()

	testStateStore(t, store)
}

func TestSQLStore_Cleanup(t *testing.T) {
	store, err := NewSQLStore("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQL store: %v", err)
	}
	defer store.Close()

	// Write some state
	key := ipn.StateKey("test-cleanup")
	value := []byte("cleanup-value")
	if err := store.WriteState(key, value); err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	// Cleanup entries older than 1 hour (should not delete our fresh entry)
	if err := store.Cleanup(1 * time.Hour); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify state still exists
	retrieved, err := store.ReadState(key)
	if err != nil {
		t.Fatalf("ReadState after cleanup failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Retrieved value = %q, want %q", retrieved, value)
	}

	// Cleanup entries older than 0 seconds (should delete everything)
	time.Sleep(10 * time.Millisecond)
	if err := store.Cleanup(0); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify state was deleted
	_, err = store.ReadState(key)
	if err != ipn.ErrStateNotExist {
		t.Errorf("Expected ErrStateNotExist after cleanup, got: %v", err)
	}
}

func TestSQLStore_InvalidDriver(t *testing.T) {
	_, err := NewSQLStore("invalid-driver", "some-dsn")
	if err == nil {
		t.Error("Expected error for invalid driver, got nil")
	}
}

func TestSQLStore_EmptyDriver(t *testing.T) {
	_, err := NewSQLStore("", "some-dsn")
	if err == nil {
		t.Error("Expected error for empty driver, got nil")
	}
}

func TestSQLStore_EmptyDSN(t *testing.T) {
	_, err := NewSQLStore("sqlite3", "")
	if err == nil {
		t.Error("Expected error for empty DSN, got nil")
	}
}

// testStateStore runs common tests for any StateStore implementation
func testStateStore(t *testing.T, store ipn.StateStore) {
	t.Helper()

	// Test WriteState and ReadState
	key := ipn.StateKey("test-key")
	value := []byte("test-value")

	if err := store.WriteState(key, value); err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	retrieved, err := store.ReadState(key)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Retrieved value = %q, want %q", retrieved, value)
	}

	// Test ReadState for non-existent key
	nonExistentKey := ipn.StateKey("non-existent")
	_, err = store.ReadState(nonExistentKey)
	if err != ipn.ErrStateNotExist {
		t.Errorf("Expected ErrStateNotExist for non-existent key, got: %v", err)
	}

	// Test updating existing state
	newValue := []byte("updated-value")
	if err := store.WriteState(key, newValue); err != nil {
		t.Fatalf("WriteState (update) failed: %v", err)
	}

	retrieved, err = store.ReadState(key)
	if err != nil {
		t.Fatalf("ReadState (after update) failed: %v", err)
	}

	if string(retrieved) != string(newValue) {
		t.Errorf("Retrieved updated value = %q, want %q", retrieved, newValue)
	}

	// Test with binary data
	binaryKey := ipn.StateKey("binary-key")
	binaryValue := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	if err := store.WriteState(binaryKey, binaryValue); err != nil {
		t.Fatalf("WriteState (binary) failed: %v", err)
	}

	retrieved, err = store.ReadState(binaryKey)
	if err != nil {
		t.Fatalf("ReadState (binary) failed: %v", err)
	}

	if len(retrieved) != len(binaryValue) {
		t.Errorf("Binary value length = %d, want %d", len(retrieved), len(binaryValue))
	}
	for i := range binaryValue {
		if retrieved[i] != binaryValue[i] {
			t.Errorf("Binary value[%d] = 0x%02x, want 0x%02x", i, retrieved[i], binaryValue[i])
		}
	}

	// Test with empty value
	emptyKey := ipn.StateKey("empty-key")
	emptyValue := []byte{}
	if err := store.WriteState(emptyKey, emptyValue); err != nil {
		t.Fatalf("WriteState (empty) failed: %v", err)
	}

	retrieved, err = store.ReadState(emptyKey)
	if err != nil {
		t.Fatalf("ReadState (empty) failed: %v", err)
	}

	if len(retrieved) != 0 {
		t.Errorf("Empty value length = %d, want 0", len(retrieved))
	}

	// Test multiple keys
	keys := []ipn.StateKey{"key1", "key2", "key3"}
	values := [][]byte{[]byte("value1"), []byte("value2"), []byte("value3")}

	for i, k := range keys {
		if err := store.WriteState(k, values[i]); err != nil {
			t.Fatalf("WriteState for %q failed: %v", k, err)
		}
	}

	for i, k := range keys {
		retrieved, err := store.ReadState(k)
		if err != nil {
			t.Fatalf("ReadState for %q failed: %v", k, err)
		}
		if string(retrieved) != string(values[i]) {
			t.Errorf("Retrieved value for %q = %q, want %q", k, retrieved, values[i])
		}
	}
}
