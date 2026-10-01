package config

// Upstream is the KC-Core address the gateway dials.
func (g GatewayConfig) Upstream() string {
	return addressOrDefault(g.UpstreamAddress, DefaultGatewayUpstreamAddress)
}

// Identity is the KC-Identity address the gateway dials.
func (g GatewayConfig) Identity() string {
	return addressOrDefault(g.IdentityAddress, DefaultGatewayIdentityAddress)
}

// addressOrDefault is address, or fallback when the setting is empty.
func addressOrDefault(address, fallback string) string {
	if address == "" {
		return fallback
	}
	return address
}
