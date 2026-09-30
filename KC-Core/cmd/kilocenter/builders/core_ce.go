package builders

// defaultFederationWirer wires CE-only federation services (bootstrap/onboarding).
func defaultFederationWirer(fctx *FederationContext) (*FederationResult, error) {
	bootstrap, err := NewCEBootstrapHandler(fctx)
	if err != nil {
		return nil, err
	}
	return &FederationResult{BootstrapHandler: bootstrap}, nil
}
