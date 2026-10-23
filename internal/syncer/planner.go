package syncer

import "context"

//go:generate go tool minimock -i Index -g

type Source struct {
	Provider string `json:"provider"`
	ID       string `json:"source_id"`
	URL      string `json:"download_url"`
	Format   string `json:"format"`
	SHA256   string `json:"expected_sha256,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type Track struct {
	ID     string  `json:"spotify_id"`
	Title  string  `json:"title"`
	Source *Source `json:"source,omitempty"`
}

type Index interface {
	Present(context.Context, Source) (bool, error)
}

type Plan struct {
	New                 []Source
	Present, Unresolved int
}

// Build schedules each stable source identity once, regardless of playlist membership.
func Build(ctx context.Context, index Index, tracks []Track) (Plan, error) {
	var plan Plan
	seen := make(map[[3]string]bool)
	for _, track := range tracks {
		if err := ctx.Err(); err != nil {
			return Plan{}, err
		}
		if track.Source == nil {
			plan.Unresolved++
			continue
		}
		s := *track.Source
		key := [3]string{s.Provider, s.ID, s.Format}
		if seen[key] {
			continue
		}
		seen[key] = true
		present, err := index.Present(ctx, s)
		if err != nil {
			return Plan{}, err
		}
		if present {
			plan.Present++
		} else {
			plan.New = append(plan.New, s)
		}
	}
	return plan, nil
}
