package config

import "testing"

func TestIsWildcardHost(t *testing.T) {
	cases := map[string]bool{
		"":                  true,
		"0.0.0.0":           true,
		"::":                true,
		"localhost":         false,
		"127.0.0.1":         false,
		"::1":               false,
		"bssci.example.com": false,
	}
	for host, want := range cases {
		if got := IsWildcardHost(host); got != want {
			t.Errorf("IsWildcardHost(%q) = %v, want %v", host, got, want)
		}
	}
}

const (
	testPortBSSCI  = 5005
	testPortSCACI  = 5001
	testPortGRPC   = 50051
	testPortHealth = 8086
	testPortRemote = 5000
)

func TestListenerProbeAddress(t *testing.T) {
	cases := []struct {
		host string
		port int
		want string
	}{
		{host: "", port: testPortBSSCI, want: "localhost:5005"},
		{host: "0.0.0.0", port: testPortBSSCI, want: "localhost:5005"},
		{host: "::", port: testPortSCACI, want: "localhost:5001"},
		{host: "127.0.0.1", port: testPortGRPC, want: "127.0.0.1:50051"},
		{host: "::1", port: testPortHealth, want: "[::1]:8086"},
		{host: "kc-core", port: testPortRemote, want: "kc-core:5000"},
	}
	for _, tc := range cases {
		if got := ListenerProbeAddress(tc.host, tc.port); got != tc.want {
			t.Errorf("ListenerProbeAddress(%q, %d) = %q, want %q", tc.host, tc.port, got, tc.want)
		}
	}
}
