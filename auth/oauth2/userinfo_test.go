package oauth2

import "github.com/satori/go.uuid"

type UserInfoMock struct {
	uid uuid.UUID
}

func (u *UserInfoMock) GetUID() uuid.UUID {
	return u.uid
}

func (*UserInfoMock) GetName() string {
	return "testuser"
}
