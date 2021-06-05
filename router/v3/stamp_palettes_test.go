package v3

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/optional"
)

func stampPaletteEquals(t *testing.T, expect *model.StampPalette, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().Equal(expect.ID.String())
	actual.Value("name").String().Equal(expect.Name)
	actual.Value("description").String().Equal(expect.Description)
	actual.Value("creatorId").String().Equal(expect.CreatorID.String())
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()

	stamps := make([]interface{}, len(expect.Stamps))
	for i, stamp := range expect.Stamps {
		stamps[i] = stamp.String()
	}
	// Order DOES matter here
	actual.Value("stamps").Array().Elements(stamps...)
}

func TestHandlers_GetStampPalettes(t *testing.T) {
	t.Parallel()

	path := "/api/v3/stamp-palettes"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	sp := env.CreateStampPalette(t, user.GetID(), rand, model.UUIDs{stamp.ID})
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().Equal(1)
		stampPaletteEquals(t, sp, obj.First().Object())
	})
}

func TestCreateStampPaletteRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        string
		Description string
		Stamps      model.UUIDs
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty",
			fields{},
			true,
		},
		{
			"too long name",
			fields{Name: strings.Repeat("a", 50)},
			true,
		},
		{
			"success",
			fields{Name: "test palette", Description: "description", Stamps: model.UUIDs{uuid.Must(uuid.NewV4())}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CreateStampPaletteRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
				Stamps:      tt.fields.Stamps,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_CreateStampPalette(t *testing.T) {
	t.Parallel()

	path := "/api/v3/stamp-palettes"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&CreateStampPaletteRequest{Name: "po"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&CreateStampPaletteRequest{}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid stamp id", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&CreateStampPaletteRequest{Name: "po", Stamps: model.UUIDs{uuid.Must(uuid.NewV4())}}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&CreateStampPaletteRequest{Name: "po", Stamps: model.UUIDs{stamp.ID}}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("name").String().Equal("po")
		stamps := obj.Value("stamps").Array()
		stamps.Length().Equal(1)
		stamps.First().String().Equal(stamp.ID.String())
		obj.Value("creatorId").String().Equal(user.GetID().String())
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("description").String().Empty()
	})
}

func TestPatchStampPaletteRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        optional.String
		Description optional.String
		Stamps      model.UUIDs
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty",
			fields{},
			false,
		},
		{
			"empty name",
			fields{Name: optional.StringFrom("")},
			true,
		},
		{
			"too long name",
			fields{Name: optional.StringFrom(strings.Repeat("a", 50))},
			true,
		},
		{
			"success",
			fields{
				Name:        optional.StringFrom("test"),
				Description: optional.StringFrom("description"),
				Stamps:      model.UUIDs{uuid.Must(uuid.NewV4())},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PatchStampPaletteRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
				Stamps:      tt.fields.Stamps,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditStampPalette(t *testing.T) {
	t.Parallel()

	path := "/api/v3/stamp-palettes/{paletteId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	sp := env.CreateStampPalette(t, user.GetID(), rand, model.UUIDs{stamp.ID})
	sp2 := env.CreateStampPalette(t, user2.GetID(), rand, model.UUIDs{})
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, sp.ID).
			WithJSON(&PatchStampPaletteRequest{Name: optional.StringFrom("test")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, sp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchStampPaletteRequest{Name: optional.StringFrom(strings.Repeat("a", 50))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid stamp", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, sp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchStampPaletteRequest{Stamps: model.UUIDs{uuid.Must(uuid.NewV4())}}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, sp2.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchStampPaletteRequest{Name: optional.StringFrom("test")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchStampPaletteRequest{Name: optional.StringFrom("test")}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, sp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchStampPaletteRequest{Name: optional.StringFrom("test")}).
			Expect().
			Status(http.StatusNoContent)

		updated, err := env.Repository.GetStampPalette(sp.ID)
		require.NoError(t, err)
		assert.EqualValues(t, sp.ID.String(), updated.ID.String())
		assert.EqualValues(t, "test", updated.Name)
	})
}

func TestHandlers_GetStampPalette(t *testing.T) {
	t.Parallel()

	path := "/api/v3/stamp-palettes/{paletteId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	sp := env.CreateStampPalette(t, user.GetID(), rand, model.UUIDs{stamp.ID})
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, sp.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, sp.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		stampPaletteEquals(t, sp, obj)
	})
}

func TestHandlers_DeleteStampPalette(t *testing.T) {
	t.Parallel()

	path := "/api/v3/stamp-palettes/{paletteId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	sp := env.CreateStampPalette(t, user.GetID(), rand, model.UUIDs{stamp.ID})
	sp2 := env.CreateStampPalette(t, user2.GetID(), rand, model.UUIDs{})
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, sp2.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, sp2.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, sp.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetStampPalette(sp.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}
