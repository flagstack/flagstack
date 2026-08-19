package database

import (
	flagstackent "github.com/flagstack/flagstack/backend/ent"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func NewEntClient(pool *pgxpool.Pool) *flagstackent.Client {
	database := stdlib.OpenDBFromPool(pool)
	driver := entsql.OpenDB(dialect.Postgres, database)
	return flagstackent.NewClient(flagstackent.Driver(driver))
}
