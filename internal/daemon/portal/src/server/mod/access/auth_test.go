package access

import (
	"encoding/base64"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/util/vcode"
)

func TestParseCredentialFromAuthorizationMapsFieldsCaseInsensitively(t *testing.T) {
	credential, ok := parseCredential(testCredentialSchema(), "Key1 token123, key2 dXNlcjpwd2Q=")

	require.True(t, ok)
	assert.Equal(t, map[string]string{
		"key1": "token123",
		"key2": "dXNlcjpwd2Q=",
	}, credential)
}

func TestParseCredentialFromAuthorizationRejectsUnknownField(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "Key1 token123, unknown value")

	assert.False(t, ok)
}

func TestParseCredentialFromAuthorizationRejectsMissingField(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "Key1 token123")

	assert.False(t, ok)
}

func TestParseCredentialFromAuthorizationRejectsBadItem(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "Key1")

	assert.False(t, ok)
}

func TestParseCredentialRejectsEmptyCredentialSchema(t *testing.T) {
	_, ok := parseCredential(&skel.DataSchema{}, "")

	assert.False(t, ok)
}

func TestParseCredentialRejectsAllEmptyCredentialValues(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "key1 , key2 ")

	assert.False(t, ok)
}

func testCredentialSchema() *skel.DataSchema {
	return &skel.DataSchema{
		SkelName: "demo.UserCredential",
		Members: []*skel.MemberSchema{
			{Name: "key1"},
			{Name: "key2"},
		},
	}
}

func TestAuthPropagatesIdentifierAndRejectsInvalidResponse(t *testing.T) {
	for _, tt := range []struct {
		name, info, want string
		valid            bool
	}{
		{"large integer", `{"userId":9007199254740993}`, "9007199254740993", true},
		{"zero", `{"userId":0}`, "0", true},
		{"missing", `{}`, "", false},
		{"null", `{"userId":null}`, "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := registerTestAuthService(t, http.StatusOK, "OK", tt.info)
			values := testAuthValues(endpoint)
			schema := testAuthActorSchema()
			schema.IdentifierField = "userId"
			schema.AuthInfo.Members = []*skel.MemberSchema{{Name: "userId", Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarInt}}}
			values[redised.FormatSchemaActorKey("demo.UserActor")] = vcode.MustMarshalJsonS(schema)
			manager := testManager(values)
			for _, web := range []bool{false, true} {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
				setTestRequestHeaders(t, request)
				request.Header.Set("Authorization", "key1 token, key2 token")
				request.Header.Set(rpchttp.HeaderRpcActor, "forged")
				request.Header.Set(webspec.HeaderWebActor, "forged")
				var ok bool
				header := rpchttp.HeaderRpcActor
				if web {
					header = webspec.HeaderWebActor
					ok = manager.AuthWeb(&WebOperation{Auther: authOperationForTest(t, request, recorder), ActorVia: redised.PortalActorVia{ActorSkelName: "demo.UserActor"}})
				} else {
					ok = manager.AllowRpc(&RpcOperation{Auther: authOperationForTest(t, request, recorder), Server: testServerApp(), ActorVia: redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, ServiceName: "demo.UserService", MethodName: "Get"})
				}
				require.Equal(t, tt.valid, ok)
				if !ok {
					require.Contains(t, recorder.Body.String(), "bad auth response")
					continue
				}
				data, err := base64.RawURLEncoding.DecodeString(request.Header.Get(header))
				require.NoError(t, err)
				var payload struct {
					Realm      string `json:"realm"`
					Identifier string `json:"identifier"`
				}
				require.NoError(t, json.Unmarshal(data, &payload))
				assert.Equal(t, "demo.UserActor", payload.Realm)
				assert.Equal(t, tt.want, payload.Identifier)
			}
		})
	}
}
