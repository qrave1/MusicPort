package main

import (
	tea "charm.land/bubbletea/v2"
	"context"

	"flag"
	"fmt"
	"github.com/qrave1/MusicPort/internal/app"
	"github.com/qrave1/MusicPort/internal/config"
	"github.com/qrave1/MusicPort/internal/spotify"
	"github.com/qrave1/MusicPort/internal/store"
	"github.com/qrave1/MusicPort/internal/syncer"
	"github.com/qrave1/MusicPort/internal/tui"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	configPath := flag.String("config", "config.yaml", "YAML configuration file")

	root := flag.String("library", "library", "destination directory")
	dry := flag.Bool("dry-run", false, "show pending downloads")
	ui := flag.Bool("tui", false, "interactive progress and retry")
	login := flag.Bool("login", false, "authorize Spotify")
	list := flag.Bool("list", false, "list Spotify collections")
	collections := flag.String("collections", "liked", "comma-separated playlist IDs or liked")
	clientID := flag.String("client-id", os.Getenv("SPOTIFY_CLIENT_ID"), "Spotify app client ID")
	redirect := flag.String("redirect", "http://127.0.0.1:8888/callback", "registered OAuth redirect")
	flag.Parse()
	settings, loadErr := config.Load(*configPath)
	explicit := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	if loadErr != nil && (!os.IsNotExist(loadErr) || explicit["config"]) {
		return loadErr
	}
	if flag.NFlag() == 0 && os.IsNotExist(loadErr) {
		fmt.Println("MusicPort: start with -login, then -collections liked -tui.")
		flag.PrintDefaults()
		return nil
	}
	if env := os.Getenv("SPOTIFY_CLIENT_ID"); env != "" {
		settings.Spotify.ClientID = env
	}
	if explicit["client-id"] {
		settings.Spotify.ClientID = *clientID
	}
	if explicit["redirect"] {
		settings.Spotify.RedirectURI = *redirect
	}
	if explicit["library"] {
		settings.Library = *root
	}
	if explicit["collections"] {
		settings.Collections = strings.Split(*collections, ",")
	}
	if explicit["tui"] {
		settings.TUI = *ui
	}
	if explicit["dry-run"] {
		settings.DryRun = *dry
	}
	if err := settings.Validate(); err != nil {
		return err
	}
	*clientID = settings.Spotify.ClientID
	*redirect = settings.Spotify.RedirectURI
	*root = settings.Library
	*collections = strings.Join(settings.Collections, ",")
	*ui = settings.TUI
	*dry = settings.DryRun
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cfg := spotify.Config(*clientID, *redirect)
	tokens := spotify.Keyring{ClientID: *clientID}
	if *login {
		if *clientID == "" {
			return fmt.Errorf("--client-id or SPOTIFY_CLIENT_ID required")
		}
		loginCtx, stop := context.WithTimeout(ctx, 5*time.Minute)
		defer stop()
		return spotify.Login(loginCtx, cfg, tokens, func(link string) { fmt.Println("Open this URL in your browser:\n" + link) })
	}
	var catalog *spotify.Client
	if *list || *collections != "" {
		if *clientID == "" {
			return fmt.Errorf("--client-id or SPOTIFY_CLIENT_ID required")
		}
		client, err := spotify.AuthClient(ctx, cfg, tokens)
		if err != nil {
			return err
		}
		catalog = &spotify.Client{HTTP: client}
	}
	if *list {
		items, err := catalog.Collections(ctx)
		if err != nil {
			return err
		}
		for _, c := range items {
			fmt.Printf("%s\t%s\n", c.ID, c.Name)
		}
		return nil
	}
	var tracks []syncer.Track
	for _, id := range strings.Split(*collections, ",") {
		imported, err := catalog.Tracks(ctx, strings.TrimSpace(id))
		if err != nil {
			return err
		}
		tracks = append(tracks, imported...)
	}
	index, err := store.Open(*root)
	if err != nil {
		return err
	}
	defer index.DB.Close()
	runner := app.Runner{Tracks: tracks, Store: index, HTTP: &http.Client{Timeout: 5 * time.Minute}, Dry: *dry}
	if *ui {
		_, err = tea.NewProgram(tui.New(runner), tea.WithContext(ctx)).Run()
		return err
	}
	return runner.Run(ctx, func(s string) { fmt.Println(s) })
}
