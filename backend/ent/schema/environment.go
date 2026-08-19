package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Environment struct {
	ent.Schema
}

func (Environment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDV7).Immutable(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("key").NotEmpty().MaxLen(64).Immutable(),
		field.String("description").Default("").MaxLen(2000),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Environment) Indexes() []ent.Index {
	return []ent.Index{index.Fields("project_id", "key").Unique()}
}

func (Environment) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "environments"}}
}
