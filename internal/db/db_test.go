package db

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeSQLitePathExpandsHomeDir(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS != "windows" {
		t.Setenv("HOME", home)
	}

	path, err := normalizeSQLitePath("~/.config/sensorpanel.db.sqlite3")
	if err != nil {
		t.Fatalf("normalizeSQLitePath returned error: %v", err)
	}

	expected := filepath.Join(home, ".config", "sensorpanel.db.sqlite3")
	if path != expected {
		t.Fatalf("expected expanded path %q, got %q", expected, path)
	}
}

func TestNormalizeSQLitePathUsesDefaultPath(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS != "windows" {
		t.Setenv("HOME", home)
	}

	path, err := normalizeSQLitePath("")
	if err != nil {
		t.Fatalf("normalizeSQLitePath returned error: %v", err)
	}

	expected := filepath.Join(home, ".config", "sensorpanel.db.sqlite3")
	if path != expected {
		t.Fatalf("expected default path %q, got %q", expected, path)
	}
}

func TestNormalizeSQLitePathRejectsURIs(t *testing.T) {
	if _, err := normalizeSQLitePath("sqlite:///tmp/sensorpanel.db.sqlite3"); err == nil {
		t.Fatal("expected normalizeSQLitePath to reject URI format")
	}
}
