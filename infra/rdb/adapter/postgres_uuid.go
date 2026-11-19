package adapter

import (
	"uuid"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgx scans Go's array-backed UUID through its type map, including for
// database/sql callers. Text columns need a UUID scan plan of their own.
func registerPostgresUUIDText(types *pgtype.Map) {
	for _, oid := range []uint32{pgtype.TextOID, pgtype.VarcharOID, pgtype.BPCharOID} {
		column, _ := types.TypeForOID(oid)
		types.RegisterType(&pgtype.Type{Name: column.Name, OID: oid, Codec: _UUIDTextCodec{column.Codec}})
	}
}

type _UUIDTextCodec struct{ pgtype.Codec }

func (c _UUIDTextCodec) PlanScan(types *pgtype.Map, oid uint32, format int16, target any) pgtype.ScanPlan {
	// For **uuid.UUID, pgx handles SQL NULL and delegates non-NULL values to this plan.
	if _, ok := target.(*uuid.UUID); ok {
		return _ScanTextToUUID{}
	}
	return c.Codec.PlanScan(types, oid, format, target)
}

type _ScanTextToUUID struct{}

func (_ScanTextToUUID) Scan(src []byte, target any) error {
	return target.(*uuid.UUID).UnmarshalText(src)
}
