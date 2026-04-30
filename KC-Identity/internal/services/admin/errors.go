package admin

import "errors"

// Admin service errors.

// User admin errors
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserEmailExists  = errors.New("user email already exists")
	ErrUserDeleteFailed = errors.New("user delete failed")
	ErrUserPasswordWeak = errors.New("password too weak")
)

// Organization admin errors
var (
	ErrOrganizationNotFound     = errors.New("organization not found")
	ErrOrganizationNameRequired = errors.New("organization name is required")
	ErrTenantCreationFailed     = errors.New("failed to create tenant for organization")
)

// Membership errors
var (
	ErrMemberNotFound        = errors.New("organization member not found")
	ErrMemberAlreadyExists   = errors.New("user is already a member of this organization")
	ErrCannotRemoveLastOwner = errors.New("cannot remove the last owner of an organization")
)

// API key errors
var (
	ErrAPIKeyNotFound = errors.New("api key not found")
)
