package adapter

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestPostgresUUIDTextScan(t *testing.T) {
	types := pgtype.NewMap()
	registerPostgresUUIDText(types)
	id := uuid.NewV7()
	for _, oid := range []uint32{pgtype.TextOID, pgtype.VarcharOID, pgtype.BPCharOID} {
		column, _ := types.TypeForOID(oid)
		t.Run(column.Name, func(t *testing.T) {
			var got uuid.UUID
			require.NoError(t, types.Scan(oid, pgtype.TextFormatCode, []byte(id.String()), &got))
			require.Equal(t, id, got)
			require.Error(t, types.Scan(oid, pgtype.TextFormatCode, []byte("invalid"), &got))
			require.Equal(t, id, got)

			zero := uuid.Nil()
			var nullable *uuid.UUID
			require.NoError(t, types.Scan(oid, pgtype.TextFormatCode, []byte(zero.String()), &nullable))
			require.Equal(t, &zero, nullable)
			require.NoError(t, types.Scan(oid, pgtype.TextFormatCode, nil, &nullable))
			require.Nil(t, nullable)
		})
	}
}
