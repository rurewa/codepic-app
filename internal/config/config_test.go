package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rurewa/codepic-app/internal/render"
)

func TestLoad_MissingFileReturnsDefault(t *testing.T) {
	store := NewFileStoreAt(filepath.Join(t.TempDir(), "config.json"))
	c, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.LastLanguage != "auto" {
		t.Errorf("LastLanguage = %q, want auto", c.LastLanguage)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	store := NewFileStoreAt(path)

	orig := DefaultConfig()
	orig.LastLanguage = "go"
	orig.LastStyle.ThemeID = "nord"
	orig.LastStyle.Padding = 64
	orig, _ = orig.AddPreset("my-dark", orig.LastStyle)

	if err := store.Save(orig); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("round-trip mismatch:\n got: %+v\nwant: %+v", got, orig)
	}
}

func TestLoad_CorruptFileReturnsDefaultAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewFileStoreAt(path)
	c, err := store.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if c.LastLanguage != "auto" {
		t.Errorf("expected default config, got %+v", c)
	}
}

func TestPresets_AddGetRemove(t *testing.T) {
	c := DefaultConfig()
	style := render.DefaultStyle()
	style.ThemeID = "monokai"

	c, err := c.AddPreset("mk", style)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(c.Presets) != 1 {
		t.Fatalf("presets = %d, want 1", len(c.Presets))
	}

	got, err := c.GetPreset("mk")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ThemeID != "monokai" {
		t.Errorf("theme = %q, want monokai", got.ThemeID)
	}

	c, err = c.RemovePreset("mk")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(c.Presets) != 0 {
		t.Errorf("presets = %d, want 0", len(c.Presets))
	}
}

func TestPresets_AddEmptyNameFails(t *testing.T) {
	if _, err := DefaultConfig().AddPreset("", render.DefaultStyle()); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestPresets_AddDuplicateReplaces(t *testing.T) {
	c := DefaultConfig()
	s1 := render.DefaultStyle()
	s1.ThemeID = "nord"
	s2 := render.DefaultStyle()
	s2.ThemeID = "dracula"

	c, _ = c.AddPreset("x", s1)
	c, _ = c.AddPreset("x", s2)

	if len(c.Presets) != 1 {
		t.Fatalf("presets = %d, want 1", len(c.Presets))
	}
	got, _ := c.GetPreset("x")
	if got.ThemeID != "dracula" {
		t.Errorf("theme = %q, want dracula", got.ThemeID)
	}
}