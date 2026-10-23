package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qrave1/MusicPort/internal/syncer"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	HTTP    *http.Client
	BaseURL string
}
type Collection struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Snapshot string `json:"snapshot_id"`
}

func (c Client) base() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "https://api.spotify.com/v1"
}
func (c Client) get(ctx context.Context, path string, out any) error {
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", c.base()+path, nil)
		if err != nil {
			return err
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode == 200 {
			return json.Unmarshal(body, out)
		}
		var failure struct {
			Error struct {
				Reason string `json:"reason"`
			}
		}
		_ = json.Unmarshal(body, &failure)
		if failure.Error.Reason == "QUOTA_EXCEEDED" {
			return fmt.Errorf("Spotify quota exceeded")
		}
		if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < 2 {
			delay := time.Duration(attempt+1) * time.Second
			if n, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && n >= 0 {
				delay = time.Duration(n) * time.Second
			}
			if delay > time.Minute {
				return fmt.Errorf("Spotify retry required after %s", delay)
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return fmt.Errorf("Spotify HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("Spotify retries exhausted")
}
func (c Client) Collections(ctx context.Context) ([]Collection, error) {
	result := []Collection{{ID: "liked", Name: "Liked Songs"}}
	for offset := 0; ; {
		var page struct {
			Items []Collection `json:"items"`
			Next  *string      `json:"next"`
		}
		if err := c.get(ctx, fmt.Sprintf("/me/playlists?limit=50&offset=%d", offset), &page); err != nil {
			return nil, err
		}
		result = append(result, page.Items...)
		if page.Next == nil {
			return result, nil
		}
		if len(page.Items) == 0 {
			return nil, fmt.Errorf("empty Spotify page with next link")
		}
		offset += len(page.Items)
	}
}
func (c Client) Tracks(ctx context.Context, id string) ([]syncer.Track, error) {
	before := Collection{}
	endpoint := "/me/tracks"
	if id != "liked" {
		endpoint = "/playlists/" + url.PathEscape(id) + "/items"
		if err := c.get(ctx, "/playlists/"+url.PathEscape(id), &before); err != nil {
			return nil, err
		}
	}
	var result []syncer.Track
	for offset := 0; ; {
		type track struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		}
		var page struct {
			Items []struct {
				Track *track `json:"track"`
				Item  *track `json:"item"`
				Local bool   `json:"is_local"`
			} `json:"items"`
			Next *string `json:"next"`
		}
		if err := c.get(ctx, fmt.Sprintf("%s?limit=50&offset=%d", endpoint, offset), &page); err != nil {
			return nil, err
		}
		for _, entry := range page.Items {
			t := entry.Item
			if t == nil {
				t = entry.Track
			}
			if t != nil && t.ID != "" && !entry.Local && (t.Type == "track" || t.Type == "") {
				result = append(result, syncer.Track{ID: t.ID, Title: t.Name})
			}
		}
		if page.Next == nil {
			break
		}
		if len(page.Items) == 0 {
			return nil, fmt.Errorf("empty Spotify page with next link")
		}
		offset += len(page.Items)
	}
	if id != "liked" {
		var after Collection
		if err := c.get(ctx, "/playlists/"+url.PathEscape(id), &after); err != nil {
			return nil, err
		}
		if before.Snapshot != after.Snapshot {
			return nil, fmt.Errorf("playlist changed during import; retry")
		}
	}
	return result, nil
}
