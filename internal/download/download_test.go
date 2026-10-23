package download

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/qrave1/MusicPort/internal/syncer"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetch(t *testing.T) {
	for _, tc := range []struct {
		name, body, content, hash string
		status                    int
		fail                      bool
	}{
		{name: "valid", body: "audio", status: 200},
		{name: "empty", status: 200, fail: true},
		{name: "html", body: "html", content: "text/html", status: 200, fail: true},
		{name: "http error", status: 404, fail: true},
		{name: "wrong hash", body: "audio", hash: "bad", status: 200, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.content)
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "audio"), 0755); err != nil {
				t.Fatal(err)
			}
			path, hash, err := Fetch(context.Background(), server.Client(), root, syncer.Source{URL: server.URL, Format: "mp3", SHA256: tc.hash})
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v", err)
			}
			if !tc.fail {
				sum := sha256.Sum256([]byte(tc.body))
				if hash != fmt.Sprintf("%x", sum) {
					t.Fatal(hash)
				}
				if _, err := os.Stat(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			}
			parts, _ := filepath.Glob(filepath.Join(root, "audio", "*.part"))
			if len(parts) != 0 {
				t.Fatal(parts)
			}
		})
	}
}

func TestPreparedAndReuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "audio") }))
	defer server.Close()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "audio"), 0755); err != nil {
		t.Fatal(err)
	}
	src := syncer.Source{URL: server.URL, Format: "mp3"}
	failure := errors.New("journal failed")
	_, _, err := FetchPrepared(context.Background(), server.Client(), root, src, func(string, string) error { return failure })
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	for range 2 {
		_, _, err = FetchPrepared(context.Background(), server.Client(), root, src, func(string, string) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	src.Size = 99
	if _, _, err = Fetch(context.Background(), server.Client(), root, src); err == nil {
		t.Fatal("expected size error")
	}
	src.Format = "../bad"
	if _, _, err = Fetch(context.Background(), server.Client(), root, src); err == nil {
		t.Fatal("expected format error")
	}
	src.Format = "mp3"
	src.URL = ":invalid"
	if _, _, err = Fetch(context.Background(), server.Client(), root, src); err == nil {
		t.Fatal("expected URL error")
	}
	src.URL = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err = Fetch(ctx, server.Client(), root, src); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
