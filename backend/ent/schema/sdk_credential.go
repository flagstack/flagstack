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

type SDKCredential struct{ ent.Schema }

func (SDKCredential) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("organisation_id", uuid.UUID{}).Immutable(),
		field.UUID("project_id", uuid.UUID{}).Immutable(),
		field.UUID("environment_id", uuid.UUID{}).Immutable(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("kind").NotEmpty().Immutable().MaxLen(16),
		field.String("client_key").Default("").Immutable().MaxLen(128),
		field.Bytes("secret_digest").Optional().Immutable(),
		field.Time("revoked_at").Optional().Nillable(),
		createdAtField(), updatedAtField(),
	}
}

func (SDKCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("environment", Environment.Type).
			Ref("sdk_credentials").
			Field("environment_id").
			Unique().
			Required().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (SDKCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organisation_id", "project_id", "environment_id", "id").Unique(),
		index.Fields("client_key").Unique().Annotations(entsql.IndexWhere("client_key <> ''")),
		index.Fields("organisation_id", "project_id", "environment_id", "created_at"),
	}
}

func (SDKCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "sdk_credentials", Checks: map[string]string{
		"sdk_credentials_name_not_blank": "btrim(name) <> ''",
		"sdk_credentials_kind":           "kind IN ('server', 'client')",
		"sdk_credentials_material":       "(kind = 'server' AND secret_digest IS NOT NULL AND octet_length(secret_digest) = 32 AND client_key = '') OR (kind = 'client' AND secret_digest IS NULL AND client_key <> '')",
	}}}
}
