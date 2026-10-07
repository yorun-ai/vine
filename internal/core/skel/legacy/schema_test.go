package legacy

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestSchemaSensitiveMetadataJSON(t *testing.T) {
	schema := &DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-hash",
		Data: []*DataSchema{{
			Name:      "Credential",
			SkelName:  "demo.user.Credential",
			Hash:      "data-hash",
			Sensitive: true,
			Members: []*MemberSchema{{
				Name:      "token",
				Sensitive: true,
				Type:      &TypeSchema{Kind: TypeKindScalar, Scalar: ScalarString},
			}},
		}},
		Services: []*ServiceSchema{{
			Name:     "CredentialService",
			SkelName: "demo.user.CredentialService",
			Hash:     "service-hash",
			Methods: []*MethodSchema{{
				Name:               "Exchange",
				SkelName:           "exchange",
				Hash:               "method-hash",
				ArgumentsSensitive: true,
				ResultSensitive:    true,
			}},
		}},
		Tasks: []*TaskSchema{{
			Name:     "RotateCredentialTask",
			SkelName: "demo.user.RotateCredentialTask",
			Hash:     "task-hash",
			Triggers: []*TriggerSchema{{
				Name:               "Manually",
				SkelName:           "manually",
				Hash:               "trigger-hash",
				ArgumentsSensitive: true,
			}},
		}},
	}

	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	for _, expected := range []string{
		`"sensitive":true`,
		`"argumentsSensitive":true`,
		`"resultSensitive":true`,
	} {
		if !strings.Contains(string(encoded), expected) {
			t.Fatalf("expected %s in schema JSON: %s", expected, encoded)
		}
	}
}

func TestSchemaDeprecatedMetadataJSON(t *testing.T) {
	schema := &DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-hash",
		Services: []*ServiceSchema{{
			Name:             "LegacyService",
			SkelName:         "demo.user.LegacyService",
			Deprecated:       true,
			DeprecatedReason: "Use UserService instead.",
			Hash:             "service-hash",
			Methods: []*MethodSchema{{
				Name:             "GetLegacyUser",
				SkelName:         "getLegacyUser",
				Deprecated:       true,
				DeprecatedReason: "Use getUser instead.",
				Hash:             "method-hash",
			}},
		}},
	}

	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	for _, expected := range []string{
		`"deprecated":true`,
		`"deprecatedReason":"Use UserService instead."`,
		`"deprecatedReason":"Use getUser instead."`,
	} {
		if !strings.Contains(string(encoded), expected) {
			t.Fatalf("expected %s in schema JSON: %s", expected, encoded)
		}
	}

	var decoded DomainSchema
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	service := decoded.Services[0]
	if !service.Deprecated || service.DeprecatedReason != "Use UserService instead." {
		t.Fatalf("unexpected decoded service deprecation: %+v", service)
	}
	method := service.Methods[0]
	if !method.Deprecated || method.DeprecatedReason != "Use getUser instead." {
		t.Fatalf("unexpected decoded method deprecation: %+v", method)
	}

	zeroEncoded, err := json.Marshal(&ServiceSchema{})
	if err != nil {
		t.Fatalf("marshal zero-value service schema: %v", err)
	}
	if strings.Contains(string(zeroEncoded), `"deprecated"`) {
		t.Fatalf("expected zero-value deprecation metadata to be omitted: %s", zeroEncoded)
	}
}

func TestExtensionServiceSchemaJSON(t *testing.T) {
	var service ServiceSchema
	if err := json.Unmarshal([]byte(`{"ext":true,"authMode":"noauth"}`), &service); err != nil {
		t.Fatal(err)
	}
	if !service.Ext || service.Api {
		t.Fatalf("extension lost its direction or was classified as an API: %+v", service)
	}
	encoded, err := json.Marshal(service)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"ext":true`) {
		t.Fatalf("extension flag lost in JSON: %s", encoded)
	}
	encoded, err = json.Marshal(ServiceSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"ext"`) {
		t.Fatalf("zero extension flag must be omitted: %s", encoded)
	}
}

func TestExtensionEventSchemaJSON(t *testing.T) {
	var event EventSchema
	if err := json.Unmarshal([]byte(`{"ext":true}`), &event); err != nil {
		t.Fatal(err)
	}
	if !event.Ext {
		t.Fatal("extension event flag lost during decoding")
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"ext":true`) {
		t.Fatalf("extension flag lost: %s", encoded)
	}
	encoded, err = json.Marshal(EventSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"ext"`) {
		t.Fatalf("zero extension flag must be omitted: %s", encoded)
	}
}

func TestPermCheckInvocationCodeArgumentNameJSON(t *testing.T) {
	for name, input := range map[string]string{
		"code":  `{"resourceSkelName":"app.User","actionName":"read","codeArgumentName":"code"}`,
		"code_": `{"resourceSkelName":"app.User","actionName":"read","codeArgumentName":"code_"}`,
	} {
		var check PermCheckInvocation
		if err := json.Unmarshal([]byte(input), &check); err != nil {
			t.Fatal(err)
		}
		if check.CodeArgumentName != name {
			t.Fatalf("decoded argument name = %q, want %q", check.CodeArgumentName, name)
		}
		encoded, err := json.Marshal(check)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if fields["codeArgumentName"] != name {
			t.Fatalf("argument name lost in schema JSON: %s", encoded)
		}
	}
}
