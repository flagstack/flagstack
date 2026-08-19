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

type ScheduledFlagChange struct{ ent.Schema }

func (ScheduledFlagChange) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.UUID("environment_id", uuid.UUID{}).Immutable(),
		field.UUID("feature_flag_id", uuid.UUID{}).Immutable(),
		field.UUID("created_by_user_id", uuid.UUID{}).Optional().Nillable().Immutable(),
		field.Time("execute_at").Immutable(),
		field.JSON("patch", json.RawMessage{}).Immutable(),
		field.String("status").Default("pending"),
		field.UUID("claim_token", uuid.UUID{}).Optional().Nillable(),
		field.Time("claimed_at").Optional().Nillable(),
		field.Time("executed_at").Optional().Nillable(),
		field.String("last_error").Default("").MaxLen(4000),
		createdAtField(),
		updatedAtField(),
	}
}

func (ScheduledFlagChange) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("scheduled_flag_changes").Field("project_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("environment", Environment.Type).Ref("scheduled_flag_changes").Field("environment_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("feature_flag", FeatureFlag.Type).Ref("scheduled_changes").Field("feature_flag_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("created_by", User.Type).Ref("scheduled_flag_changes").Field("created_by_user_id").Unique().Immutable().Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (ScheduledFlagChange) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "execute_at").StorageKey("scheduled_flag_changes_due_idx"),
		index.Fields("project_id", "created_at").StorageKey("scheduled_flag_changes_project_idx").Annotations(entsql.DescColumns("created_at")),
		index.Fields("environment_id", "feature_flag_id", "execute_at").StorageKey("scheduled_flag_changes_target_idx"),
	}
}

func (ScheduledFlagChange) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "scheduled_flag_changes", Checks: map[string]string{
		"scheduled_flag_changes_status": "status IN ('pending', 'running', 'executed', 'cancelled', 'failed')",
		"scheduled_flag_changes_patch":  "jsonb_typeof(patch) = 'object'",
	}}}
}
