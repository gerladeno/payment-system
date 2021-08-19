package pgStore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"payment-system/pkg"
	"time"
)

const txRetries = 3
const maxConnectionsPools = 90

//go:embed migrations
var migrations embed.FS

type PG struct {
	db  *pgxpool.Pool
	dsn string
	log *logrus.Logger
}

func GetPGStore(ctx context.Context, log *logrus.Logger, dsn string) (*PG, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.ConnConfig.PreferSimpleProtocol = true
	config.MaxConns = maxConnectionsPools
	db, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(ctx); err != nil {
		return nil, err
	}
	return &PG{
		db:  db,
		dsn: dsn,
		log: log,
	}, nil
}

func (pg *PG) DC() {
	pg.db.Close()
}

func (pg *PG) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", pg.dsn)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			pg.log.Error("err closing migration connection")
		}
	}()
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
	_, err = migrate.Exec(conn, "postgres", asset, direction)
	return err
}

func (pg *PG) tx(ctx context.Context, method string, fn func(tx pgx.Tx) error) error {
	var err error
	started := time.Now()
	for i := 0; i < txRetries; i++ {
		var tx pgx.Tx
		if tx, err = pg.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}); err != nil {
			pkg.MetricDBErrors.WithLabelValues(method).Inc()
			continue
		}
		if err = fn(tx); err != nil {
			_ = tx.Rollback(ctx)
			var errDup pkg.ErrDuplicateAction
			if errors.As(err, &errDup) || err == pkg.ErrInsufficientFunds {
				return err
			}
			pkg.MetricDBErrors.WithLabelValues(method).Inc()
			continue
		}
		select {
		case <-ctx.Done():
			_ = tx.Rollback(ctx)
			return ctx.Err()
		default:
		}
		if err = tx.Commit(ctx); err != nil {
			pkg.MetricDBErrors.WithLabelValues(method).Inc()
			_ = tx.Rollback(ctx)
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
	_, err := pg.db.Exec(context.Background(), "TRUNCATE TABLE wallet;")
	if err != nil {
		return err
	}
	_, err = pg.db.Exec(context.Background(), "TRUNCATE TABLE transaction;")
	return err
}
