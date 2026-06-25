package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFeatureFlag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "feature_flags", (&FeatureFlag{}).TableName())
}
