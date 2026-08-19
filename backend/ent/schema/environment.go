package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Environment struct {
	ent.Schema
}

func (Environment) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("key").NotEmpty().MaxLen(64).Immutable(),
		field.String("description").Default("").MaxLen(2000),
		createdAtField(),
		updatedAtField(),
	}
}

func (Environment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).
			Field("project_id").
			Unique().
			Required().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("flag_configs", EnvironmentFlagConfig.Type).Ref("environment"),
	}
}

func (Environment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organisation_id", "project_id", "id").Unique(),
		index.Fields("project_id", "key").Unique(),
	}
}

func (Environment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "environments",
			Checks: map[string]string{
				"environments_name_not_blank": "btrim(name) <> ''",
				"environments_key_not_blank":  "btrim(key) <> ''",
			},
		},
	}
}
