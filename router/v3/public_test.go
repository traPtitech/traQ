package v3

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/repository"
	file2 "github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/random"
)

func TestHandlers_GetVersion(t *testing.T) {
	t.Parallel()

	path := "/api/v3/version"
	env := Setup(t, common1)

	e := env.R(t)
	obj := e.GET(path).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()

	obj.Value("version").String().IsEqual("version")
	obj.Value("revision").String().IsEqual("revision")

	flags := obj.Value("flags").Object()

	flags.Value("signUpAllowed").Boolean().IsFalse()

	ext := flags.Value("externalLogin").Array()
	ext.Length().IsEqual(1)
	ext.Value(0).String().IsEqual("traq")
}

func TestHandlers_GetPublicUserIcon(t *testing.T) {
	t.Parallel()

	path := "/api/v3/public/icon/{username}"
	env := Setup(t, common1)
	iconFileID, err := file2.GenerateIconFile(env.FM, "test")
	require.NoError(t, err)
	user, err := env.Repository.CreateUser(repository.CreateUserArgs{
		Name:       random.AlphaNumeric(20),
		Password:   "totallyASecurePassword",
		Role:       role.User,
		IconFileID: iconFileID,
	})
	require.NoError(t, err)

	e := env.R(t)
	e.GET(path, user.GetName()).
		Expect().
		Status(http.StatusOK).
		HasContentType("image/png")
}
