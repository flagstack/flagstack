package database

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func NewEntClient(pool *pgxpool.Pool) *switchonyourcodeent.Client {
	database := stdlib.OpenDBFromPool(pool)
	driver := entsql.OpenDB(dialect.Postgres, database)
	return switchonyourcodeent.NewClient(switchonyourcodeent.Driver(driver))
}
