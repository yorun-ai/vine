package access

import (
	"encoding/base64"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
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

func TestParseCredentialOptionalFields(t *testing.T) {
	for _, tt := range []struct {
		name, header string
		valid        bool
		want         map[string]string
	}{
		{name: "omitted", header: "key1 token", valid: true, want: map[string]string{"key1": "token"}},
		{name: "provided", header: "KEY1 token, KEY2 tenant", valid: true, want: map[string]string{"key1": "token", "key2": "tenant"}},
		{name: "missing required", header: "key2 tenant"},
		{name: "empty optional", header: "key1 token, key2 "},
		{name: "empty required", header: "key1 , key2 tenant"},
		{name: "blank optional", header: "key1 token, key2 \t"},
		{name: "unknown", header: "key1 token, other tenant"},
		{name: "malformed", header: "key1 token, key2"},
		{name: "missing credentials"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			schema := testCredentialSchema()
			schema.Members[0].Type = &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString}
			schema.Members[1].Type = &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString, Nullable: true}
			got, ok := parseCredential(schema, tt.header)
			require.Equal(t, tt.valid, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOptionalCredentialAuthForwarding(t *testing.T) {
	for _, transport := range []struct {
		name         string
		web, network bool
	}{
		{name: "rpc inproc"}, {name: "web inproc", web: true}, {name: "rpc http", network: true}, {name: "web http", web: true, network: true},
	} {
		t.Run(transport.name, func(t *testing.T) {
			for _, tt := range []struct {
				name, header string
				valid        bool
				optional     *string
			}{
				{name: "omitted", header: "key1 token", valid: true},
				{name: "provided", header: "key1 token, key2 tenant", valid: true, optional: new("tenant")},
				{name: "empty", header: "key1 token, key2 "},
				{name: "required missing", header: "key2 tenant"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					endpoint := "link+inproc://vine/optional-credential"
					calls := 0
					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						var args struct {
							Params struct {
								Credential struct {
									Key1 string  `json:"key1"`
									Key2 *string `json:"key2"`
								} `json:"credential"`
							} `json:"params"`
						}
						require.NoError(t, json.Unmarshal(body, &args))
						assert.Equal(t, "token", args.Params.Credential.Key1)
						assert.Equal(t, tt.optional, args.Params.Credential.Key2)
						if tt.optional == nil {
							assert.False(t, strings.Contains(string(body), `"key2"`))
						}
						writeTestAuthResponse(w, r, http.StatusOK, "OK", `{"userId":"u1"}`)
					})
					if transport.network {
						server := httptest.NewUnstartedServer(handler)
						server.Config.Protocols = new(http.Protocols)
						server.Config.Protocols.SetHTTP1(true)
						server.Config.Protocols.SetUnencryptedHTTP2(true)
						server.Start()
						t.Cleanup(server.Close)
						endpoint = server.URL
					} else {
						ingressinproc.Register(endpoint, handler)
						t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
					}
					schema := testAuthActorSchema()
					schema.AuthCredential.Members[1].Type = &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString, Nullable: true}
					values := testAuthValues(endpoint)
					values[redised.FormatSchemaActorKey("demo.UserActor")] = vcode.MustMarshalJsonS(schema)
					manager := testManager(values)
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest(http.MethodPost, "http://demo.local/demo.UserService/Get", nil)
					setTestRequestHeaders(t, request)
					request.Header.Set("Authorization", tt.header)
					var ok bool
					if transport.web {
						ok = manager.AuthWeb(&WebOperation{Auther: authOperationForTest(t, request, recorder), ActorVia: redised.PortalActorVia{ActorSkelName: "demo.UserActor"}})
					} else {
						ok = manager.AllowRpc(testRpcAuthContext(t, redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, request, recorder))
					}
					require.Equal(t, tt.valid, ok)
					if tt.valid {
						assert.Equal(t, 1, calls)
					} else {
						assert.Zero(t, calls)
						assert.Contains(t, recorder.Body.String(), "bad credential")
					}
				})
			}
		})
	}
}
