package skel

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/buildinfo"
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

func TestRegisterDomainSchemaPanicsOnDuplicatePartialDomain(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "first-partial-hash",
		Generated: validGeneratedInfoForTest(),
		Services: []*ServiceSchema{
			{SkelName: "demo.user.PublicService"},
		},
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate partial domain to panic")
		}
	}()
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "next-partial-hash",
		Generated: validGeneratedInfoForTest(),
		Actors: []*ActorSchema{
			{SkelName: "demo.user.ClientActor"},
		},
	})
}

func TestRegisterDomainSchemaFullOverridesRegisteredDomain(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "pub-hash",
		Generated: validGeneratedInfoForTest(),
		Services: []*ServiceSchema{
			{SkelName: "demo.user.PublicService"},
		},
	})
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "full-hash",
		Full:      true,
		Generated: validGeneratedInfoForTest(),
		Actors: []*ActorSchema{
			{SkelName: "demo.user.ClientActor"},
		},
	})

	schemas := registry.RegisteredDomainSchemas()
	if len(schemas) != 1 || schemas[0].Hash != "full-hash" {
		t.Fatalf("expected full schema to override partial schema, got %+v", schemas)
	}
}

func TestRegisteredDomainSchemasReturnsDomainSortedSchemas(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "user-hash",
		Generated: validGeneratedInfoForTest(),
	})
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.booker",
		Hash:      "booker-hash",
		Generated: validGeneratedInfoForTest(),
	})
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.base",
		Hash:      "base-hash",
		Generated: validGeneratedInfoForTest(),
	})

	schemas := registry.RegisteredDomainSchemas()
	if len(schemas) != 3 {
		t.Fatalf("unexpected registered schemas count: %d", len(schemas))
	}
	if schemas[0].Domain != "demo.base" || schemas[1].Domain != "demo.booker" || schemas[2].Domain != "demo.user" {
		t.Fatalf("unexpected schema order: %s, %s, %s", schemas[0].Domain, schemas[1].Domain, schemas[2].Domain)
	}
}

func TestServiceSchemaHasAudienceMatchesActorVia(t *testing.T) {
	service := &ServiceSchema{Audiences: []*ActorAudienceSchema{
		{SkelName: "demo.user.UserActor", Via: ActorViaAgent},
		{SkelName: "demo.admin.AdminActor"},
	}}

	if !service.HasAudience("demo.user.UserActor", ActorViaAgent) {
		t.Fatal("expected exact actor via to match")
	}
	if service.HasAudience("demo.user.UserActor", ActorViaClient) {
		t.Fatal("expected different actor via not to match")
	}
	if !service.HasAudience("demo.admin.AdminActor", ActorViaClient) {
		t.Fatal("expected empty audience via to match every actor via")
	}
	if service.HasAudience("demo.missing.MissingActor", ActorViaAgent) {
		t.Fatal("expected different actor not to match")
	}
}

func TestRegisterDomainSchemaPanicsOnDuplicateFullDomain(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "full-hash",
		Full:      true,
		Generated: validGeneratedInfoForTest(),
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate full domain to panic")
		}
	}()
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.user",
		Hash:      "next-full-hash",
		Full:      true,
		Generated: validGeneratedInfoForTest(),
	})
}

func validGeneratedInfoForTest() *GeneratedInfo {
	return &GeneratedInfo{CompilerVersion: "v99.0.0"}
}

func TestRegistrySelectsEncodingByCompilerVersion(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.legacy",
		Full:      true,
		Generated: &GeneratedInfo{CompilerVersion: "v0.14.9"},
		Services:  []*ServiceSchema{{SkelName: "demo.legacy.Service"}},
	})
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.current",
		Full:      true,
		Generated: &GeneratedInfo{CompilerVersion: "v0.15.0"},
		Services:  []*ServiceSchema{{SkelName: "demo.current.Service"}},
	})
	registry.RegisterDomainSchema(&DomainSchema{
		Domain:    "demo.devel",
		Full:      true,
		Generated: &GeneratedInfo{CompilerVersion: buildinfo.DevVersion},
		Services:  []*ServiceSchema{{SkelName: "demo.devel.Service"}},
	})

	payload := struct {
		Items []string `json:"items"`
	}{}
	legacy := registry.EncoderForSkelName("demo.legacy.Service").MustMarshalJson(payload)
	current := registry.EncoderForSkelName("demo.current.Service").MustMarshalJson(payload)
	devel := registry.EncoderForSkelName("demo.devel.Service").MustMarshalJson(payload)

	if got := string(legacy); got != `{"items":null}` {
		t.Fatalf("legacy encoding = %s", got)
	}
	if got := string(current); got != `{"items":[]}` {
		t.Fatalf("current encoding = %s", got)
	}
	if got := string(devel); got != `{"items":[]}` {
		t.Fatalf("devel encoding = %s", got)
	}

	cborPayload := struct {
		Items []string `cbor:"items"`
	}{}
	legacyCbor := registry.EncoderForSkelName("demo.legacy.Service").MustMarshalCbor(cborPayload)
	currentCbor := registry.EncoderForSkelName("demo.current.Service").MustMarshalCbor(cborPayload)
	var legacyValues, currentValues map[string]cbor.RawMessage
	if err := cbor.Unmarshal(legacyCbor, &legacyValues); err != nil {
		t.Fatal(err)
	}
	if err := cbor.Unmarshal(currentCbor, &currentValues); err != nil {
		t.Fatal(err)
	}
	if got := legacyValues["items"]; string(got) != string(cbor.RawMessage{0xf6}) {
		t.Fatalf("legacy CBOR encoding = %x", got)
	}
	if got := currentValues["items"]; string(got) != string(cbor.RawMessage{0x80}) {
		t.Fatalf("current CBOR encoding = %x", got)
	}
}

func TestRegisterDomainSchemaPanicsOnLowCompilerVersion(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("expected low compiler version to panic")
		}
		if !strings.Contains(value.(error).Error(), "generated by skelc v0.8.9 is lower than Vine required skelc version") {
			t.Fatalf("unexpected panic: %v", value)
		}
	}()
	registry.RegisterDomainSchema(&DomainSchema{
		Domain: "demo.user",
		Hash:   "full-hash",
		Generated: &GeneratedInfo{
			CompilerVersion: "v0.8.9",
		},
	})
}

func TestRegisterDomainSchemaPanicsOnMissingCompilerVersion(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("expected missing compiler version to panic")
		}
		if !strings.Contains(value.(error).Error(), "missing generated compiler version") {
			t.Fatalf("unexpected panic: %v", value)
		}
	}()
	registry.RegisterDomainSchema(&DomainSchema{
		Domain: "demo.user",
		Hash:   "full-hash",
	})
}
