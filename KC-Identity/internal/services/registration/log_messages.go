// Package registration log message constants.
// Each Log* identifier names a structured log message produced by the
// registration service. Keeping the strings here lets the verify-constants
// gate enforce zero hardcoded log literals in this package.
package registration

const (
	// LogRegistrationEmailUniquenessCheckFailed is logged when the email-uniqueness
	// pre-check returns an error other than "not found".
	LogRegistrationEmailUniquenessCheckFailed = "failed to check email uniqueness"

	// LogRegistrationCEDefaultOrgLookupFailed is logged when the CE-edition default
	// org lookup fails or returns nil during account registration.
	LogRegistrationCEDefaultOrgLookupFailed = "CE default org lookup failed"

	// LogRegistrationCERegistrationFailed is logged when the CE-edition registration
	// adapter rejects the account-creation request.
	LogRegistrationCERegistrationFailed = "CE registration failed"

	// LogRegistrationTransactionFailed is logged when the registration transaction
	// (user + membership + token writes) fails.
	LogRegistrationTransactionFailed = "registration transaction failed"

	// LogRegistrationAccessTokenIssueFailed is logged when the access-token issuer
	// fails after a successful user-record write.
	LogRegistrationAccessTokenIssueFailed = "failed to issue access token after registration"

	// LogRegistrationRefreshTokenIssueFailed is logged when the refresh-token issuer
	// fails after a successful user-record write.
	LogRegistrationRefreshTokenIssueFailed = "failed to issue refresh token after registration"

	// LogRegistrationRefreshTokenStoreFailed is logged when the refresh-token store
	// fails to persist the freshly issued token.
	LogRegistrationRefreshTokenStoreFailed = "failed to store refresh token after registration"

	// LogRegistrationMembershipLoadFailed is logged when the post-registration
	// membership fetch fails (the registration itself succeeded).
	LogRegistrationMembershipLoadFailed = "failed to load memberships after registration"
)
