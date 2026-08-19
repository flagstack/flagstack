package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type FeatureFlag struct{ ent.Schema }

func (FeatureFlag) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("key").NotEmpty().MaxLen(128).Immutable(),
		field.String("description").Default("").MaxLen(2000),
		field.String("kind").NotEmpty().Immutable(),
		field.JSON("default_value", json.RawMessage{}),
		field.Time("archived_at").Optional().Nillable(),
		createdAtField(), updatedAtField(),
	}
}

func (FeatureFlag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("feature_flags").Field("project_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("environment_configs", EnvironmentFlagConfig.Type),
	}
}

func (FeatureFlag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organisation_id", "project_id", "id").Unique(),
		index.Fields("project_id", "key").Unique(),
		index.Fields("project_id", "created_at").StorageKey("feature_flags_project_active_idx").Annotations(entsql.DescColumns("created_at"), entsql.IndexWhere("archived_at IS NULL")),
	}
}

func (FeatureFlag) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "feature_flags", Checks: map[string]string{
		"feature_flags_name_not_blank":        "btrim(name) <> ''",
		"feature_flags_key_not_blank":         "btrim(key) <> ''",
		"feature_flags_kind":                  "kind IN ('boolean', 'string', 'number', 'json')",
		"feature_flags_default_value_type":    "kind = 'json' OR (kind = 'boolean' AND jsonb_typeof(default_value) = 'boolean') OR (kind = 'string' AND jsonb_typeof(default_value) = 'string') OR (kind = 'number' AND jsonb_typeof(default_value) = 'number')",
	}}}
}
