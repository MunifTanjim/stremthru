package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginDirDefault(t *testing.T) {
	if PluginDir == "" {
		t.Fatal("PluginDir empty")
	}
}

func TestPluginIMDBPath(t *testing.T) {
	want := filepath.Join(PluginDir, "imdb")
	if got := PluginIMDBPath(); got != want {
		t.Fatalf("PluginIMDBPath() = %q, want %q", got, want)
	}
}

func TestIsIMDBPluginAvailable(t *testing.T) {
	dir := t.TempDir()
	if IsIMDBPluginAvailable(dir) {
		t.Fatal("expected false without plugin binary")
	}

	bin := filepath.Join(dir, "imdb")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsIMDBPluginAvailable(dir) {
		t.Fatal("expected true with executable plugin binary")
	}

	if err := os.Chmod(bin, 0o644); err != nil {
		t.Fatal(err)
	}
	if IsIMDBPluginAvailable(dir) {
		t.Fatal("expected false with non-executable plugin file")
	}
}
