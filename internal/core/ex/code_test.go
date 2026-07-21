package ex

import "testing"

func TestCodeMetadata(t *testing.T) {
	testCases := []struct {
		code             Code
		kind             Type
		category         Category
		unresponsive     bool
		canRaiseDirectly bool
		defaultMessage   string
	}{
		{code: OK, kind: NoError, category: SuccessCategory, canRaiseDirectly: true},
		{code: ServiceUnavailable, kind: SystemError, category: FrameworkCategory, canRaiseDirectly: true},
		{code: GatewayTimeout, kind: SystemError, category: FrameworkCategory, canRaiseDirectly: true},
		{code: ClientForbidden, kind: SystemError, category: FrameworkCategory, canRaiseDirectly: true},
		{code: InvalidRequest, kind: SystemError, category: FrameworkCategory, canRaiseDirectly: true},
		{code: InvalidEvent, kind: SystemError, category: FrameworkCategory, canRaiseDirectly: true},
		{code: InvalidTask, kind: SystemError, category: FrameworkCategory, canRaiseDirectly: true},
		{code: ServerUnreachable, kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirectly: true},
		{code: InvocationCancelled, kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirectly: true},
		{code: InvocationTimeout, kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirectly: true},
		{code: InvocationFailed, kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirectly: true},
		{code: UnexpectedResponse, kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirectly: true},
		{code: Internal, kind: SystemError, category: FallbackCategory, defaultMessage: "error occurred, please retry"},
		{code: Unknown, kind: SystemError, category: FallbackCategory, defaultMessage: "unknown error"},
		{code: Unauthorized, kind: ApplicationError, category: ApplicationCategory, canRaiseDirectly: true},
		{code: PermissionDenied, kind: ApplicationError, category: ApplicationCategory, canRaiseDirectly: true},
		{code: ElevationRequired, kind: ApplicationError, category: ApplicationCategory, canRaiseDirectly: true},
		{code: ValidationFailed, kind: ApplicationError, category: ApplicationCategory, canRaiseDirectly: true},
		{code: OperationFailed, kind: ApplicationError, category: ApplicationCategory, canRaiseDirectly: true},
		{code: NotFound, kind: ApplicationError, category: ApplicationCategory, canRaiseDirectly: true},
	}

	for _, tc := range testCases {
		if !tc.code.IsValid() {
			t.Fatalf("expected %s to be valid", tc.code)
		}
		if got := tc.code.Type(); got != tc.kind {
			t.Fatalf("unexpected type for %s: got %s want %s", tc.code, got, tc.kind)
		}
		if got := tc.code.Category(); got != tc.category {
			t.Fatalf("unexpected category for %s: got %s want %s", tc.code, got, tc.category)
		}
		if got := tc.code.IsUnresponsive(); got != tc.unresponsive {
			t.Fatalf("unexpected unresponsive flag for %s: got %t want %t", tc.code, got, tc.unresponsive)
		}
		if got := tc.code.CanRaiseDirectly(); got != tc.canRaiseDirectly {
			t.Fatalf("unexpected direct-raise flag for %s: got %t want %t", tc.code, got, tc.canRaiseDirectly)
		}
		if got := tc.code.DefaultMessage(); got != tc.defaultMessage {
			t.Fatalf("unexpected default message for %s: got %q want %q", tc.code, got, tc.defaultMessage)
		}
	}
}

func TestInvalidCodeMetadata(t *testing.T) {
	code := Code("BOGUS")

	if code.IsValid() {
		t.Fatalf("expected invalid code")
	}
	if got := code.Type(); got != InvalidType {
		t.Fatalf("unexpected invalid type: got %s", got)
	}
	if got := code.Category(); got != InvalidCategory {
		t.Fatalf("unexpected invalid category: got %s", got)
	}
	if code.IsUnresponsive() {
		t.Fatalf("invalid code should not be unresponsive")
	}
	if code.CanRaiseDirectly() {
		t.Fatalf("invalid code should not be directly raisable")
	}
	if got := code.DefaultMessage(); got != "" {
		t.Fatalf("unexpected default message for invalid code: %q", got)
	}
}

func TestParseCode(t *testing.T) {
	if _, err := ParseCode(string(OK)); err != nil {
		t.Fatalf("expected OK to parse, got error: %v", err)
	}
	if _, err := ParseCode("BOGUS"); err == nil {
		t.Fatalf("expected invalid code parse to fail")
	}
}
