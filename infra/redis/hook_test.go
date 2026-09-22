package redis

import (
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSelectHook(t *testing.T) {
	server := miniredis.RunT(t)
	for _, base := range []string{"redis://" + server.Addr(), "redis+memory://select-hook"} {
		for _, dbIndex := range []int{0, 1} {
			endpoint := fmt.Sprintf("%s/%d", base, dbIndex)
			t.Run(endpoint, func(t *testing.T) {
				ctx := t.Context()
				client, release := acquireRedisClient(&Option{Endpoint: endpoint})
				t.Cleanup(release)
				other, releaseOther := acquireRedisClient(&Option{Endpoint: fmt.Sprintf("%s/%d", base, 2)})
				t.Cleanup(releaseOther)
				require.NoError(t, other.Set(ctx, "db", "other", 0).Err())

				// The first connection and subsequent pipeline connections must initialize
				// the configured DB, including nonzero DBs, through go-redis's own SELECT.
				require.NoError(t, client.Set(ctx, "db", "configured", 0).Err())
				require.ErrorIs(t, client.Do(ctx, "SeLeCt", 2).Err(), errSelectDatabase)
				cmd := goredis.NewStatusCmd(ctx, "SELECT", 2)
				require.ErrorIs(t, client.Process(ctx, cmd), errSelectDatabase)
				require.ErrorIs(t, cmd.Err(), errSelectDatabase)
				require.NoError(t, client.Do(ctx, "SELECT", dbIndex).Err())
				require.NoError(t, client.Do(ctx, "SELECT", []byte(fmt.Sprint(dbIndex))).Err())
				require.ErrorIs(t, client.Do(ctx, "SELECT", "invalid").Err(), errSelectDatabase)

				conn := client.Conn()
				require.ErrorIs(t, conn.Select(ctx, 2).Err(), errSelectDatabase)
				require.Equal(t, "configured", conn.Get(ctx, "db").Val())
				require.NoError(t, conn.Close())
				require.NoError(t, client.Watch(ctx, func(tx *goredis.Tx) error {
					require.ErrorIs(t, tx.Select(ctx, 2).Err(), errSelectDatabase)
					return tx.Get(ctx, "db").Err()
				}, "db"))

				for _, tx := range []bool{false, true} {
					pipe := client.Pipeline()
					if tx {
						pipe = client.TxPipeline()
					}
					before := pipe.Set(ctx, "before-select", "value", 0)
					selection := pipe.Select(ctx, 2)
					after := pipe.Set(ctx, "after-select", "value", 0)
					_, err := pipe.Exec(ctx)
					require.ErrorIs(t, err, errSelectDatabase)
					for _, queued := range []goredis.Cmder{before, selection, after} {
						require.ErrorIs(t, queued.Err(), errSelectDatabase)
					}
					require.EqualValues(t, 0, client.Exists(ctx, "before-select", "after-select").Val())
					require.EqualValues(t, 0, other.Exists(ctx, "before-select", "after-select").Val())

					pipe.Set(ctx, "pipeline", "value", 0)
					read := pipe.Get(ctx, "pipeline")
					_, err = pipe.Exec(ctx)
					require.NoError(t, err)
					require.Equal(t, "value", read.Val())
				}
				require.Equal(t, "configured", client.Get(ctx, "db").Val())
				require.Equal(t, "other", other.Get(ctx, "db").Val())
			})
		}
	}
}
