package postgres

import (
	"context"
	"database/sql"

	"git.blinkpay.vn/software/authix/internal/cache"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type Persister struct {
	conn    *bun.DB
	pgxPool *pgxpool.Pool
	cache   *cache.PermissionCache
}

func NewPersister(ctx context.Context, dsn string) (*Persister, error) {
	sqldb, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	conn := bun.NewDB(sqldb, pgdialect.New())

	pgxpol, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	p := &Persister{
		conn:    conn,
		pgxPool: pgxpol,
		cache:   cache.NewPermissionCache(),
	}
	go p.listenForInvalidation(ctx)
	// return &Persister{conn: conn}, nil

	return p, nil
}

func (p *Persister) DB() *bun.DB {
	return p.conn
}
