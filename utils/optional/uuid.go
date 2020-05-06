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

func (u *UUID) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		u.Valid = false
		return nil
	}
	if err := u.UUID.UnmarshalText(text); err != nil {
		return err
	}
	u.Valid = true
	return nil
}

func (u UUID) MarshalText() ([]byte, error) {
	if u.Valid {
		return u.UUID.MarshalText()
	}
	return []byte{}, nil
}
