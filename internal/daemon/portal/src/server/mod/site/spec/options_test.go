package spec

import "testing"

func TestSameOriginNormalizesDefaultPorts(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{
			name:  "https default port",
			left:  "https://a.example.com",
			right: "https://a.example.com:443",
			want:  true,
		},
		{
			name:  "http default port",
			left:  "http://a.example.com",
			right: "http://a.example.com:80",
			want:  true,
		},
		{
			name:  "different scheme",
			left:  "https://a.example.com",
			right: "http://a.example.com:443",
			want:  false,
		},
		{
			name:  "different non default port",
			left:  "https://a.example.com",
			right: "https://a.example.com:8443",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameOrigin(tt.left, tt.right); got != tt.want {
				t.Fatalf("sameOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSameDomainOriginAllowsExactIPOrigin(t *testing.T) {
	if !sameDomainOrigin("http://127.0.0.1:2000", EntryOrigin{
		Scheme: SchemeHTTP,
		Host:   "127.0.0.1",
		Port:   2000,
	}) {
		t.Fatal("expected exact ip origin to be allowed")
	}
}

func TestSameDomainOriginRejectsDifferentIPPort(t *testing.T) {
	if sameDomainOrigin("http://127.0.0.1:2000", EntryOrigin{
		Scheme: SchemeHTTP,
		Host:   "127.0.0.1",
		Port:   3000,
	}) {
		t.Fatal("expected different ip port to be rejected")
	}
}

func TestSameDomainOriginAllowsSubdomain(t *testing.T) {
	if !sameDomainOrigin("https://console.example.com", EntryOrigin{
		Scheme: SchemeHTTPS,
		Host:   "api.example.com",
		Port:   443,
	}) {
		t.Fatal("expected same registrable domain to be allowed")
	}
}
