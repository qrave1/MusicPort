package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/qrave1/MusicPort/internal/syncer"
	"io"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

type Store struct {
	DB   *sql.DB
	Root string
}

func Open(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "audio"), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(root, "index.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files(provider TEXT, source_id TEXT, format TEXT, path TEXT, hash TEXT, PRIMARY KEY(provider,source_id,format))`)
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pending(provider TEXT, source_id TEXT, format TEXT, path TEXT, hash TEXT, PRIMARY KEY(provider,source_id,format))`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db, root}, nil
}
func (s *Store) Present(ctx context.Context, src syncer.Source) (bool, error) {
	var path, hash string
	err := s.DB.QueryRowContext(ctx, `SELECT path,hash FROM files WHERE provider=? AND source_id=? AND format=?`, src.Provider, src.ID, src.Format).Scan(&path, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	f, err := os.Open(filepath.Join(s.Root, path))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == hash, nil
}
func (s *Store) Save(ctx context.Context, src syncer.Source, path, hash string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO files VALUES(?,?,?,?,?) ON CONFLICT(provider,source_id,format) DO UPDATE SET path=excluded.path,hash=excluded.hash`, src.Provider, src.ID, src.Format, path, hash)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM pending WHERE provider=? AND source_id=? AND format=?`, src.Provider, src.ID, src.Format); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Prepare(ctx context.Context, src syncer.Source, path, hash string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO pending VALUES(?,?,?,?,?) ON CONFLICT(provider,source_id,format) DO UPDATE SET path=excluded.path,hash=excluded.hash`, src.Provider, src.ID, src.Format, path, hash)
	return err
}

// Recover adopts verified objects published before a previous process could commit.
func (s *Store) Recover(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT provider,source_id,format,path,hash FROM pending`)
	if err != nil {
		return err
	}
	type pending struct {
		src        syncer.Source
		path, hash string
	}
	var jobs []pending
	for rows.Next() {
		var j pending
		if err = rows.Scan(&j.src.Provider, &j.src.ID, &j.src.Format, &j.path, &j.hash); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, j)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if !filepath.IsLocal(j.path) {
			return errors.New("invalid pending path")
		}
		f, e := os.Open(filepath.Join(s.Root, j.path))
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil {
			return e
		}
		if hex.EncodeToString(h.Sum(nil)) == j.hash {
			if e = s.Save(ctx, j.src, j.path, j.hash); e != nil {
				return e
			}
		}
	}
	return nil
}
