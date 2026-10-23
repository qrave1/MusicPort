package spotify

import (
	"context"
	"errors"
	"fmt"
	"github.com/gojuno/minimock/v3"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTracks(t *testing.T) {
	for _, tc := range []struct {
		name          string
		changed, fail bool
	}{{name: "pages"}, {name: "snapshot changed", changed: true}, {name: "second page failed", fail: true}} {
		t.Run(tc.name, func(t *testing.T) {
			snapshots := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/playlists/p" {
					snapshots++
					snapshot := "one"
					if tc.changed && snapshots > 1 {
						snapshot = "two"
					}
					fmt.Fprintf(w, `{"snapshot_id":%q}`, snapshot)
					return
				}
				if r.URL.Query().Get("offset") == "0" {
					fmt.Fprint(w, `{"items":[{"item":{"id":"a","name":"A","type":"track"}}],"next":"page2"}`)
					return
				}
				if tc.fail {
					w.WriteHeader(403)
					return
				}
				fmt.Fprint(w, `{"items":[{"item":{"id":"b","name":"B","type":"track"}}],"next":null}`)
			}))
			defer server.Close()
			got, err := (Client{HTTP: server.Client(), BaseURL: server.URL}).Tracks(context.Background(), "p")
			if tc.changed || tc.fail {
				if err == nil || got != nil {
					t.Fatalf("got=%v err=%v", got, err)
				}
				return
			}
			if err != nil || len(got) != 2 || got[1].ID != "b" {
				t.Fatalf("got=%v err=%v", got, err)
			}
		})
	}
}
func TestQuotaDoesNotRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(429)
		fmt.Fprint(w, `{"error":{"reason":"QUOTA_EXCEEDED"}}`)
	}))
	defer server.Close()
	_, err := (Client{HTTP: server.Client(), BaseURL: server.URL}).Collections(context.Background())
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
func TestAuthClientMissingToken(t *testing.T) {
	missing := errors.New("missing")
	store := NewTokenStoreMock(minimock.NewController(t)).LoadMock.Return(nil, missing)
	_, err := AuthClient(context.Background(), Config("id", ""), store)
	if !errors.Is(err, missing) {
		t.Fatal(err)
	}
}
func TestAuthClientValidToken(t *testing.T) {
	token := &oauth2.Token{AccessToken: "access", Expiry: time.Now().Add(time.Hour)}
	store := NewTokenStoreMock(minimock.NewController(t)).LoadMock.Return(token, nil)
	client, err := AuthClient(context.Background(), Config("id", ""), store)
	if err != nil || client == nil {
		t.Fatal(err)
	}
}
