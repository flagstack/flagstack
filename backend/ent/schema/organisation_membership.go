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

type OrganisationMembership struct{ ent.Schema }

func (OrganisationMembership) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("user_id", uuid.UUID{}).Immutable(),
		field.String("role").NotEmpty(),
		createdAtField(),
		updatedAtField(),
	}
}

func (OrganisationMembership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organisation", Organisation.Type).Ref("memberships").Field("organisation_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("user", User.Type).Ref("memberships").Field("user_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (OrganisationMembership) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "organisation_id").StorageKey("organisation_memberships_user_idx")}
}

func (OrganisationMembership) Annotations() []schema.Annotation {
	return []schema.Annotation{
		field.ID("organisation_id", "user_id"),
		entsql.Annotation{Table: "organisation_memberships", Checks: map[string]string{"organisation_memberships_role": "role IN ('owner', 'admin', 'developer', 'viewer')"}},
	}
}
