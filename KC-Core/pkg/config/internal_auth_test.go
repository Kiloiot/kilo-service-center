package config

import (
	"fmt"
	"strings"
	"testing"
)

const (
	testPeerSecret      = "0123456789abcdef0123456789abcdef"
	testShortPeerSecret = "0123456789abcdef0123456789abcde"
	testPeerSecretEnv   = "KILOCENTER_INTERNAL_AUTH_PEER_SECRET"
)

// peerSecretCase is one configuration and whether its binary may start with it.
type peerSecretCase struct {
	config   string
	accepted bool
}

func runPeerSecretCases(t *testing.T, load func(string) (*Config, error), cases map[string]peerSecretCase) {
	t.Helper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := load(writeTestConfig(t, tc.config))
			switch {
			case tc.accepted && err != nil:
				t.Fatalf("expected the configuration to load, got: %v", err)
			case !tc.accepted && err == nil:
				t.Fatal("expected the configuration to be refused: an internal hop other hosts reach has no peer secret")
			case !tc.accepted && !strings.Contains(err.Error(), "internal_auth.peer_secret"):
				t.Fatalf("expected the refusal to name internal_auth.peer_secret, got: %v", err)
			}
		})
	}
}

func listenerConfig(host string, trusted bool, secret string) string {
	return fmt.Sprintf("grpc:\n  host: %q\n  internal_trust_enabled: %t\ninternal_auth:\n  peer_secret: %q\n", host, trusted, secret)
}

func gatewayConfig(upstream, identity, secret string) string {
	return fmt.Sprintf("gateway:\n  upstream_address: %q\n  identity_address: %q\ninternal_auth:\n  peer_secret: %q\n", upstream, identity, secret)
}

// KC-Core trusts the gateway's identity headers when
// grpc.internal_trust_enabled is set. On an address other hosts reach, any
// of them could send those headers, so the listener requires the peer secret.
func TestLoadCoreRequiresThePeerSecretOnATrustingListenerOtherHostsReach(t *testing.T) {
	runPeerSecretCases(t, Load, map[string]peerSecretCase{
		"all interfaces without a secret":      {listenerConfig("", true, ""), false},
		"0.0.0.0 without a secret":             {listenerConfig("0.0.0.0", true, ""), false},
		"a service address with a short one":   {listenerConfig("10.0.0.5", true, testShortPeerSecret), false},
		"0.0.0.0 with a secret":                {listenerConfig("0.0.0.0", true, testPeerSecret), true},
		"localhost without a secret":           {listenerConfig("localhost", true, ""), true},
		"127.0.0.1 without a secret":           {listenerConfig("127.0.0.1", true, ""), true},
		"::1 without a secret":                 {listenerConfig("::1", true, ""), true},
		"0.0.0.0 without trusting the headers": {listenerConfig("0.0.0.0", false, ""), true},
	})
}

// KC-Core presents the peer secret to KC-Identity, which requires it
// whenever another host reaches it.
func TestLoadCoreRequiresThePeerSecretForKCIdentityOffLoopback(t *testing.T) {
	runPeerSecretCases(t, Load, map[string]peerSecretCase{
		"KC-Identity on another host":               {"identity:\n  address: \"kc-identity:50052\"\n", false},
		"KC-Identity on another host with a secret": {"identity:\n  address: \"kc-identity:50052\"\ninternal_auth:\n  peer_secret: \"" + testPeerSecret + "\"\n", true},
		"KC-Identity on the default local address":  {"grpc:\n  port: 50051\n", true},
		"no KC-Identity":                            {"identity:\n  address: \"\"\n", true},
	})
}

