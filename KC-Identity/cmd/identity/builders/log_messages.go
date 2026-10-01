package builders

// Composition-root log messages emitted while wiring KC-Identity services.
const (
	LogWiringAuthAdminServices    = "Wiring Auth and Admin services..."
	LogUserAdminServiceWired      = "UserAdminService wired"
	LogOrgAdminServiceWired       = "OrganizationAdminService wired"
	LogMembershipAdminWired       = "MembershipAdminService wired"
	LogCEAdminServicesSkipped     = "Community Edition: OrganizationAdminService and MembershipAdminService skipped"
	LogAPIKeyAdminServiceWired    = "APIKeyAdminService wired" //nolint:gosec // G101: wiring log message, not a credential
	LogCEDefaultOrgProviderWired  = "CE default-org provider wired for AuthService"
	LogAuthServiceWired           = "AuthService wired"
	LogRegistrationServiceWired   = "RegistrationService wired"
	LogRegistrationServiceSkipped = "RegistrationService skipped"
	LogRedisClientCreateFailed    = "failed to create Redis client for external auth"
	LogRedisClientCloseFailed     = "failed to close Redis client"
	LogStorageCloseFailed         = "failed to close storage"
	LogAuditEmitterCreateFailed   = "failed to create audit emitter"
	LogAuditRecorderCreateFailed  = "failed to create audit recorder"
	LogOAuth2ClientCreateFailed   = "failed to create OAuth2 client"
	LogOIDCClientCreateFailed     = "failed to create OIDC client"
	LogOrgResolverTypeMismatch    = "orgResolverSvc does not implement org.OrganizationResolver"
	LogExternalAuthServiceWired   = "ExternalAuthService wired"
	LogIdentityInternalWired      = "IdentityInternalService wired"
	LogCompatIdentityWired        = "KiloCenterServiceCompatIdentity wired"
)

// Infrastructure bring-up log messages.
const (
	LogInitializingStorage        = "Initializing storage..."
	LogWaitingForDatabase         = "Waiting for database connection..."
	LogRunningMigrations          = "Running database migrations..."
	LogMigrationFailedEventError  = "failed to emit migration.failed event"
	LogMigrationAppliedEventError = "failed to emit migration.applied event"
	LogMigrationsComplete         = "Database migrations complete"
	LogCESingleTenantResolver     = "Community Edition: initializing single-tenant org resolver"
	LogInitializingOrgResolver    = "Initializing organization resolver with caching..."
)

// gRPC server lifecycle log messages.
const (
	LogGRPCServicesRegistered = "gRPC services registered (Identity + IdentityInternal + KiloCenterCompat)"
	LogGRPCReflectionEnabled  = "gRPC reflection enabled"
	LogGRPCListenFailed       = "Failed to listen"
	LogGRPCServerListening    = "gRPC server listening"
	LogGRPCServerFailed       = "gRPC server failed"
)
