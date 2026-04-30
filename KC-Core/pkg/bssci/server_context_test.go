// server_context_test.go
package bssci

import "context"

// SetContextForTests sets the server context for testing
// Only compiled during `go test` due to _test.go suffix
// This allows tests to initialize the context without exposing production API
func (s *Server) SetContextForTests(ctx context.Context) {
	s.ctx = ctx
}
