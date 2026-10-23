package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/qrave1/MusicPort/internal/syncer"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Fetch stages data on the destination filesystem and publishes only verified bytes.
func Fetch(ctx context.Context, client *http.Client, root string, s syncer.Source) (string, string, error) {
	return FetchPrepared(ctx, client, root, s, nil)
}

// FetchPrepared persists the final object identity before publishing its bytes.
func FetchPrepared(ctx context.Context, client *http.Client, root string, s syncer.Source, prepare func(string, string) error) (string, string, error) {
	switch s.Format {
	case "mp3", "flac", "ogg", "m4a", "wav":
	default:
		return "", "", fmt.Errorf("unsupported format %q", s.Format)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("audio HTTP %d", resp.StatusCode)
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return "", "", fmt.Errorf("received HTML instead of audio")
	}
	f, err := os.CreateTemp(filepath.Join(root, "audio"), "*.part")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), resp.Body)
	if err != nil {
		return "", "", err
	}
	if n == 0 || (s.Size > 0 && n != s.Size) || (resp.ContentLength >= 0 && n != resp.ContentLength) {
		return "", "", fmt.Errorf("audio size mismatch")
	}
	hash := hex.EncodeToString(h.Sum(nil))
	if s.SHA256 != "" && !strings.EqualFold(hash, s.SHA256) {
		return "", "", fmt.Errorf("audio SHA-256 mismatch")
	}
	if err = f.Sync(); err != nil {
		return "", "", err
	}
	if err = f.Close(); err != nil {
		return "", "", err
	}
	path := filepath.Join("audio", hash+"."+s.Format)
	target := filepath.Join(root, path)
	if prepare != nil {
		if err = prepare(path, hash); err != nil {
			return "", "", err
		}
	}
	// A previously published identical object can be reused after checking its bytes.
	if data, e := os.ReadFile(target); e == nil {
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) == hash {
			return path, hash, nil
		}
	}
	if err = os.Rename(f.Name(), target); err != nil {
		return "", "", err
	}
	return path, hash, nil
}
