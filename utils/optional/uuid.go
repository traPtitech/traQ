package optional

import "github.com/gofrs/uuid"

// UUID uuid.NullUUIDのラッパー
type UUID struct {
	uuid.NullUUID
}

func UUIDFrom(v uuid.UUID) UUID {
	return NewUUID(v, true)
}

func NewUUID(v uuid.UUID, valid bool) UUID {
	return UUID{NullUUID: uuid.NullUUID{UUID: v, Valid: valid}}
}
