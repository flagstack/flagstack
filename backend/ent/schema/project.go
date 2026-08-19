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

type Project struct{ ent.Schema }

func (Project) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("key").NotEmpty().MaxLen(64).Immutable(),
		field.String("description").Default("").MaxLen(2000),
		field.Time("archived_at").Optional().Nillable(),
		createdAtField(), updatedAtField(),
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organisation", Organisation.Type).Ref("projects").Field("organisation_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("environments", Environment.Type),
		edge.To("feature_flags", FeatureFlag.Type),
	}
}

func (Project) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organisation_id", "id").Unique(),
		index.Fields("organisation_id", "key").Unique(),
		index.Fields("organisation_id", "created_at").StorageKey("projects_organisation_active_idx").Annotations(entsql.DescColumns("created_at"), entsql.IndexWhere("archived_at IS NULL")),
	}
}

func (Project) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "projects", Checks: map[string]string{
		"projects_name_not_blank": "btrim(name) <> ''",
		"projects_key_not_blank":  "btrim(key) <> ''",
	}}}
}
