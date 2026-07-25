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
