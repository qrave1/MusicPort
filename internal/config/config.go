// Package config loads and validates MusicPort settings.
package config

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Spotify struct {
	ClientID    string `yaml:"client_id"`
	RedirectURI string `yaml:"redirect_uri"`
}
type Config struct {
	Spotify     Spotify  `yaml:"spotify"`
	Library     string   `yaml:"library"`
	Collections []string `yaml:"collections"`
	// TUI is accepted for compatibility with pre-GUI configuration files and ignored.
	TUI         bool     `yaml:"tui"`
	DryRun      bool     `yaml:"dry_run"`
}

func Default() Config {
	return Config{Spotify: Spotify{RedirectURI: "http://127.0.0.1:8888/callback"}, Library: "library", Collections: []string{"liked"}}
}

// Load accepts exactly one YAML document and rejects misspelled setting names.
func Load(path string) (Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err = decoder.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	var extra any
	if err = decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("config must contain exactly one YAML document")
	}
	return cfg, nil
}

// Validate runs after file, environment and explicit CLI overrides are applied.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Spotify.ClientID) == "" {
		return errors.New("Spotify client ID required: set spotify.client_id in config.yaml, SPOTIFY_CLIENT_ID, or -client-id")
	}
	if strings.TrimSpace(c.Library) == "" {
		return errors.New("library must not be empty")
	}
	if len(c.Collections) == 0 {
		return errors.New("collections must not be empty")
	}
	for _, id := range c.Collections {
		if strings.TrimSpace(id) == "" {
			return errors.New("collection ID must not be empty")
		}
	}
	u, err := url.Parse(c.Spotify.RedirectURI)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Path == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return errors.New("spotify.redirect_uri must be http://127.0.0.1:<port>/<path>")
	}
	return nil
}
