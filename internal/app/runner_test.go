//go:build integration

package app

import (
	"context"
	"fmt"
	"github.com/qrave1/MusicPort/internal/store"
	"github.com/qrave1/MusicPort/internal/syncer"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRepeatedSync(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, "audio") }))
	defer server.Close()
	index, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer index.DB.Close()
	src := syncer.Source{Provider: "test", ID: "one", URL: server.URL, Format: "mp3"}
	runner := Runner{Store: index, HTTP: server.Client(), Tracks: []syncer.Track{{Source: &src}, {Source: &src}}}
	for range 2 {
		if err = runner.Run(context.Background(), func(string) {}); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("downloads=%d", calls)
	}
	files, err := filepath.Glob(filepath.Join(index.Root, "audio", "*.mp3"))
	if err != nil || len(files) != 1 {
		t.Fatal(files, err)
	}
	if err = os.Remove(files[0]); err != nil {
		t.Fatal(err)
	}
	if err = runner.Run(context.Background(), func(string) {}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("downloads=%d", calls)
	}
}
