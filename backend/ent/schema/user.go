package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.String("email").Optional().Nillable().MaxLen(320),
		field.String("display_name").Default("").MaxLen(120),
		createdAtField(),
		updatedAtField(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", OrganisationMembership.Type),
		edge.To("local_credential", LocalCredential.Type).Unique(),
		edge.To("sessions", UserSession.Type),
		edge.To("scheduled_flag_changes", ScheduledFlagChange.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique().StorageKey("users_email_unique"),
	}
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "users",
			Checks: map[string]string{
				"users_email_normalized": "email IS NULL OR (btrim(email) <> '' AND email = lower(btrim(email)))",
			},
		},
	}
}
