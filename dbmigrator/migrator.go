package dbmigrator

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxdrv "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog"
)

type Source struct {
	FS   embed.FS
	Path string
}

type MigrationResult struct {
	PrevVersion uint
	NewVersion  uint
}

// multiFS merges multiple fs.FS instances into a single virtual root directory.
type multiFS struct {
	subs []fs.FS
}

func newMultiFS(sources []Source) (fs.FS, error) {
	subs := make([]fs.FS, len(sources))
	for i, s := range sources {
		sub, err := fs.Sub(s.FS, s.Path)
		if err != nil {
			return nil, fmt.Errorf("source %q: %w", s.Path, err)
		}
		subs[i] = sub
	}
	return &multiFS{subs: subs}, nil
}

func (m *multiFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &multiDir{subs: m.subs}, nil
	}
	for _, sub := range m.subs {
		f, err := sub.Open(name)
		if err == nil {
			return f, nil
		}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// multiDir is a virtual directory merging entries from all sub-FSes.
type multiDir struct {
	subs    []fs.FS
	entries []fs.DirEntry
	pos     int
}

func (d *multiDir) Stat() (fs.FileInfo, error) { return &multiDirInfo{}, nil }
func (d *multiDir) Read([]byte) (int, error)    { return 0, fmt.Errorf("is a directory") }
func (d *multiDir) Close() error                { return nil }

func (d *multiDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.entries == nil {
		seen := map[string]bool{}
		for _, sub := range d.subs {
			entries, err := fs.ReadDir(sub, ".")
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				if !seen[e.Name()] {
					seen[e.Name()] = true
					d.entries = append(d.entries, e)
				}
			}
		}
	}

	if n <= 0 {
		rest := d.entries[d.pos:]
		d.pos = len(d.entries)
		return rest, nil
	}

	if d.pos >= len(d.entries) {
		return nil, io.EOF
	}

	end := d.pos + n
	if end > len(d.entries) {
		end = len(d.entries)
	}
	entries := d.entries[d.pos:end]
	d.pos = end
	return entries, nil
}

type multiDirInfo struct{}

func (multiDirInfo) Name() string      { return "." }
func (multiDirInfo) Size() int64       { return 0 }
func (multiDirInfo) Mode() fs.FileMode { return fs.ModeDir | 0o555 }
func (multiDirInfo) ModTime() time.Time { return time.Time{} }
func (multiDirInfo) IsDir() bool       { return true }
func (multiDirInfo) Sys() any          { return nil }

func RunPostgres(url string, l zerolog.Logger, sources ...Source) (*MigrationResult, error) {
	merged, err := newMultiFS(sources)
	if err != nil {
		return nil, fmt.Errorf("merge migration sources: %w", err)
	}

	srcDrv, err := iofs.New(merged, ".")
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}

	dbDrv, err := (&pgxdrv.Postgres{}).Open(url)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err := dbDrv.Close(); err != nil {
			l.Error().Err(err).Msg("failed to close db driver")
		}
	}()

	mig, err := migrate.NewWithInstance("iofs", srcDrv, "postgres", dbDrv)
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}

	prevVer, dirty, err := mig.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("get db version before migration: %w", err)
	}

	if dirty {
		return nil, fmt.Errorf("current version is dirty: %d", prevVer)
	}

	err = mig.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	newVer, _, err := mig.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("get db version after migration: %w", err)
	}

	return &MigrationResult{
		PrevVersion: prevVer,
		NewVersion:  newVer,
	}, nil
}
