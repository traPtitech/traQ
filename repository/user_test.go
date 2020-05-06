package repository

import (
	"database/sql"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils/optional"
	"testing"
)

func TestUsersQuery_Active(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsActive: optional.Bool{NullBool: sql.NullBool{Bool: true, Valid: true}}},
		UsersQuery{}.Active(),
	)
}

func TestUsersQuery_NotBot(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsBot: optional.Bool{NullBool: sql.NullBool{Bool: false, Valid: true}}},
		UsersQuery{}.NotBot(),
	)
}

func TestUsersQuery_CMemberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV4()
	assert.EqualValues(t,
		UsersQuery{IsCMemberOf: optional.UUID{NullUUID: uuid.NullUUID{UUID: id, Valid: true}}},
		UsersQuery{}.CMemberOf(id),
	)
}

func TestUsersQuery_GMemberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV4()
	assert.EqualValues(t,
		UsersQuery{IsGMemberOf: optional.UUID{NullUUID: uuid.NullUUID{UUID: id, Valid: true}}},
		UsersQuery{}.GMemberOf(id),
	)
}

func TestUsersQuery_Composite(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsActive: optional.Bool{NullBool: sql.NullBool{Bool: true, Valid: true}}, IsBot: optional.Bool{NullBool: sql.NullBool{Bool: false, Valid: true}}},
		UsersQuery{}.NotBot().Active(),
	)
}
