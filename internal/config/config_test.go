package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		fail       bool
	}{
		{"defaults", "", false},
		{"valid", "spotify:\n  client_id: test\nlibrary: music\ncollections: [liked, playlist]\ntui: true", false},
		{"unknown", "typo: true", true},
		{"malformed", "collections: [", true},
		{"wrong type", "tui: [true]", true},
		{"multiple documents", "tui: true\n---\ntui: false", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := Load(path)
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v", err)
			}
			if tc.fail {
				return
			}
			if got.Spotify.RedirectURI != Default().Spotify.RedirectURI {
				t.Fatal(got)
			}
			if tc.name == "valid" && (got.Spotify.ClientID != "test" || got.Library != "music" || len(got.Collections) != 2 || !got.TUI) {
				t.Fatal(got)
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Config)
	}{
		{"client", func(c *Config) { c.Spotify.ClientID = "" }},
		{"library", func(c *Config) { c.Library = " " }},
		{"collections", func(c *Config) { c.Collections = nil }},
		{"collection", func(c *Config) { c.Collections = []string{" "} }},
		{"redirect", func(c *Config) { c.Spotify.RedirectURI = "http://0.0.0.0:8888/callback" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			c.Spotify.ClientID = "test"
			tc.change(&c)
			if c.Validate() == nil {
				t.Fatal("expected error")
			}
		})
	}
	c := Default()
	c.Spotify.ClientID = "test"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
