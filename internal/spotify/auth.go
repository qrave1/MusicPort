package spotify

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

//go:generate go tool minimock -i TokenStore -g
type TokenStore interface {
	Load() (*oauth2.Token, error)
	Save(*oauth2.Token) error
}
type Keyring struct{ ClientID string }

func (k Keyring) Load() (*oauth2.Token, error) {
	data, err := keyring.Get("musicport", k.ClientID)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	err = json.Unmarshal([]byte(data), &token)
	return &token, err
}
func (k Keyring) Save(t *oauth2.Token) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return keyring.Set("musicport", k.ClientID, string(data))
}
func Config(clientID, redirect string) oauth2.Config {
	return oauth2.Config{ClientID: clientID, RedirectURL: redirect, Scopes: []string{"user-library-read", "playlist-read-private", "playlist-read-collaborative"}, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.spotify.com/authorize", TokenURL: "https://accounts.spotify.com/api/token", AuthStyle: oauth2.AuthStyleInParams}}
}

// Login binds only the registered loopback callback and validates OAuth state.
func Login(ctx context.Context, cfg oauth2.Config, store TokenStore, show func(string)) error {
	u, err := url.Parse(cfg.RedirectURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		return errors.New("redirect must be http://127.0.0.1:<port>/<path>")
	}
	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return err
	}
	defer listener.Close()
	state := rand.Text()
	verifier := oauth2.GenerateVerifier()
	codes := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid OAuth state", 400)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Authorization was not granted", 400)
			return
		}
		select {
		case codes <- code:
			fmt.Fprint(w, "MusicPort: authorization received. You can close this tab.")
		default:
			http.Error(w, "Already received", 409)
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	ended := make(chan error, 1)
	go func() { ended <- server.Serve(listener) }()
	defer func() { server.Close(); <-ended }()
	show(cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case code := <-codes:
		token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		if err != nil {
			return err
		}
		return store.Save(token)
	}
}

type persistedSource struct {
	source   oauth2.TokenSource
	store    TokenStore
	previous string
}

func (s *persistedSource) Token() (*oauth2.Token, error) {
	t, err := s.source.Token()
	if err != nil {
		return nil, err
	}
	if t.AccessToken != s.previous {
		if err = s.store.Save(t); err != nil {
			return nil, fmt.Errorf("save refreshed token: %w", err)
		}
		s.previous = t.AccessToken
	}
	return t, nil
}
func AuthClient(ctx context.Context, cfg oauth2.Config, store TokenStore) (*http.Client, error) {
	token, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("Spotify login required: %w", err)
	}
	source := &persistedSource{source: cfg.TokenSource(ctx, token), store: store, previous: token.AccessToken}
	client := oauth2.NewClient(ctx, oauth2.ReuseTokenSource(token, source))
	client.Timeout = 45 * time.Second
	return client, nil
}
