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

type EnvironmentFlagConfig struct{ ent.Schema }

func (EnvironmentFlagConfig) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.UUID("environment_id", uuid.UUID{}).Immutable(),
		field.UUID("feature_flag_id", uuid.UUID{}).Immutable(),
		field.Bool("enabled").Default(false),
		field.JSON("value", json.RawMessage{}).Optional(),
		field.Int64("revision").Default(1),
		createdAtField(),
		updatedAtField(),
	}
}

func (EnvironmentFlagConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("environment", Environment.Type).Ref("flag_configs").Field("environment_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("feature_flag", FeatureFlag.Type).Ref("environment_configs").Field("feature_flag_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (EnvironmentFlagConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("environment_id", "feature_flag_id").Unique(),
		index.Fields("feature_flag_id", "environment_id").StorageKey("environment_flag_configs_flag_idx"),
	}
}

func (EnvironmentFlagConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "environment_flag_configs", Checks: map[string]string{
			"environment_flag_configs_revision_positive": "revision > 0",
		}},
	}
}
