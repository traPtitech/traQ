package model

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestFeatureFlag_TableName (t *testing.T) {
	t.Parallel()
	assert.Equal(t, "feature_flags", (&FeatureFlag{}).TableName())
}