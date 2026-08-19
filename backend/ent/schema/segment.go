package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/flagstack/flagstack/backend/internal/evaluation"
	"github.com/google/uuid"
)

type Segment struct{ ent.Schema }

func (Segment) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("key").NotEmpty().MaxLen(128).Immutable(),
		field.String("description").Default("").MaxLen(2000),
		field.String("match").Default(string(evaluation.MatchAll)),
		field.JSON("conditions", []evaluation.Condition{}),
		field.Time("archived_at").Optional().Nillable(),
		createdAtField(),
		updatedAtField(),
	}
}

func (Segment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("segments").Field("project_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Segment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organisation_id", "project_id", "id").Unique(),
		index.Fields("project_id", "key").Unique(),
		index.Fields("project_id", "created_at").StorageKey("segments_project_active_idx").Annotations(entsql.DescColumns("created_at"), entsql.IndexWhere("archived_at IS NULL")),
	}
}

func (Segment) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "segments", Checks: map[string]string{
		"segments_name_not_blank": "btrim(name) <> ''",
		"segments_key_not_blank":  "btrim(key) <> ''",
		"segments_match":          "match IN ('all', 'any')",
	}}}
}