// KC-Identity always trusts internal identity headers and serves
// IdentityInternalService on the peer secret alone, whatever
// grpc.internal_trust_enabled says.
func TestLoadIdentityRequiresThePeerSecretOffLoopback(t *testing.T) {
	runPeerSecretCases(t, LoadIdentity, map[string]peerSecretCase{
		"0.0.0.0 without a secret":           {listenerConfig("0.0.0.0", false, ""), false},
		"all interfaces with a short secret": {listenerConfig("", true, testShortPeerSecret), false},
		"0.0.0.0 with a secret":              {listenerConfig("0.0.0.0", true, testPeerSecret), true},
		"localhost without a secret":         {listenerConfig("localhost", true, ""), true},
	})
}

// KC-Gateway presents the peer secret to KC-Core and KC-Identity, which
// require it whenever another host reaches them.
func TestLoadGatewayRequiresThePeerSecretForUpstreamsOffLoopback(t *testing.T) {
	runPeerSecretCases(t, LoadGateway, map[string]peerSecretCase{
		"the default upstreams":                 {"grpc:\n  port: 9090\n", true},
		"KC-Core on another host":               {gatewayConfig("kilocenter:50051", DefaultGatewayIdentityAddress, ""), false},
		"KC-Identity on another host":           {gatewayConfig(DefaultGatewayUpstreamAddress, "kc-identity:50052", ""), false},
		"both on other hosts, a short secret":   {gatewayConfig("kilocenter:50051", "kc-identity:50052", testShortPeerSecret), false},
		"both on other hosts with a secret":     {gatewayConfig("kilocenter:50051", "kc-identity:50052", testPeerSecret), true},
		"both on loopback addresses, no secret": {gatewayConfig("127.0.0.1:50051", "[::1]:50052", ""), true},
	})
}

// A deployment supplies the secret through the environment, as the Helm
// chart, Docker Compose and Kilo Cloud do.
func TestLoadTakesThePeerSecretFromTheEnvironment(t *testing.T) {
	t.Setenv(testPeerSecretEnv, testPeerSecret)
	runPeerSecretCases(t, Load, map[string]peerSecretCase{
		"0.0.0.0 with the secret in the environment": {"grpc:\n  host: \"0.0.0.0\"\n  internal_trust_enabled: true\n", true},
	})
	runPeerSecretCases(t, LoadIdentity, map[string]peerSecretCase{
		"0.0.0.0 with the secret in the environment": {"grpc:\n  host: \"0.0.0.0\"\n", true},
	})
	runPeerSecretCases(t, LoadGateway, map[string]peerSecretCase{
		"remote upstreams with the secret in the environment": {"gateway:\n  upstream_address: \"kc-core:50051\"\n  identity_address: \"kc-identity:50052\"\n", true},
	})
}

// The Docker configurations bind every interface and dial the other services
// by name, so each refuses to start until the deployment supplies the secret.
func TestShippedDockerConfigurationsNeedThePeerSecret(t *testing.T) {
	for path, load := range map[string]func(string) (*Config, error){
		"../../../config/config.docker.yaml":          Load,
		"../../../config/config.identity-docker.yaml": LoadIdentity,
		"../../../config/config.gateway-docker.yaml":  LoadGateway,
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := load(path); err == nil || !strings.Contains(err.Error(), "internal_auth.peer_secret") {
				t.Fatalf("expected the configuration to be refused without the peer secret, got: %v", err)
			}
			t.Setenv(testPeerSecretEnv, testPeerSecret)
			if _, err := load(path); err != nil {
				t.Fatalf("expected the configuration to load with the peer secret, got: %v", err)
			}
		})
	}
}

// The local development configurations keep every internal hop on the
// loopback address, so they start without a secret.
func TestShippedLocalConfigurationsNeedNoPeerSecret(t *testing.T) {
	for path, load := range map[string]func(string) (*Config, error){
		"../../config.yaml":                Load,
		"../../../KC-Identity/config.yaml": LoadIdentity,
		"../../../KC-Gateway/config.yaml":  LoadGateway,
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := load(path); err != nil {
				t.Fatalf("expected the local configuration to load without a peer secret, got: %v", err)
			}
		})
	}
}
