package config

import (
	"net"
	"strconv"
)

// IsWildcardHost reports whether a bind host means "every interface".
func IsWildcardHost(host string) bool {
	switch host {
	case "", wildcardHostIPv4, wildcardHostIPv6:
		return true
	default:
		return false
	}
}

// ListenerProbeAddress returns the address a same-host probe dials to reach a
// listener bound to host:port; a wildcard bind maps to the loopback name.
func ListenerProbeAddress(host string, port int) string {
	if IsWildcardHost(host) {
		host = ProbeLoopbackHost
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}
