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

type UserSession struct{ ent.Schema }

func (UserSession) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("user_id", uuid.UUID{}).Immutable(),
		field.Bytes("token_hash").Unique().Immutable(),
		field.Bytes("csrf_hash").Immutable(),
		field.Time("expires_at").Immutable(),
		createdAtField(),
	}
}

func (UserSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").Field("user_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (UserSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "expires_at").StorageKey("user_sessions_user_expiry_idx").Annotations(entsql.DescColumns("expires_at")),
		index.Fields("expires_at").StorageKey("user_sessions_expiry_idx"),
	}
}

func (UserSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "user_sessions", Checks: map[string]string{
		"user_sessions_token_hash_length":     "octet_length(token_hash) = 32",
		"user_sessions_csrf_hash_length":      "octet_length(csrf_hash) = 32",
		"user_sessions_expiry_after_creation": "expires_at > created_at",
	}}}
}
