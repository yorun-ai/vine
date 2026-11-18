package watched

import (
	"encoding/json/v2"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestRegistrationUUIDWireCompatibility(t *testing.T) {
	const text = "019a1234-5678-7000-8000-123456789abc"
	id := uuid.MustParse(text)
	const rpcJSON = `{"endpoint":"http://link/rpc/proxy/in/019a1234-5678-7000-8000-123456789abc","serviceName":"demo.Service","appName":"demo.app","appVersion":"1.2.3","appInstanceId":"019a1234-5678-7000-8000-123456789abc"}`
	var registration RpcServiceRegistration
	require.NoError(t, json.Unmarshal([]byte(rpcJSON), &registration))
	require.Equal(t, id, registration.AppInstanceId)
	encoded, err := json.Marshal(registration)
	require.NoError(t, err)
	require.Equal(t, rpcJSON, string(encoded))
	// Old Link reads a string from the same payload and uses the original key.
	var legacy struct {
		AppInstanceId string `json:"appInstanceId"`
	}
	require.NoError(t, json.Unmarshal(encoded, &legacy))
	require.Equal(t, text, legacy.AppInstanceId)
	require.Equal(t, "rpc:demo.Service:endpoint:demo.app:"+text, FormatRpcServiceRegistrationKey("demo.Service", "demo.app", id))
	require.Equal(t, "web:demo.Web:endpoint:demo.app:"+text, FormatWebRegistrationKey("demo.Web", "demo.app", id))
	require.Equal(t, "app:demo.app:status:"+text, FormatAppStatusKey("demo.app", id))
	require.Equal(t, "portal:instance:"+text, FormatPortalInstanceKey(id))
}
