package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type LocalCredential struct{ ent.Schema }

func (LocalCredential) Fields() []ent.Field {
	return []ent.Field{
		uuidIDField(),
		field.UUID("user_id", uuid.UUID{}).Unique().Immutable(),
		field.String("password_hash").NotEmpty(),
		createdAtField(),
		field.Time("password_changed_at").Default(time.Now).Annotations(entsql.DefaultExpr("CURRENT_TIMESTAMP")),
	}
}

func (LocalCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("local_credential").Field("user_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (LocalCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "local_credentials", Checks: map[string]string{
		"local_credentials_password_hash_not_blank": "btrim(password_hash) <> ''",
	}}}
}
