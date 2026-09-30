package grpc

import "testing"

// TestPublicMethodsSubsetOfOrgExempt ensures every public method is also org-exempt.
func TestPublicMethodsSubsetOfOrgExempt(t *testing.T) {
	for method := range publicMethods {
		if !orgExemptMethods[method] {
			t.Errorf("PublicMethod %q is not in orgExemptMethods — every auth-exempt method must also be org-exempt", method)
		}
	}
}

// TestOrgExemptSupersetIsStrictlyLarger ensures orgExemptMethods has at least one
// method not in publicMethods (e.g., GetSystemStatus requires auth but not org context).
func TestOrgExemptSupersetIsStrictlyLarger(t *testing.T) {
	var extraCount int
	for method := range orgExemptMethods {
		if !publicMethods[method] {
			extraCount++
		}
	}
	if extraCount == 0 {
		t.Error("orgExemptMethods should contain at least one method beyond publicMethods (e.g., GetSystemStatus)")
	}
}
