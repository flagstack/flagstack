package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

func newUUIDV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}

func uuidIDField() ent.Field {
	return field.UUID("id", uuid.UUID{}).
		Default(newUUIDV7).
		Immutable().
		Annotations(entsql.DefaultExpr("uuidv7()"))
}

func createdAtField() ent.Field {
	return field.Time("created_at").
		Default(time.Now).
		Immutable().
		Annotations(entsql.DefaultExpr("CURRENT_TIMESTAMP"))
}

func updatedAtField() ent.Field {
	return field.Time("updated_at").
		Default(time.Now).
		UpdateDefault(time.Now).
		Annotations(entsql.DefaultExpr("CURRENT_TIMESTAMP"))
}
