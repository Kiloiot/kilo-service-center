package grpc

import "testing"

// TestPublicMethodsSubsetOfOrgExempt ensures every public method is also org-exempt.
func TestPublicMethodsSubsetOfOrgExempt(t *testing.T) {
	for method := range PublicMethods {
		if !OrgExemptMethods[method] {
			t.Errorf("PublicMethod %q is not in OrgExemptMethods — every auth-exempt method must also be org-exempt", method)
		}
	}
}

// TestOrgExemptSupersetIsStrictlyLarger ensures OrgExemptMethods has at least one
// method not in PublicMethods (e.g., GetSystemStatus requires auth but not org context).
func TestOrgExemptSupersetIsStrictlyLarger(t *testing.T) {
	extraCount := 0
	for method := range OrgExemptMethods {
		if !PublicMethods[method] {
			extraCount++
		}
	}
	if extraCount == 0 {
		t.Error("OrgExemptMethods should contain at least one method beyond PublicMethods (e.g., GetSystemStatus)")
	}
}
