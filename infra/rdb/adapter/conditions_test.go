package adapter

import (
	"database/sql"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestNormalizeConditionsPreservesNonUUIDArguments(t *testing.T) {
	expression := clause.Expr{SQL: "name = ?", Vars: []any{"a"}}
	conditions := []any{"id = ? AND name = ?", 42, "a", []byte{1, 2}, expression}
	require.Equal(t, conditions, NormalizeConditions(conditions))
	id := uuid.NewV7()
	original := map[string]any{"args": []any{sql.Named("id", id), (*uuid.UUID)(nil)}}
	result := NormalizeConditions([]any{original})
	args := result[0].(map[string]any)["args"].([]any)
	require.Equal(t, sql.Named("id", id.String()), args[0])
	require.Nil(t, args[1])
	require.Equal(t, id, original["args"].([]any)[0].(sql.NamedArg).Value)
}
