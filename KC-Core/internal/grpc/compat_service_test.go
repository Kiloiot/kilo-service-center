package grpc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
)

const (
	compatShimFile     = "compat_service.go"
	compatShimReceiver = "KiloCenterServiceCompat"
	compatCoreField    = "core"
)

// compatDelegates parses the shim source and returns, for every method
// declared on the shim receiver, the CoreService method it forwards to.
func compatDelegates(t *testing.T) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, compatShimFile, nil, 0)
	require.NoError(t, err)

	delegates := map[string]string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if ident, ok := star.X.(*ast.Ident); !ok || ident.Name != compatShimReceiver {
			continue
		}
		delegates[fn.Name.Name] = delegateTarget(fn)
	}
	return delegates
}

// delegateTarget returns the name of the s.core method a shim method calls in
// its single return statement, or "" when the body has any other shape.
func delegateTarget(fn *ast.FuncDecl) string {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return ""
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return ""
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != compatCoreField {
		return ""
	}
	return sel.Sel.Name
}

func descMethodNames(desc grpc.ServiceDesc) map[string]struct{} {
	names := map[string]struct{}{}
	for _, m := range desc.Methods {
		names[m.MethodName] = struct{}{}
	}
	for _, s := range desc.Streams {
		names[s.StreamName] = struct{}{}
	}
	return names
}

// TestCompatShim_DelegatesEveryCoreRPC pins the compatibility contract: every
// CoreService RPC on KiloCenterService must reach the same-named CoreService
// method through the shim, and no Identity RPC may be declared on it (those
// route to KC-Identity at the gateway and must keep falling through).
func TestCompatShim_DelegatesEveryCoreRPC(t *testing.T) {
	delegates := compatDelegates(t)
	unified := descMethodNames(pb.KiloCenterService_ServiceDesc)
	core := descMethodNames(pb.CoreService_ServiceDesc)
	identity := descMethodNames(pb.IdentityService_ServiceDesc)

	for name := range unified {
		_, isCore := core[name]
		_, isIdentity := identity[name]
		require.True(t, isCore || isIdentity, "KiloCenterService.%s belongs to neither CoreService nor IdentityService", name)
		target, declared := delegates[name]
		if isCore {
			assert.True(t, declared, "KiloCenterService.%s must be delegated by the shim", name)
			assert.Equal(t, name, target, "KiloCenterService.%s must forward to CoreService.%s", name, name)
			continue
		}
		assert.False(t, declared, "Identity RPC %s must not be declared on the shim; it is served by KC-Identity", name)
	}
	for name := range delegates {
		_, unifiedHas := unified[name]
		assert.True(t, unifiedHas, "shim declares %s, which KiloCenterService does not define", name)
	}
}
