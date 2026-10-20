package logger

import "testing"

func TestSetGlobalFormatRejectsInvalidFormat(t *testing.T) {
	resetGlobalOptionForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	SetGlobalFormat(Format("PLAIN"))
}
