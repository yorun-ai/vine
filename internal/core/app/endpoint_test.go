package app

import "testing"

func TestInprocHostPath(t *testing.T) {
	if got := InprocHostPath("instance-1"); got != "app/instance-1" {
		t.Fatalf("unexpected app inproc endpoint: %s", got)
	}
}
