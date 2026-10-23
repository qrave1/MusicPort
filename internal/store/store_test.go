//go:build integration

package store

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/qrave1/MusicPort/internal/syncer"
	"os"
	"path/filepath"
	"testing"
)

func TestRecover(t *testing.T) {
	for _, tc := range []struct {
		name          string
		exists, valid bool
	}{{"published", true, true}, {"corrupt", true, false}, {"missing", false, false}} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer s.DB.Close()
			ctx := context.Background()
			src := syncer.Source{Provider: "test", ID: "1", Format: "mp3"}
			path := filepath.Join("audio", "test.mp3")
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte("audio")))
			if tc.exists {
				data := []byte("audio")
				if !tc.valid {
					data = []byte("corrupt")
				}
				if err = os.WriteFile(filepath.Join(s.Root, path), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.Prepare(ctx, src, path, hash); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if err = s.Recover(ctx); err != nil {
					t.Fatal(err)
				}
			}
			present, err := s.Present(ctx, src)
			if err != nil || present != tc.valid {
				t.Fatalf("present=%v err=%v", present, err)
			}
		})
	}
}
