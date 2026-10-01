package config

import (
	"fmt"
	"net"
)

// An internal gRPC hop - the gateway to KC-Core and KC-Identity, KC-Core to
// KC-Identity - carries identity headers the receiving listener trusts. Any
// such hop other hosts can reach must authenticate its peer with the shared
// internal_auth.peer_secret; only a loopback address keeps the hop on one
// host.

// validateInternalTrust refuses a listener that trusts the gateway's
// identity headers on an address other hosts reach without the peer secret.
func (c *Config) validateInternalTrust() error {
	if !c.GRPC.InternalTrustEnabled {
		return nil
	}
	return c.requirePeerSecret(c.GRPC.Host, ErrInternalTrustPeerAuthRequiredFmt)
}

// validateCore is KC-Core's validation. In ECE KC-Core requires organization
// enforcement, and its SCACI listener must isolate organizations. It presents
// the peer secret to KC-Identity, which requires it whenever another host
// reaches it; without an identity.address KC-Core dials no KC-Identity.
func (c *Config) validateCore() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := c.validateEnterpriseOrgEnforcement(); err != nil {
		return err
	}
	if err := c.validateSCACIListener(); err != nil {
		return err
	}
	if c.Identity.Address == "" {
		return nil
	}
	return c.requirePeerSecret(hostOf(c.Identity.Address), ErrIdentityUpstreamPeerAuthRequiredFmt)
}

// validateIdentity is KC-Identity's validation. Its listener trusts internal
// identity headers and serves IdentityInternalService on the peer secret
// alone, whatever grpc.internal_trust_enabled says.
func (c *Config) validateIdentity() error {
	if err := c.Validate(); err != nil {
		return err
	}
	return c.requirePeerSecret(c.GRPC.Host, ErrIdentityPeerAuthRequiredFmt)
}

// validateGateway is KC-Gateway's validation. In ECE the gateway requires
// organization enforcement. It presents the peer secret to KC-Core and
// KC-Identity, which require it whenever another host reaches them.
func (c *Config) validateGateway() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := c.validateEnterpriseOrgEnforcement(); err != nil {
		return err
	}
	for _, upstream := range []string{c.Gateway.Upstream(), c.Gateway.Identity()} {
		if err := c.requirePeerSecret(hostOf(upstream), ErrGatewayPeerAuthRequiredFmt); err != nil {
			return err
		}
	}
	return nil
}

// requirePeerSecret refuses a peer secret shorter than
// InternalAuthPeerSecretMinLength for an internal hop on host, unless host is
// a loopback address.
func (c *Config) requirePeerSecret(host, format string) error {
	length := len(c.InternalAuth.PeerSecret)
	if isLoopbackHost(host) || length >= InternalAuthPeerSecretMinLength {
		return nil
	}
	return fmt.Errorf(format, host, InternalAuthPeerSecretMinLength, length)
}

// isLoopbackHost reports whether host names only this machine; an empty host
// binds every interface.
func isLoopbackHost(host string) bool {
	if host == LoopbackHostname {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// hostOf is the host of a host:port address; an address without a port is
// all host.
func hostOf(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}
