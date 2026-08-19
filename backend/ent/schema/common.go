package schema

import "github.com/google/uuid"

func newUUIDV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}
