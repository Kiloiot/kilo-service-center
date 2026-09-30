package config

import "testing"

func TestGetServiceCenterURLNamesOnlyAnAddressStationsCanReach(t *testing.T) {
	tests := []struct {
		name        string
		externalURL string
		host        string
		want        string
	}{
		{name: "external URL", externalURL: "tls://bssci.example.com:5000", host: "0.0.0.0", want: "tls://bssci.example.com:5000"},
		{name: "external URL on loopback", externalURL: "tls://localhost:5000", host: "0.0.0.0", want: ""},
		{name: "external URL on loopback IP", externalURL: "tls://127.0.0.1:5000", host: testBSSCIHost, want: ""},
		{name: "external URL on the wildcard address", externalURL: "tls://0.0.0.0:5000", host: testBSSCIHost, want: ""},
		{name: "bind host without external URL", host: testBSSCIHost, want: "tls://bssci.mioty.local:5000"},
		{name: "wildcard bind host without external URL", host: "0.0.0.0", want: ""},
		{name: "loopback bind host without external URL", host: "localhost", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProtocolConfig{BSCIExternalURL: tt.externalURL, BSCIHost: tt.host, BSCIPort: DefaultProtocolBSCIPort}
			if got := GetServiceCenterURL(cfg); got != tt.want {
				t.Errorf("GetServiceCenterURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// testExternalHost is a DNS name a base station can reach.
const testExternalHost = "bssci.example.com"

func TestExternalBSSCIHost(t *testing.T) {
	tests := []struct {
		externalURL string
		want        string
	}{
		{externalURL: "", want: ""},
		{externalURL: "tls://" + testExternalHost + ":5000", want: testExternalHost},
		{externalURL: "tcp://10.0.0.1:5000", want: "10.0.0.1"},
		{externalURL: testExternalHost, want: testExternalHost},
		{externalURL: "tls://0.0.0.0:5000", want: ""},
		{externalURL: "tls://localhost:5000", want: "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.externalURL, func(t *testing.T) {
			if got := ExternalBSSCIHost(&ProtocolConfig{BSCIExternalURL: tt.externalURL}); got != tt.want {
				t.Errorf("ExternalBSSCIHost(%q) = %q, want %q", tt.externalURL, got, tt.want)
			}
		})
	}
}
