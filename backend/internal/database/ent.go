package database

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	switchonyourcodeent "github.com/switchonyourcode/switchonyourcode/backend/ent"
)

func NewEntClient(pool *pgxpool.Pool) *switchonyourcodeent.Client {
	database := stdlib.OpenDBFromPool(pool)
	driver := entsql.OpenDB(dialect.Postgres, database)
	return switchonyourcodeent.NewClient(switchonyourcodeent.Driver(driver))
}
