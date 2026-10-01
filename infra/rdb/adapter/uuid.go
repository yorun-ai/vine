package adapter

import (
	"context"
	"fmt"
	"reflect"
	"uuid"

	"gorm.io/gorm/schema"
)

func init() {
	schema.RegisterSerializer("vine-rdb-uuid", _UUIDSerializer{})
}

type _UUIDSerializer struct{}

// Scan decodes SQL text or bytes into a UUID field.
// SQL NULL becomes a zero UUID value or a nil UUID pointer; invalid values return an error.
func (_UUIDSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, value any) error {
	var id uuid.UUID
	var err error
	switch value := value.(type) {
	case nil:
	case string:
		id, err = uuid.Parse(value)
	case []byte:
		id, err = uuid.Parse(string(value))
	default:
		return fmt.Errorf("rdb: cannot scan UUID from %T", value)
	}
	if err != nil {
		return err
	}
	target := field.ReflectValueOf(ctx, dst)
	switch target.Type() {
	case reflect.TypeFor[uuid.UUID]():
		target.Set(reflect.ValueOf(id))
	case reflect.TypeFor[*uuid.UUID]():
		if value == nil {
			target.SetZero()
		} else {
			target.Set(reflect.ValueOf(&id))
		}
	default:
		return fmt.Errorf("rdb: UUID serializer requires uuid.UUID or *uuid.UUID, got %s", target.Type())
	}
	return nil
}

// Value encodes UUID values as strings and nil pointers as SQL NULL.
// Unsupported field values return an error.
func (_UUIDSerializer) Value(_ context.Context, _ *schema.Field, _ reflect.Value, value any) (any, error) {
	switch id := value.(type) {
	case nil:
		return nil, nil
	case uuid.UUID:
		return id.String(), nil
	case *uuid.UUID:
		if id == nil {
			return nil, nil
		}
		return id.String(), nil
	default:
		return nil, fmt.Errorf("rdb: UUID serializer requires uuid.UUID or *uuid.UUID, got %T", value)
	}
}
