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

type Organisation struct {
	ent.Schema
}

func (Organisation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDV7).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("slug").NotEmpty().MaxLen(63),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Organisation) Indexes() []ent.Index {
	return []ent.Index{index.Fields("slug").Unique()}
}

func (Organisation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "organisations"}}
}
