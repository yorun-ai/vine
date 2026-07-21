package redis

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventJSON(t *testing.T) {
	data, err := json.Marshal(Event{
		Revision: 42,
		Kind:     EventKindUpsert,
		Key:      "config:demo.FeatureConfig",
		Value:    `{"enabled":true}`,
	})

	assert.NoError(t, err)
	assert.JSONEq(t, `{"revision":42,"kind":"upsert","key":"config:demo.FeatureConfig","value":"{\"enabled\":true}"}`, string(data))
}
