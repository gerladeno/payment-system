package pgStore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"payment-system/pkg"
	"time"
)

const txRetries = 3

//go:embed migrations
var migrations embed.FS

type PG struct {
	db  *sqlx.DB
	log *logrus.Logger
}

func GetPGStore(log *logrus.Logger, dsn string) (*PG, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &PG{
		db:  db,
		log: log,
	}, nil
}

func (pg *PG) DC() error {
	return pg.db.Close()
}

func (pg *PG) Migrate(direction migrate.MigrationDirection) error {
	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, err := migrations.ReadDir(path)
			if err != nil {
				return nil, err
			}
			entries := make([]string, 0)
			for _, e := range dirEntry {
				entries = append(entries, e.Name())
			}
			return entries, nil
		}
	}()
	asset := migrate.AssetMigrationSource{
		Asset:    migrations.ReadFile,
		AssetDir: assetDir,
		Dir:      "migrations",
	}
	_, err := migrate.Exec(pg.db.DB, "postgres", asset, direction)
	return err
}

func (pg *PG) tx(ctx context.Context, method string, fn func(tx *sqlx.Tx) error) error {
	var tx *sqlx.Tx
	var err error
	started := time.Now()
	for i := 0; i < txRetries; i++ {
		if tx, err = pg.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted}); err != nil {
			pkg.MetricDBErrors.WithLabelValues(method).Inc()
			continue
		}
		if err = fn(tx); err != nil {
			var errDup pkg.ErrDuplicateAction
			if errors.As(err, &errDup) || err == pkg.ErrInsufficientFunds {
				return err
			}
			pkg.MetricDBErrors.WithLabelValues(method).Inc()
			_ = tx.Rollback()
			continue
		}
		select {
		case <-ctx.Done():
			_ = tx.Rollback()
			return ctx.Err()
		default:
		}
		if err = tx.Commit(); err != nil {
			pkg.MetricDBErrors.WithLabelValues(method).Inc()
			_ = tx.Rollback()
			continue
		}
		pkg.MetricDBTime.WithLabelValues(method).Observe(time.Since(started).Seconds())
		return nil
	}
	pkg.MetricDBTime.WithLabelValues(method).Observe(time.Since(started).Seconds())
	return err
}

// Truncate for tests
func (pg *PG) Truncate() error {
	_, err := pg.db.Exec("TRUNCATE TABLE wallet;")
	if err != nil {
		return err
	}
	_, err = pg.db.Exec("TRUNCATE TABLE transaction;")
	if err != nil {
		return err
	}
	return nil
}
