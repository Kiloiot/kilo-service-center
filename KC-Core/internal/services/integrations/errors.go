package integrations

import "errors"

// Domain sentinels for integration validation failures.
var (
	// ErrIntegrationNotFound reports an integration ID with no row under the tenant.
	ErrIntegrationNotFound = errors.New("integration not found")
	// ErrInvalidType reports an unsupported integration type.
	ErrInvalidType = errors.New("invalid integration type")
	// ErrInvalidStatus reports an unsupported integration status.
	ErrInvalidStatus = errors.New("invalid integration status")
)

// Operation sentinels wrapping repository failures.
var (
	// ErrCreateIntegration wraps repository failures while creating an integration.
	ErrCreateIntegration = errors.New("create integration")
	// ErrGetIntegration wraps repository failures while loading an integration.
	ErrGetIntegration = errors.New("get integration")
	// ErrUpdateIntegration wraps repository failures while updating an integration.
	ErrUpdateIntegration = errors.New("update integration")
	// ErrDeleteIntegration wraps repository failures while deleting an integration.
	ErrDeleteIntegration = errors.New("delete integration")
	// ErrListIntegrations wraps repository failures while listing integrations.
	ErrListIntegrations = errors.New("list integrations")
)
