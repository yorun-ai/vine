package adapter

import (
	"database/sql"
	"uuid"

	"gorm.io/gorm/clause"
)

// NormalizeConditions converts UUID query arguments before GORM interprets arrays.
// UUID primary-key shorthand becomes an explicit predicate rather than a raw SQL string.
// Maps and slices are copied so callers retain their original arguments.
func NormalizeConditions(conditions []any) []any {
	normalized := make([]any, len(conditions))
	for i, condition := range conditions {
		normalized[i] = normalizeArgument(condition)
	}
	if len(conditions) > 0 {
		switch conditions[0].(type) {
		case uuid.UUID, *uuid.UUID:
			normalized[0] = clause.Eq{Column: clause.PrimaryColumn, Value: normalized[0]}
		}
	}
	return normalized
}

func normalizeArgument(value any) any {
	switch value := value.(type) {
	case uuid.UUID:
		return value.String()
	case *uuid.UUID:
		if value == nil {
			return nil
		}
		return value.String()
	case []uuid.UUID:
		ids := make([]string, len(value))
		for i, id := range value {
			ids[i] = id.String()
		}
		return ids
	case []*uuid.UUID:
		ids := make([]any, len(value))
		for i, id := range value {
			ids[i] = normalizeArgument(id)
		}
		return ids
	case []any:
		args := make([]any, len(value))
		for i, arg := range value {
			args[i] = normalizeArgument(arg)
		}
		return args
	case map[string]any:
		args := make(map[string]any, len(value))
		for key, arg := range value {
			args[key] = normalizeArgument(arg)
		}
		return args
	case sql.NamedArg:
		value.Value = normalizeArgument(value.Value)
		return value
	default:
		return value
	}
}
