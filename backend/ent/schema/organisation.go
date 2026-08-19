package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Organisation struct {
	ent.Schema
}

func (Organisation) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("slug").NotEmpty().MaxLen(63),
		createdAtField(),
		updatedAtField(),
	}
}

func (Organisation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", OrganisationMembership.Type),
		edge.To("projects", Project.Type),
	}
}

func (Organisation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique(),
	}
}

func (Organisation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "organisations",
			Checks: map[string]string{
				"organisations_name_not_blank": "btrim(name) <> ''",
				"organisations_slug_format":    "slug ~ '^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$'",
			},
		},
	}
}
