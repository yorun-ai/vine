package logger

import "testing"

func TestIsValidFormat(t *testing.T) {
	for _, format := range []Format{FormatJSON, FormatText} {
		if !IsValidFormat(format) {
			t.Fatalf("expected valid format: %s", format)
		}
	}

	if IsValidFormat(Format("PLAIN")) {
		t.Fatal("expected invalid format")
	}
}

func TestSetGlobalFormat(t *testing.T) {
	resetGlobalOptionForTest(t)

	SetGlobalFormat(FormatJSON)
	if got := currentGlobalFormat(); got != FormatJSON {
		t.Fatalf("currentGlobalFormat() = %s, want %s", got, FormatJSON)
	}

	SetGlobalFormat(FormatText)
	if got := currentGlobalFormat(); got != FormatText {
		t.Fatalf("currentGlobalFormat() = %s, want %s", got, FormatText)
	}
}

func TestSetGlobalFormatRejectsInvalidFormat(t *testing.T) {
	resetGlobalOptionForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	SetGlobalFormat(Format("PLAIN"))
}
