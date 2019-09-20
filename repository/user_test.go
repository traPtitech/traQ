package repository

import (
	"database/sql"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/guregu/null.v3"
	"testing"
)

func TestUsersQuery_Active(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsActive: null.Bool{NullBool: sql.NullBool{Bool: true, Valid: true}}},
		UsersQuery{}.Active(),
	)
}

func TestUsersQuery_NotBot(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsBot: null.Bool{NullBool: sql.NullBool{Bool: false, Valid: true}}},
		UsersQuery{}.NotBot(),
	)
}

func TestUsersQuery_CMemberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV4()
	assert.EqualValues(t,
		UsersQuery{IsCMemberOf: uuid.NullUUID{UUID: id, Valid: true}},
		UsersQuery{}.CMemberOf(id),
	)
}

func TestUsersQuery_GMemberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV4()
	assert.EqualValues(t,
		UsersQuery{IsGMemberOf: uuid.NullUUID{UUID: id, Valid: true}},
		UsersQuery{}.GMemberOf(id),
	)
}

func TestUsersQuery_SubscriberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV4()
	assert.EqualValues(t,
		UsersQuery{IsSubscriberOf: uuid.NullUUID{UUID: id, Valid: true}},
		UsersQuery{}.SubscriberOf(id),
	)
}

func TestUsersQuery_Composite(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV4()
	assert.EqualValues(t,
		UsersQuery{IsActive: null.Bool{NullBool: sql.NullBool{Bool: true, Valid: true}}, IsBot: null.Bool{NullBool: sql.NullBool{Bool: false, Valid: true}}, IsSubscriberOf: uuid.NullUUID{UUID: id, Valid: true}},
		UsersQuery{}.NotBot().SubscriberOf(id).Active(),
	)
}
