package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mozership/probe/sdk"
)

// TestManifest checks that action.yml declares what the action does: the
// guard it keeps to, the keys it takes in with, and the fields of its res.
func TestManifest(t *testing.T) {
	data, err := os.ReadFile("action.yml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := sdk.ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Guard, keeps) {
		t.Errorf("guard = %v, want %v", m.Guard, keeps)
	}
	if want := sdk.ParamNames(Req{}); !slices.Equal(m.Params, want) {
		t.Errorf("params = %v, want %v", m.Params, want)
	}
	if want := sdk.ParamNames(Res{}); !slices.Equal(m.Res, want) {
		t.Errorf("res = %v, want %v", m.Res, want)
	}
}

// TestRun runs statements against SQLite, and checks that res has the
// fields action.yml declares, whether the statement reads, writes or fails.
func TestRun(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db")
	a := &Action{}
	tests := []struct {
		query string
		code  int
	}{
		{"CREATE TABLE users (id INTEGER)", 0},
		{"INSERT INTO users (id) VALUES (1)", 0},
		{"SELECT id FROM users", 0},
		{"SELECT * FROM missing_table", 1},
	}
	for _, tt := range tests {
		ret, err := a.Run(map[string]any{"dsn": dsn, "query": tt.query, "params": []any{}, "timeout": "5s"})
		if err != nil {
			t.Fatalf("%s: %v", tt.query, err)
		}
		res, ok := ret["res"].(map[string]any)
		if !ok {
			t.Fatalf("%s: res = %#v", tt.query, ret["res"])
		}
		for _, f := range sdk.ParamNames(Res{}) {
			if _, ok := res[f]; !ok {
				t.Errorf("%s: res has no %s: %v", tt.query, f, res)
			}
		}
		if res["code"] != tt.code {
			t.Errorf("%s: res.code = %v, want %d", tt.query, res["code"], tt.code)
		}
	}
}

func TestRunRequired(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db")
	tests := []struct {
		name string
		with map[string]any
		want string
	}{
		{"no dsn", map[string]any{"query": "SELECT 1"}, "dsn parameter is required"},
		{"an empty dsn", map[string]any{"dsn": "", "query": "SELECT 1"}, "dsn parameter must be a non-empty string"},
		{"no query", map[string]any{"dsn": dsn}, "query parameter is required"},
		{"a query that is not a string", map[string]any{"dsn": dsn, "query": 1}, "query parameter must be a non-empty string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&Action{}).Run(tt.with)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Run error = %v, want one that contains %q", err, tt.want)
			}
		})
	}
}

// TestRunStepReadOnly checks that a write is refused under read-only.
func TestRunStepReadOnly(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db")
	_, _, err := (&Action{}).RunStep(sdk.Call{
		With:  map[string]any{"dsn": dsn, "query": "CREATE TABLE t (id INTEGER)"},
		Guard: sdk.Guard{ReadOnly: true},
	})
	if !sdk.IsRefused(err) {
		t.Errorf("RunStep error = %v, want one that refuses the step", err)
	}
}
