package oauth2

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAuthorizeData_IsExpired(t *testing.T) {
	data := &AuthorizeData{
		CreatedAt: time.Date(2000, 1, 1, 12, 0, 11, 0, time.UTC),
		ExpiresIn: 10,
	}
	assert.True(t, data.IsExpired())

	data = &AuthorizeData{
		CreatedAt: time.Date(2099, 1, 1, 12, 0, 11, 0, time.UTC),
		ExpiresIn: 10,
	}
	assert.False(t, data.IsExpired())
}

func TestAuthorizeData_ValidatePKCE(t *testing.T) {
	assert := assert.New(t)

	data := &AuthorizeData{
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "plain",
	}

	if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.NoError(err) {
		assert.True(ok)
	}
	if ok, err := data.ValidatePKCE("fewfaaafaefe-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
		assert.False(ok)
	}
	if ok, err := data.ValidatePKCE("fewfaaafae"); assert.NoError(err) {
		assert.False(ok)
	}

	data = &AuthorizeData{
		CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
	}

	if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.NoError(err) {
		assert.True(ok)
	}
	if ok, err := data.ValidatePKCE("fewfaaafaefe-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
		assert.False(ok)
	}

	data = &AuthorizeData{}
	if ok, err := data.ValidatePKCE("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
		assert.False(ok)
	}
	if ok, err := data.ValidatePKCE(""); assert.NoError(err) {
		assert.True(ok)
	}

	data = &AuthorizeData{
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	}

	if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.NoError(err) {
		assert.False(ok)
	}
	if ok, err := data.ValidatePKCE("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); assert.NoError(err) {
		assert.True(ok)
	}

	data = &AuthorizeData{
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "unknown",
	}
	if ok, err := data.ValidatePKCE("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"); assert.Error(err) {
		assert.False(ok)
	}

}
