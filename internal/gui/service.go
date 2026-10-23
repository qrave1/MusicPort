package gui

import (
	"context"
	"github.com/qrave1/MusicPort/internal/config"
	"github.com/qrave1/MusicPort/internal/spotify"
	"github.com/qrave1/MusicPort/internal/syncer"
	"time"
)

//go:generate go tool minimock -i Service -g
type Service interface {
	Login(context.Context, func(string)) error
	Collections(context.Context) ([]spotify.Collection, error)
	Tracks(context.Context, string) ([]syncer.Track, error)
}
type SpotifyService struct{ Settings config.Config }

func (s SpotifyService) Login(ctx context.Context, show func(string)) error {
	if err := s.Settings.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return spotify.Login(ctx, spotify.Config(s.Settings.Spotify.ClientID, s.Settings.Spotify.RedirectURI), spotify.Keyring{ClientID: s.Settings.Spotify.ClientID}, show)
}
func (s SpotifyService) client(ctx context.Context) (*spotify.Client, error) {
	if err := s.Settings.Validate(); err != nil {
		return nil, err
	}
	client, err := spotify.AuthClient(ctx, spotify.Config(s.Settings.Spotify.ClientID, s.Settings.Spotify.RedirectURI), spotify.Keyring{ClientID: s.Settings.Spotify.ClientID})
	if err != nil {
		return nil, err
	}
	return &spotify.Client{HTTP: client}, nil
}
func (s SpotifyService) Collections(ctx context.Context) ([]spotify.Collection, error) {
	c, err := s.client(ctx)
	if err != nil {
		return nil, err
	}
	return c.Collections(ctx)
}
func (s SpotifyService) Tracks(ctx context.Context, id string) ([]syncer.Track, error) {
	c, err := s.client(ctx)
	if err != nil {
		return nil, err
	}
	return c.Tracks(ctx, id)
}
