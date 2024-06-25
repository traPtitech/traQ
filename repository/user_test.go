package repository

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/utils/optional"
)

func TestUsersQuery_Active(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsActive: optional.From(true)},
		UsersQuery{}.Active(),
	)
}

func TestUsersQuery_NotBot(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsBot: optional.From(false)},
		UsersQuery{}.NotBot(),
	)
}

func TestUsersQuery_CMemberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV7()
	assert.EqualValues(t,
		UsersQuery{IsCMemberOf: optional.From(id)},
		UsersQuery{}.CMemberOf(id),
	)
}

func TestUsersQuery_GMemberOf(t *testing.T) {
	t.Parallel()

	id, _ := uuid.NewV7()
	assert.EqualValues(t,
		UsersQuery{IsGMemberOf: optional.From(id)},
		UsersQuery{}.GMemberOf(id),
	)
}

func TestUsersQuery_Composite(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t,
		UsersQuery{IsActive: optional.From(true), IsBot: optional.From(false)},
		UsersQuery{}.NotBot().Active(),
	)
}
