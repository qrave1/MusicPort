package app

import (
	"context"
	"fmt"
	"github.com/gofrs/flock"
	"github.com/qrave1/MusicPort/internal/download"
	"github.com/qrave1/MusicPort/internal/store"
	"github.com/qrave1/MusicPort/internal/syncer"
	"net/http"
	"path/filepath"
)

type Runner struct {
	Tracks []syncer.Track
	Store  *store.Store
	HTTP   *http.Client
	Dry    bool
}

func (r Runner) Run(ctx context.Context, report func(string)) error {
	lock := flock.New(filepath.Join(r.Store.Root, "sync.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("another synchronization is running")
	}
	defer lock.Unlock()
	if err := r.Store.Recover(ctx); err != nil {
		return err
	}
	plan, err := syncer.Build(ctx, r.Store, r.Tracks)
	if err != nil {
		return err
	}
	report(fmt.Sprintf("new=%d present=%d unresolved=%d", len(plan.New), plan.Present, plan.Unresolved))
	for _, track := range r.Tracks {
		if track.Source == nil {
			report(fmt.Sprintf("%s [%s] — audio source not connected", track.Title, track.ID))
		}
	}
	if r.Dry {
		return nil
	}
	var failed int
	for _, src := range plan.New {
		path, hash, e := download.FetchPrepared(ctx, r.HTTP, r.Store.Root, src, func(path, hash string) error { return r.Store.Prepare(ctx, src, path, hash) })
		if e == nil {
			e = r.Store.Save(ctx, src, path, hash)
		}
		if e != nil {
			failed++
			report(fmt.Sprintf("%s: %v", src.ID, e))
		} else {
			report(path)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d downloads failed", failed)
	}
	return nil
}
