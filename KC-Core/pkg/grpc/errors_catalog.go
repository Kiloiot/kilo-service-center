// Package grpc provides gRPC error catalog and utilities for the KiloCenter API
package grpc

import (
	"google.golang.org/grpc/codes"
)

// Error catalog tokens for gRPC service errors
// These tokens enable centralized error messaging, tracking, and localization support
//
//nolint:gosec // Token identifiers are not credentials.
const (
	// Authentication errors (KC-GRPC-ERR-001 to KC-GRPC-ERR-020)
	ErrTokenInvalidCredentials  = "KC-GRPC-ERR-001"
	ErrTokenUserNotFound        = "KC-GRPC-ERR-002"
	ErrTokenInvalidRefreshToken = "KC-GRPC-ERR-003"
	ErrTokenSessionExpired      = "KC-GRPC-ERR-004"
	ErrTokenPasswordMismatch    = "KC-GRPC-ERR-005"
	ErrTokenUserNotActive       = "KC-GRPC-ERR-006"
	ErrTokenEmailAlreadyExists  = "KC-GRPC-ERR-007"
	ErrTokenWeakPassword        = "KC-GRPC-ERR-008"
	ErrTokenTokenGenerationFail = "KC-GRPC-ERR-009"
	ErrTokenUnauthorized        = "KC-GRPC-ERR-010"

	// Auth handler operation errors (KC-GRPC-ERR-011 to KC-GRPC-ERR-020)
	ErrTokenRefreshTokenRequired    = "KC-GRPC-ERR-011"
	ErrTokenCurrentPasswordRequired = "KC-GRPC-ERR-012"
	ErrTokenNewPasswordRequired     = "KC-GRPC-ERR-013"
	ErrTokenGetProfileFailed        = "KC-GRPC-ERR-014"
	ErrTokenGetAuthSettingsFailed   = "KC-GRPC-ERR-015"
	ErrTokenLogoutFailed            = "KC-GRPC-ERR-016"
	ErrTokenChangePasswordFailed    = "KC-GRPC-ERR-017"
	ErrTokenCreateUserFailed        = "KC-GRPC-ERR-018"
	ErrTokenUpdateUserFailed        = "KC-GRPC-ERR-019"
	ErrTokenDeleteUserFailed        = "KC-GRPC-ERR-01A"
	ErrTokenListUsersFailed         = "KC-GRPC-ERR-01B"
	ErrTokenUpdatePasswordFailed    = "KC-GRPC-ERR-01C"

	// Auth interceptor errors (KC-GRPC-ERR-151 to KC-GRPC-ERR-165)
	// Auth interceptor tokens
	ErrTokenMissingMetadata    = "KC-GRPC-ERR-151"
	ErrTokenMissingAuthToken   = "KC-GRPC-ERR-152"
	ErrTokenInvalidAuthFormat  = "KC-GRPC-ERR-153"
	ErrTokenInvalidToken       = "KC-GRPC-ERR-154"
	ErrTokenInvalidTokenIssuer = "KC-GRPC-ERR-155"
	ErrTokenMissingAudience    = "KC-GRPC-ERR-156"
	ErrTokenInvalidAudience    = "KC-GRPC-ERR-157"
	ErrTokenMissingTenantClaim = "KC-GRPC-ERR-158"
	ErrTokenMissingTenantCtx   = "KC-GRPC-ERR-159"
	ErrTokenMissingUserCtx     = "KC-GRPC-ERR-160"
	// Fail-closed tenant isolation when auth disabled
	ErrTokenTenantHeaderRequired = "KC-GRPC-ERR-161"

	// Org resolver interceptor errors (KC-GRPC-ERR-166 to KC-GRPC-ERR-175)
	// Org resolver interceptor tokens
	ErrTokenOrgResolverRequired  = "KC-GRPC-ERR-166"
	ErrTokenOrgIDHeaderRequired  = "KC-GRPC-ERR-167"
	ErrTokenOrgIDHeaderInvalid   = "KC-GRPC-ERR-168"
	ErrTokenUserIDHeaderRequired = "KC-GRPC-ERR-169"
	ErrTokenUserIDHeaderInvalid  = "KC-GRPC-ERR-170"
	ErrTokenOrgResolutionFailed  = "KC-GRPC-ERR-171"
	ErrTokenIdentityMismatch     = "KC-GRPC-ERR-172" // Header values conflict with authenticated token

	// Organization errors (KC-GRPC-ERR-300 to KC-GRPC-ERR-311)
	ErrTokenOrgNotFound           = "KC-GRPC-ERR-300"
	ErrTokenOrgAlreadyExists      = "KC-GRPC-ERR-301"
	ErrTokenMembershipExists      = "KC-GRPC-ERR-302"
	ErrTokenMembershipNotFound    = "KC-GRPC-ERR-303"
	ErrTokenMembershipRequired    = "KC-GRPC-ERR-304"
	ErrTokenInvalidOrgRole        = "KC-GRPC-ERR-305"
	ErrTokenCannotRemoveSelf      = "KC-GRPC-ERR-306"
	ErrTokenOrgHasMembers         = "KC-GRPC-ERR-307"
	ErrTokenCannotRemoveLastOwner = "KC-GRPC-ERR-308"
	ErrTokenAdminRequired         = "KC-GRPC-ERR-309"
	ErrTokenOrgContextMismatch    = "KC-GRPC-ERR-310"
	// Shared validation: update_mask required for partial-update RPCs
	ErrTokenUpdateMaskRequired = "KC-GRPC-ERR-311"

	// API Key errors (KC-GRPC-ERR-021 to KC-GRPC-ERR-030)
	//revive:disable-next-line:var-naming
	ErrTokenApiKeyNotFound = "KC-GRPC-ERR-021"
	//revive:disable-next-line:var-naming
	ErrTokenApiKeyExpired = "KC-GRPC-ERR-022"
	//revive:disable-next-line:var-naming
	ErrTokenApiKeyInactive = "KC-GRPC-ERR-023"
	//revive:disable-next-line:var-naming
	ErrTokenApiKeyNameExists = "KC-GRPC-ERR-024"
	//revive:disable-next-line:var-naming
	ErrTokenInvalidApiKeyType = "KC-GRPC-ERR-025"

	// Organization operation errors (KC-GRPC-ERR-026 to KC-GRPC-ERR-02A)
	ErrTokenCreateOrgFailed = "KC-GRPC-ERR-026"
	ErrTokenUpdateOrgFailed = "KC-GRPC-ERR-027"
	ErrTokenDeleteOrgFailed = "KC-GRPC-ERR-028"
	ErrTokenListOrgsFailed  = "KC-GRPC-ERR-029"
	ErrTokenRoleRequired    = "KC-GRPC-ERR-02A"

	// Membership operation errors (KC-GRPC-ERR-02B to KC-GRPC-ERR-02F)
	ErrTokenAddMemberFailed           = "KC-GRPC-ERR-02B"
	ErrTokenGetMemberFailed           = "KC-GRPC-ERR-02C"
	ErrTokenUpdateMemberFailed        = "KC-GRPC-ERR-02D"
	ErrTokenRemoveMemberFailed        = "KC-GRPC-ERR-02E"
	ErrTokenListMembersFailed         = "KC-GRPC-ERR-02F"
	ErrTokenMemberAddedRetrieveFail   = "KC-GRPC-ERR-088"
	ErrTokenMemberUpdatedRetrieveFail = "KC-GRPC-ERR-089"

	// API Key operation errors (KC-GRPC-ERR-08A to KC-GRPC-ERR-08D)
	//revive:disable-next-line:var-naming
	ErrTokenCreateApiKeyFailed = "KC-GRPC-ERR-08A"
	//revive:disable-next-line:var-naming
	ErrTokenDeleteApiKeyFailed = "KC-GRPC-ERR-08B"
	//revive:disable-next-line:var-naming
	ErrTokenListApiKeysFailed = "KC-GRPC-ERR-08C"

	// Certificate errors (KC-GRPC-ERR-031 to KC-GRPC-ERR-040)
	ErrTokenCertGenerationFailed       = "KC-GRPC-ERR-031"
	ErrTokenCertNotFound               = "KC-GRPC-ERR-032"
	ErrTokenCertExpired                = "KC-GRPC-ERR-033"
	ErrTokenCertInvalid                = "KC-GRPC-ERR-034"
	ErrTokenCertRenewalFailed          = "KC-GRPC-ERR-035"
	ErrTokenCertTypeRequired           = "KC-GRPC-ERR-036"
	ErrTokenCertStatusFailed           = "KC-GRPC-ERR-037"
	ErrTokenCertServerGenerationFailed = "KC-GRPC-ERR-038"
	ErrTokenCertDirCreateFailed        = "KC-GRPC-ERR-039"
	ErrTokenCertGeneratorNotFound      = "KC-GRPC-ERR-03A"
	ErrTokenCACertReadFailed           = "KC-GRPC-ERR-03B"
	ErrTokenCACertCopyFailed           = "KC-GRPC-ERR-03C"
	ErrTokenCAKeyReadFailed            = "KC-GRPC-ERR-03D"
	ErrTokenCAKeyCopyFailed            = "KC-GRPC-ERR-03E"
	ErrTokenInvalidValidityPeriod      = "KC-GRPC-ERR-03F"
	ErrTokenNoCertsToRenew             = "KC-GRPC-ERR-040"

	// Blueprint/Catalog errors (KC-GRPC-ERR-041 to KC-GRPC-ERR-060)
	ErrTokenManufacturerNotFound     = "KC-GRPC-ERR-041"
	ErrTokenManufacturerNameExists   = "KC-GRPC-ERR-042"
	ErrTokenDeviceModelNotFound      = "KC-GRPC-ERR-043"
	ErrTokenDeviceModelNameExists    = "KC-GRPC-ERR-044"
	ErrTokenBlueprintNotFound        = "KC-GRPC-ERR-045"
	ErrTokenBlueprintNameExists      = "KC-GRPC-ERR-046"
	ErrTokenBlueprintInvalid         = "KC-GRPC-ERR-047"
	ErrTokenRegistrySubmitFailed     = "KC-GRPC-ERR-048"
	ErrTokenManufacturerHasModels    = "KC-GRPC-ERR-049"
	ErrTokenDeviceModelHasBlueprints = "KC-GRPC-ERR-050"

	// Blueprint operation errors (KC-GRPC-ERR-051 to KC-GRPC-ERR-060)
	ErrTokenCreateManufacturerFailed        = "KC-GRPC-ERR-051"
	ErrTokenUpdateManufacturerFailed        = "KC-GRPC-ERR-052"
	ErrTokenDeleteManufacturerFailed        = "KC-GRPC-ERR-053"
	ErrTokenListManufacturersFailed         = "KC-GRPC-ERR-054"
	ErrTokenCreateDeviceModelFailed         = "KC-GRPC-ERR-055"
	ErrTokenUpdateDeviceModelFailed         = "KC-GRPC-ERR-056"
	ErrTokenDeleteDeviceModelFailed         = "KC-GRPC-ERR-057"
	ErrTokenListDeviceModelsFailed          = "KC-GRPC-ERR-058"
	ErrTokenCreateBlueprintFailed           = "KC-GRPC-ERR-059"
	ErrTokenUpdateBlueprintFailed           = "KC-GRPC-ERR-05A"
	ErrTokenDeleteBlueprintFailed           = "KC-GRPC-ERR-05B"
	ErrTokenListBlueprintsFailed            = "KC-GRPC-ERR-05C"
	ErrTokenSetDefaultBlueprintFailed       = "KC-GRPC-ERR-05D"
	ErrTokenInvalidTypeEUIFormat            = "KC-GRPC-ERR-05E"
	ErrTokenSubmitBlueprintToRegistryFailed = "KC-GRPC-ERR-05F"
	ErrTokenAtomicModelBlueprintFailed      = "KC-GRPC-ERR-060" // Composite model+blueprint creation transaction rollback

	// Blueprint preview errors (KC-GRPC-ERR-069 to KC-GRPC-ERR-06D)
	ErrTokenSlugGenerationFailed   = "KC-GRPC-ERR-069" // Slug collision exhausted all suffix attempts
	ErrTokenPreviewPayloadRequired = "KC-GRPC-ERR-06A" // Preview request missing payload
	ErrTokenPreviewDecodeFailed    = "KC-GRPC-ERR-06B" // Blueprint decoder returned error
	ErrTokenPreviewInvalidFormatID = "KC-GRPC-ERR-06C" // Format ID not recognized by decoder
	ErrTokenPreviewInvalidPayload  = "KC-GRPC-ERR-06D" // Payload bytes failed validation/parsing
	ErrTokenInvalidModelCode       = "KC-GRPC-ERR-06E" // Model code format validation failed

	// Integration operation errors (KC-GRPC-ERR-400 to KC-GRPC-ERR-410)
	// Integration CRUD tokens
	ErrTokenCreateIntegrationFailed = "KC-GRPC-ERR-400"
	ErrTokenGetIntegrationFailed    = "KC-GRPC-ERR-401"
	ErrTokenUpdateIntegrationFailed = "KC-GRPC-ERR-402"
	ErrTokenDeleteIntegrationFailed = "KC-GRPC-ERR-403"
	ErrTokenListIntegrationsFailed  = "KC-GRPC-ERR-404"
	ErrTokenIntegrationNotFound     = "KC-GRPC-ERR-405"
	ErrTokenMissingOrgContext       = "KC-GRPC-ERR-406"
	ErrTokenInvalidRequest          = "KC-GRPC-ERR-407"

	// Endpoint Stats/Operations errors (KC-GRPC-ERR-408 to KC-GRPC-ERR-410)
	// Endpoint stats and operations tokens
	ErrTokenGetEndpointStatsFailed      = "KC-GRPC-ERR-408"
	ErrTokenGetEndpointOperationsFailed = "KC-GRPC-ERR-409"
	ErrTokenConfigRequired              = "KC-GRPC-ERR-410"

	// Registry Provider Errors (KC-GRPC-ERR-420 to KC-GRPC-ERR-42F)
	// Blueprint registry submission tokens
	ErrTokenRegistryProviderDisabled       = "KC-GRPC-ERR-420"
	ErrTokenRegistryAuthFailed             = "KC-GRPC-ERR-421"
	ErrTokenRegistryRateLimited            = "KC-GRPC-ERR-422"
	ErrTokenRegistryBranchCreateFailed     = "KC-GRPC-ERR-423"
	ErrTokenRegistryCommitFailed           = "KC-GRPC-ERR-424"
	ErrTokenRegistrySubmissionCreateFailed = "KC-GRPC-ERR-425"
	ErrTokenRegistryAPIError               = "KC-GRPC-ERR-426"
	ErrTokenBlueprintAlreadySubmitted      = "KC-GRPC-ERR-427"
	ErrTokenContributorNameRequired        = "KC-GRPC-ERR-428"
	ErrTokenContributorEmailRequired       = "KC-GRPC-ERR-429"
	ErrTokenRegistryAPIURLRequired         = "KC-GRPC-ERR-42A"
	ErrTokenRegistryPermissionDenied       = "KC-GRPC-ERR-42B"
	ErrTokenRegistryVersionAlreadyExists   = "KC-GRPC-ERR-42C"
	ErrTokenRegistryInvalidPathSegment     = "KC-GRPC-ERR-42D"

	// SCACI Monitoring errors (KC-GRPC-ERR-061 to KC-GRPC-ERR-070)
	ErrTokenScaciSessionNotFound    = "KC-GRPC-ERR-061"
	ErrTokenScaciQueueNotFound      = "KC-GRPC-ERR-062"
	ErrTokenScaciServiceOffline     = "KC-GRPC-ERR-063"
	ErrTokenScaciListSessionsFailed = "KC-GRPC-ERR-064"
	ErrTokenScaciStatisticsFailed   = "KC-GRPC-ERR-065"
	ErrTokenScaciListErrorsFailed   = "KC-GRPC-ERR-066"
	ErrTokenScaciListQueuesFailed   = "KC-GRPC-ERR-067"
	ErrTokenScaciStatusFailed       = "KC-GRPC-ERR-068"

	// Auth claim resolution errors (KC-GRPC-ERR-270)
	// Tenant resolver tokens
	// NOTE: ErrTokenOrgResolutionFailed already exists at KC-GRPC-ERR-171
	ErrTokenTenantResolverRequired = "KC-GRPC-ERR-270"

	// External auth errors (KC-GRPC-ERR-180 to KC-GRPC-ERR-190)
	// OIDC/OAuth2 exchange tokens
	ErrTokenAuthOIDCDisabled         = "KC-GRPC-ERR-180"
	ErrTokenAuthOAuth2Disabled       = "KC-GRPC-ERR-181"
	ErrTokenAuthStateNotFound        = "KC-GRPC-ERR-182"
	ErrTokenAuthTokenExchangeFailed  = "KC-GRPC-ERR-183"
	ErrTokenAuthIDTokenInvalid       = "KC-GRPC-ERR-184"
	ErrTokenAuthEmailNotVerified     = "KC-GRPC-ERR-185"
	ErrTokenAuthRegistrationDisabled = "KC-GRPC-ERR-186"
	ErrTokenAuthRedisUnavailable     = "KC-GRPC-ERR-187"
	ErrTokenAuthUserInfoFailed       = "KC-GRPC-ERR-188"
	ErrTokenAuthStateInvalid         = "KC-GRPC-ERR-189"
	ErrTokenAuthExternalOrgFailed    = "KC-GRPC-ERR-18A"

	// Self-service registration errors (KC-GRPC-ERR-190 to KC-GRPC-ERR-199)
	ErrTokenRegistrationDisabled = "KC-GRPC-ERR-190"
	ErrTokenFirstNameRequired    = "KC-GRPC-ERR-191"
	ErrTokenLastNameRequired     = "KC-GRPC-ERR-192"
	ErrTokenCompanyNameRequired  = "KC-GRPC-ERR-193"
	ErrTokenRegistrationFailed   = "KC-GRPC-ERR-194"
	ErrTokenOrgAdminRequired     = "KC-GRPC-ERR-195"
	ErrTokenUserEmailRequired    = "KC-GRPC-ERR-196"
	ErrTokenRateLimitExceeded    = "KC-GRPC-ERR-197"
	ErrTokenEmailInvalidFormat   = "KC-GRPC-ERR-198"
	ErrTokenFieldTooLong         = "KC-GRPC-ERR-199"

	// SCACI operation mapping errors (KC-GRPC-ERR-271 to KC-GRPC-ERR-281)
	// SCACI-to-gRPC error mapping tokens
	ErrTokenScaciCntDependMismatch        = "KC-GRPC-ERR-271"
	ErrTokenScaciCntDependPacketCntOmit   = "KC-GRPC-ERR-272"
	ErrTokenScaciNonCntDependMultiPayload = "KC-GRPC-ERR-273"
	ErrTokenScaciQueueIDOutOfRange        = "KC-GRPC-ERR-274"
	ErrTokenScaciFailedPersistDownlink    = "KC-GRPC-ERR-275"
	ErrTokenScaciQueueIDExists            = "KC-GRPC-ERR-276"
	ErrTokenScaciSchedulerUnavailable     = "KC-GRPC-ERR-277"
	ErrTokenScaciBaseStationUnavailable   = "KC-GRPC-ERR-278"
	ErrTokenScaciEndpointNotFound         = "KC-GRPC-ERR-279"
	ErrTokenScaciDownlinkNotFound         = "KC-GRPC-ERR-27A"
	ErrTokenScaciOperationFailed          = "KC-GRPC-ERR-27B"

	// Message/Analytics errors (KC-GRPC-ERR-071 to KC-GRPC-ERR-080)
	ErrTokenMessageNotFound       = "KC-GRPC-ERR-071"
	ErrTokenInvalidTimeRange      = "KC-GRPC-ERR-072"
	ErrTokenExportFailed          = "KC-GRPC-ERR-073"
	ErrTokenStreamDisconnected    = "KC-GRPC-ERR-074"
	ErrTokenAnalyticsOverviewFail = "KC-GRPC-ERR-075"
	ErrTokenAnalyticsActivityFail = "KC-GRPC-ERR-076"
	ErrTokenAnalyticsSignalFail   = "KC-GRPC-ERR-077"
	ErrTokenListEventsFailed      = "KC-GRPC-ERR-078"
	ErrTokenListBSEventsFailed    = "KC-GRPC-ERR-079"
	ErrTokenListEPEventsFailed    = "KC-GRPC-ERR-07A"
	ErrTokenListAlertsFailed      = "KC-GRPC-ERR-07B"
	ErrTokenAlertSummaryFailed    = "KC-GRPC-ERR-07C"

	// Event streaming errors (KC-GRPC-ERR-08D to KC-GRPC-ERR-08F)
	// Realtime streaming tokens
	ErrTokenStreamEventsStartFailed   = "KC-GRPC-ERR-08D"
	ErrTokenStreamBSEventsStartFailed = "KC-GRPC-ERR-08E"
	ErrTokenStreamEPEventsStartFailed = "KC-GRPC-ERR-08F"

	// Message listing operation errors (KC-GRPC-ERR-07D to KC-GRPC-ERR-07F, KC-GRPC-ERR-084 to KC-GRPC-ERR-090)
	ErrTokenListMessagesFailed          = "KC-GRPC-ERR-07D"
	ErrTokenStreamStartFailed           = "KC-GRPC-ERR-07E"
	ErrTokenListBSMessagesFailed        = "KC-GRPC-ERR-07F"
	ErrTokenMessageStatsFailed          = "KC-GRPC-ERR-084"
	ErrTokenSearchMessagesFailed        = "KC-GRPC-ERR-085"
	ErrTokenExportMessagesFailed        = "KC-GRPC-ERR-086"
	ErrTokenStreamBSMessagesStartFailed = "KC-GRPC-ERR-087"
	ErrTokenUnsupportedGranularity      = "KC-GRPC-ERR-090"
	ErrTokenListActivityFailed          = "KC-GRPC-ERR-500" // Unified activity feed

	// Export operation errors (KC-GRPC-ERR-200 to KC-GRPC-ERR-210)
	// Message store adapter export tokens
	ErrTokenExportFetchFailed       = "KC-GRPC-ERR-200"
	ErrTokenExportUnsupportedFormat = "KC-GRPC-ERR-201"
	ErrTokenExportEncodeFailed      = "KC-GRPC-ERR-202"
	ErrTokenExportCSVWriteFailed    = "KC-GRPC-ERR-203"

	// Downlink operation errors (KC-GRPC-ERR-211 to KC-GRPC-ERR-230)
	ErrTokenDownlinkPayloadRequired     = "KC-GRPC-ERR-211"
	ErrTokenDownlinkPayloadTooLarge     = "KC-GRPC-ERR-212"
	ErrTokenDownlinkFormatInvalid       = "KC-GRPC-ERR-213"
	ErrTokenDownlinkQueueFailed         = "KC-GRPC-ERR-214"
	ErrTokenDownlinkRevokeFailed        = "KC-GRPC-ERR-215"
	ErrTokenDownlinkGetResultsFailed    = "KC-GRPC-ERR-216"
	ErrTokenDownlinkQueueListFailed     = "KC-GRPC-ERR-217"
	ErrTokenEndpointNotBidirectional    = "KC-GRPC-ERR-218"
	ErrTokenBaseStationNotBidirectional = "KC-GRPC-ERR-219"
	ErrTokenDownlinkRevokeNotFound      = "KC-GRPC-ERR-21A"
	ErrTokenDownlinkStatusNotFound      = "KC-GRPC-ERR-21B"
	ErrTokenDownlinkAttachFailed        = "KC-GRPC-ERR-21C"
	ErrTokenDownlinkDetachFailed        = "KC-GRPC-ERR-21D"
	ErrTokenQueueIDRequired             = "KC-GRPC-ERR-21E"
	ErrTokenInvalidQueueIDFormat        = "KC-GRPC-ERR-21F"
	ErrTokenNoBaseStationsConnected     = "KC-GRPC-ERR-220"
	ErrTokenQueueIDNonNegative          = "KC-GRPC-ERR-221"
	ErrTokenDownlinkGetFailed           = "KC-GRPC-ERR-222"
	ErrTokenNoBaseStationOwnsQueue      = "KC-GRPC-ERR-223"
	ErrTokenDownlinkRevokeNoResult      = "KC-GRPC-ERR-224"
	ErrTokenResultCountOverflow         = "KC-GRPC-ERR-225"

	// UL Transmit errors (KC-GRPC-ERR-230 to KC-GRPC-ERR-240)
	ErrTokenBaseStationOwnershipFailed = "KC-GRPC-ERR-230"
	ErrTokenNwkSnKeyLength             = "KC-GRPC-ERR-231"
	ErrTokenShortAddressZero           = "KC-GRPC-ERR-232"
	ErrTokenShortAddressOverflow       = "KC-GRPC-ERR-233"
	ErrTokenFormatOverflow             = "KC-GRPC-ERR-234"
	ErrTokenULTransmitFailed           = "KC-GRPC-ERR-235"
	ErrTokenAppKeyLength               = "KC-GRPC-ERR-236"

	// DL RX Status errors (KC-GRPC-ERR-240 to KC-GRPC-ERR-250)
	ErrTokenDLRxStatusFailed             = "KC-GRPC-ERR-240"
	ErrTokenDLRxStatusQueryHistoryFailed = "KC-GRPC-ERR-241"
	ErrTokenLimitNonNegative             = "KC-GRPC-ERR-242"
	ErrTokenOffsetNonNegative            = "KC-GRPC-ERR-243"
	ErrTokenEUIHexLengthInvalid          = "KC-GRPC-ERR-244"
	ErrTokenEUIHexCharsInvalid           = "KC-GRPC-ERR-245"
	ErrTokenEndpointSessionLookupFailed  = "KC-GRPC-ERR-246"

	// System/Release errors (KC-GRPC-ERR-250 to KC-GRPC-ERR-260)
	ErrTokenReleaseManifestFailed = "KC-GRPC-ERR-250"

	// BSSCI Session errors (KC-GRPC-ERR-260 to KC-GRPC-ERR-270)
	ErrTokenBSSessionNotFound   = "KC-GRPC-ERR-260"
	ErrTokenHandshakeIncomplete = "KC-GRPC-ERR-261"

	// Events/Alerts errors (KC-GRPC-ERR-081 to KC-GRPC-ERR-090)
	ErrTokenEventNotFound = "KC-GRPC-ERR-081"
	ErrTokenAlertNotFound = "KC-GRPC-ERR-082"

	// Base Station errors (KC-GRPC-ERR-091 to KC-GRPC-ERR-100)
	ErrTokenBaseStationNotFound        = "KC-GRPC-ERR-091"
	ErrTokenBaseStationOffline         = "KC-GRPC-ERR-092"
	ErrTokenBaseStationRequired        = "KC-GRPC-ERR-093"
	ErrTokenCreateBaseStationFailed    = "KC-GRPC-ERR-094"
	ErrTokenGetBaseStationFailed       = "KC-GRPC-ERR-095"
	ErrTokenUpdateBaseStationFailed    = "KC-GRPC-ERR-096"
	ErrTokenDeleteBaseStationFailed    = "KC-GRPC-ERR-097"
	ErrTokenListBaseStationsFailed     = "KC-GRPC-ERR-098"
	ErrTokenGetBaseStationStatsFailed  = "KC-GRPC-ERR-099"
	ErrTokenTenantAccessDenied         = "KC-GRPC-ERR-09A"
	ErrTokenBaseStationEUIExists       = "KC-GRPC-ERR-09B"
	ErrTokenUpdateBaseStationEUIFailed = "KC-GRPC-ERR-09C"
	ErrTokenNewBaseStationEUIRequired  = "KC-GRPC-ERR-09D"
	ErrTokenLatLonPairRequired         = "KC-GRPC-ERR-09E"

	// Endpoint errors (KC-GRPC-ERR-101 to KC-GRPC-ERR-110)
	ErrTokenEndpointNotFound        = "KC-GRPC-ERR-101"
	ErrTokenEndpointExists          = "KC-GRPC-ERR-102"
	ErrTokenEndpointRequired        = "KC-GRPC-ERR-103"
	ErrTokenCreateEndpointFailed    = "KC-GRPC-ERR-104"
	ErrTokenGetEndpointFailed       = "KC-GRPC-ERR-105"
	ErrTokenUpdateEndpointFailed    = "KC-GRPC-ERR-106"
	ErrTokenDeleteEndpointFailed    = "KC-GRPC-ERR-107"
	ErrTokenListEndpointsFailed     = "KC-GRPC-ERR-108"
	ErrTokenInvalidPageToken        = "KC-GRPC-ERR-109"
	ErrTokenTenantContextRequired   = "KC-GRPC-ERR-10A"
	ErrTokenAttachFailed            = "KC-GRPC-ERR-10B" // Endpoint attach operation failed
	ErrTokenDetachFailed            = "KC-GRPC-ERR-10C"
	ErrTokenUpdateEndpointEUIFailed = "KC-GRPC-ERR-10D"
	ErrTokenResolveTypeEUIFailed    = "KC-GRPC-ERR-10E"

	// Required field errors (KC-GRPC-ERR-111 to KC-GRPC-ERR-130)
	ErrTokenEmailRequired          = "KC-GRPC-ERR-111"
	ErrTokenPasswordRequired       = "KC-GRPC-ERR-112"
	ErrTokenNameRequired           = "KC-GRPC-ERR-113"
	ErrTokenIDRequired             = "KC-GRPC-ERR-114"
	ErrTokenEUIRequired            = "KC-GRPC-ERR-115"
	ErrTokenEndpointEUIRequired    = "KC-GRPC-ERR-116"
	ErrTokenBasestationEUIRequired = "KC-GRPC-ERR-117"
	ErrTokenOrgIDRequired          = "KC-GRPC-ERR-118"
	ErrTokenUserIDRequired         = "KC-GRPC-ERR-119"
	ErrTokenKeyTypeRequired        = "KC-GRPC-ERR-120"
	ErrTokenManufacturerIDRequired = "KC-GRPC-ERR-121"
	ErrTokenDeviceModelIDRequired  = "KC-GRPC-ERR-122"
	ErrTokenBlueprintIDRequired    = "KC-GRPC-ERR-123"
	ErrTokenVersionRequired        = "KC-GRPC-ERR-124"
	ErrTokenCodeRequired           = "KC-GRPC-ERR-125"
	ErrTokenQueryRequired          = "KC-GRPC-ERR-126"
	ErrTokenFormatRequired         = "KC-GRPC-ERR-127"
	ErrTokenMessageIDRequired      = "KC-GRPC-ERR-128"
	ErrTokenSessionIDRequired      = "KC-GRPC-ERR-129"
	ErrTokenStateRequired          = "KC-GRPC-ERR-130" // External auth exchange state parameter

	// Invalid format errors (KC-GRPC-ERR-131 to KC-GRPC-ERR-150)
	ErrTokenInvalidUserIDFormat         = "KC-GRPC-ERR-131"
	ErrTokenInvalidOrgIDFormat          = "KC-GRPC-ERR-132"
	ErrTokenInvalidEndpointEUIFormat    = "KC-GRPC-ERR-133"
	ErrTokenInvalidBasestationEUIFormat = "KC-GRPC-ERR-134"
	ErrTokenInvalidAPIKeyIDFormat       = "KC-GRPC-ERR-135"
	ErrTokenInvalidManufacturerIDFormat = "KC-GRPC-ERR-136"
	ErrTokenInvalidDeviceModelIDFormat  = "KC-GRPC-ERR-137"
	ErrTokenInvalidBlueprintIDFormat    = "KC-GRPC-ERR-138"
	ErrTokenInvalidSessionIDFormat      = "KC-GRPC-ERR-139"
	ErrTokenInvalidMessageIDFormat      = "KC-GRPC-ERR-140"
	ErrTokenInvalidEUIFormat            = "KC-GRPC-ERR-141"
	ErrTokenInvalidIDFormat             = "KC-GRPC-ERR-142"
	ErrTokenInvalidRoleFormat           = "KC-GRPC-ERR-143"
	ErrTokenInvalidExportFormat         = "KC-GRPC-ERR-144"

	// Gateway errors (KC-GRPC-ERR-600 to KC-GRPC-ERR-610)
	ErrTokenGatewayUpstreamUnavailable        = "KC-GRPC-ERR-600"
	ErrTokenGatewayProxyFailed                = "KC-GRPC-ERR-601"
	ErrTokenGatewayInternalTrustInvalidHeader = "KC-GRPC-ERR-602"

	// Generic errors (KC-GRPC-ERR-900+)
	ErrTokenInternalError        = "KC-GRPC-ERR-900"
	ErrTokenInvalidArgument      = "KC-GRPC-ERR-901"
	ErrTokenNotImplemented       = "KC-GRPC-ERR-902"
	ErrTokenServiceUnavailable   = "KC-GRPC-ERR-903"
	ErrTokenDatabaseError        = "KC-GRPC-ERR-904"
	ErrTokenTenantRequired       = "KC-GRPC-ERR-905"
	ErrTokenServiceNotConfigured = "KC-GRPC-ERR-906"
	ErrTokenOrgRequired          = "KC-GRPC-ERR-907"
	ErrTokenUserRequired         = "KC-GRPC-ERR-908"
	ErrTokenContextMissing       = "KC-GRPC-ERR-909"
	ErrTokenCEStatusUnavailable     = "KC-GRPC-ERR-90A"
	ErrTokenCEOnboardingUnavailable = "KC-GRPC-ERR-90B"
	ErrTokenCERegistryUnavailable   = "KC-GRPC-ERR-90C"
)

// ErrorDefinition maps error tokens to messages and gRPC codes
// This enables centralized error tracking and handling
type ErrorDefinition struct {
	Token   string     // Error token (e.g., "KC-GRPC-ERR-001")
	Message string     // Human-readable message
	Code    codes.Code // gRPC status code
}

// errorCatalog maps error tokens to full metadata
var errorCatalog = map[string]ErrorDefinition{
	// Authentication errors
	ErrTokenInvalidCredentials: {
		Token:   ErrTokenInvalidCredentials,
		Message: "invalid email or password",
		Code:    codes.Unauthenticated,
	},
	ErrTokenEmailRequired: {
		Token:   ErrTokenEmailRequired,
		Message: "email is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenPasswordRequired: {
		Token:   ErrTokenPasswordRequired,
		Message: "password is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenUserNotFound: {
		Token:   ErrTokenUserNotFound,
		Message: "user not found",
		Code:    codes.NotFound,
	},
	ErrTokenInvalidRefreshToken: {
		Token:   ErrTokenInvalidRefreshToken,
		Message: "invalid or expired refresh token",
		Code:    codes.Unauthenticated,
	},
	ErrTokenSessionExpired: {
		Token:   ErrTokenSessionExpired,
		Message: "session has expired",
		Code:    codes.Unauthenticated,
	},
	ErrTokenPasswordMismatch: {
		Token:   ErrTokenPasswordMismatch,
		Message: "current password is incorrect",
		Code:    codes.InvalidArgument,
	},
	ErrTokenUserNotActive: {
		Token:   ErrTokenUserNotActive,
		Message: "user account is not active",
		Code:    codes.PermissionDenied,
	},
	ErrTokenEmailAlreadyExists: {
		Token:   ErrTokenEmailAlreadyExists,
		Message: "email address already registered",
		Code:    codes.AlreadyExists,
	},
	ErrTokenWeakPassword: {
		Token:   ErrTokenWeakPassword,
		Message: "password does not meet security requirements",
		Code:    codes.InvalidArgument,
	},
	ErrTokenTokenGenerationFail: {
		Token:   ErrTokenTokenGenerationFail,
		Message: "failed to generate authentication token",
		Code:    codes.Internal,
	},
	ErrTokenUnauthorized: {
		Token:   ErrTokenUnauthorized,
		Message: "unauthorized access",
		Code:    codes.Unauthenticated,
	},

	// Auth handler operation errors
	ErrTokenRefreshTokenRequired: {
		Token:   ErrTokenRefreshTokenRequired,
		Message: "refresh_token is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCurrentPasswordRequired: {
		Token:   ErrTokenCurrentPasswordRequired,
		Message: "current_password is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenNewPasswordRequired: {
		Token:   ErrTokenNewPasswordRequired,
		Message: "new_password is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenGetProfileFailed: {
		Token:   ErrTokenGetProfileFailed,
		Message: "failed to get profile",
		Code:    codes.Internal,
	},
	ErrTokenGetAuthSettingsFailed: {
		Token:   ErrTokenGetAuthSettingsFailed,
		Message: "failed to get auth settings",
		Code:    codes.Internal,
	},
	ErrTokenLogoutFailed: {
		Token:   ErrTokenLogoutFailed,
		Message: "failed to logout",
		Code:    codes.Internal,
	},
	ErrTokenChangePasswordFailed: {
		Token:   ErrTokenChangePasswordFailed,
		Message: "failed to change password",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCreateUserFailed: {
		Token:   ErrTokenCreateUserFailed,
		Message: "failed to create user",
		Code:    codes.Internal,
	},
	ErrTokenUpdateUserFailed: {
		Token:   ErrTokenUpdateUserFailed,
		Message: "failed to update user",
		Code:    codes.Internal,
	},
	ErrTokenDeleteUserFailed: {
		Token:   ErrTokenDeleteUserFailed,
		Message: "failed to delete user",
		Code:    codes.Internal,
	},
	ErrTokenListUsersFailed: {
		Token:   ErrTokenListUsersFailed,
		Message: "failed to list users",
		Code:    codes.Internal,
	},
	ErrTokenUpdatePasswordFailed: {
		Token:   ErrTokenUpdatePasswordFailed,
		Message: "failed to update password",
		Code:    codes.Internal,
	},

	// Auth interceptor errors
	ErrTokenMissingMetadata: {
		Token:   ErrTokenMissingMetadata,
		Message: "missing metadata",
		Code:    codes.Unauthenticated,
	},
	ErrTokenMissingAuthToken: {
		Token:   ErrTokenMissingAuthToken,
		Message: "missing authorization token",
		Code:    codes.Unauthenticated,
	},
	ErrTokenInvalidAuthFormat: {
		Token:   ErrTokenInvalidAuthFormat,
		Message: "invalid authorization format",
		Code:    codes.Unauthenticated,
	},
	ErrTokenInvalidToken: {
		Token:   ErrTokenInvalidToken,
		Message: "invalid token",
		Code:    codes.Unauthenticated,
	},
	ErrTokenInvalidTokenIssuer: {
		Token:   ErrTokenInvalidTokenIssuer,
		Message: "invalid token issuer",
		Code:    codes.Unauthenticated,
	},
	ErrTokenMissingAudience: {
		Token:   ErrTokenMissingAudience,
		Message: "missing audience",
		Code:    codes.Unauthenticated,
	},
	ErrTokenInvalidAudience: {
		Token:   ErrTokenInvalidAudience,
		Message: "invalid audience",
		Code:    codes.Unauthenticated,
	},
	ErrTokenMissingTenantClaim: {
		Token:   ErrTokenMissingTenantClaim,
		Message: "missing tenant claim",
		Code:    codes.Unauthenticated,
	},
	ErrTokenMissingTenantCtx: {
		Token:   ErrTokenMissingTenantCtx,
		Message: "no tenant in context",
		Code:    codes.Unauthenticated,
	},
	ErrTokenMissingUserCtx: {
		Token:   ErrTokenMissingUserCtx,
		Message: "no user in context",
		Code:    codes.Unauthenticated,
	},
	// Fail-closed tenant isolation when auth disabled
	ErrTokenTenantHeaderRequired: {
		Token:   ErrTokenTenantHeaderRequired,
		Message: "x-tenant-id header required",
		Code:    codes.Unauthenticated,
	},

	// Organization errors
	ErrTokenOrgNotFound: {
		Token:   ErrTokenOrgNotFound,
		Message: "organization not found",
		Code:    codes.NotFound,
	},
	ErrTokenOrgAlreadyExists: {
		Token:   ErrTokenOrgAlreadyExists,
		Message: "organization already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenMembershipExists: {
		Token:   ErrTokenMembershipExists,
		Message: "user is already a member of this organization",
		Code:    codes.AlreadyExists,
	},
	ErrTokenMembershipNotFound: {
		Token:   ErrTokenMembershipNotFound,
		Message: "membership not found",
		Code:    codes.NotFound,
	},
	ErrTokenMembershipRequired: {
		Token:   ErrTokenMembershipRequired,
		Message: "organization membership is required",
		Code:    codes.PermissionDenied,
	},
	ErrTokenInvalidOrgRole: {
		Token:   ErrTokenInvalidOrgRole,
		Message: "invalid organization role",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCannotRemoveSelf: {
		Token:   ErrTokenCannotRemoveSelf,
		Message: "cannot remove yourself from the organization",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenOrgHasMembers: {
		Token:   ErrTokenOrgHasMembers,
		Message: "cannot delete organization with existing members",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenCannotRemoveLastOwner: {
		Token:   ErrTokenCannotRemoveLastOwner,
		Message: "cannot remove the last active owner of the organization",
		Code:    codes.Aborted,
	},
	ErrTokenAdminRequired: {
		Token:   ErrTokenAdminRequired,
		Message: "admin privileges required",
		Code:    codes.PermissionDenied,
	},
	ErrTokenOrgContextMismatch: {
		Token:   ErrTokenOrgContextMismatch,
		Message: "request organization does not match authenticated context",
		Code:    codes.PermissionDenied,
	},
	ErrTokenUpdateMaskRequired: {
		Token:   ErrTokenUpdateMaskRequired,
		Message: "update_mask is required",
		Code:    codes.InvalidArgument,
	},

	// API Key errors
	ErrTokenApiKeyNotFound: {
		Token:   ErrTokenApiKeyNotFound,
		Message: "API key not found",
		Code:    codes.NotFound,
	},
	ErrTokenApiKeyExpired: {
		Token:   ErrTokenApiKeyExpired,
		Message: "API key has expired",
		Code:    codes.Unauthenticated,
	},
	ErrTokenApiKeyInactive: {
		Token:   ErrTokenApiKeyInactive,
		Message: "API key is not active",
		Code:    codes.PermissionDenied,
	},
	ErrTokenApiKeyNameExists: {
		Token:   ErrTokenApiKeyNameExists,
		Message: "API key with this name already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenInvalidApiKeyType: {
		Token:   ErrTokenInvalidApiKeyType,
		Message: "invalid API key type",
		Code:    codes.InvalidArgument,
	},

	// Organization operation errors
	ErrTokenCreateOrgFailed: {
		Token:   ErrTokenCreateOrgFailed,
		Message: "failed to create organization",
		Code:    codes.Internal,
	},
	ErrTokenUpdateOrgFailed: {
		Token:   ErrTokenUpdateOrgFailed,
		Message: "failed to update organization",
		Code:    codes.Internal,
	},
	ErrTokenDeleteOrgFailed: {
		Token:   ErrTokenDeleteOrgFailed,
		Message: "failed to delete organization",
		Code:    codes.Internal,
	},
	ErrTokenListOrgsFailed: {
		Token:   ErrTokenListOrgsFailed,
		Message: "failed to list organizations",
		Code:    codes.Internal,
	},
	ErrTokenRoleRequired: {
		Token:   ErrTokenRoleRequired,
		Message: "role is required",
		Code:    codes.InvalidArgument,
	},

	// Membership operation errors
	ErrTokenAddMemberFailed: {
		Token:   ErrTokenAddMemberFailed,
		Message: "failed to add user to organization",
		Code:    codes.Internal,
	},
	ErrTokenGetMemberFailed: {
		Token:   ErrTokenGetMemberFailed,
		Message: "failed to get membership",
		Code:    codes.Internal,
	},
	ErrTokenUpdateMemberFailed: {
		Token:   ErrTokenUpdateMemberFailed,
		Message: "failed to update membership",
		Code:    codes.Internal,
	},
	ErrTokenRemoveMemberFailed: {
		Token:   ErrTokenRemoveMemberFailed,
		Message: "failed to remove user from organization",
		Code:    codes.Internal,
	},
	ErrTokenListMembersFailed: {
		Token:   ErrTokenListMembersFailed,
		Message: "failed to list organization members",
		Code:    codes.Internal,
	},
	ErrTokenMemberAddedRetrieveFail: {
		Token:   ErrTokenMemberAddedRetrieveFail,
		Message: "member added but failed to retrieve",
		Code:    codes.Internal,
	},
	ErrTokenMemberUpdatedRetrieveFail: {
		Token:   ErrTokenMemberUpdatedRetrieveFail,
		Message: "role updated but failed to retrieve",
		Code:    codes.Internal,
	},

	// API Key operation errors
	ErrTokenCreateApiKeyFailed: {
		Token:   ErrTokenCreateApiKeyFailed,
		Message: "failed to create API key",
		Code:    codes.Internal,
	},
	ErrTokenDeleteApiKeyFailed: {
		Token:   ErrTokenDeleteApiKeyFailed,
		Message: "failed to delete API key",
		Code:    codes.Internal,
	},
	ErrTokenListApiKeysFailed: {
		Token:   ErrTokenListApiKeysFailed,
		Message: "failed to list API keys",
		Code:    codes.Internal,
	},

	// Certificate errors
	ErrTokenCertGenerationFailed: {
		Token:   ErrTokenCertGenerationFailed,
		Message: "failed to generate certificate",
		Code:    codes.Internal,
	},
	ErrTokenCertNotFound: {
		Token:   ErrTokenCertNotFound,
		Message: "certificate not found",
		Code:    codes.NotFound,
	},
	ErrTokenCertExpired: {
		Token:   ErrTokenCertExpired,
		Message: "certificate has expired",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenCertInvalid: {
		Token:   ErrTokenCertInvalid,
		Message: "certificate is invalid",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCertRenewalFailed: {
		Token:   ErrTokenCertRenewalFailed,
		Message: "failed to renew certificate",
		Code:    codes.Internal,
	},
	ErrTokenCertTypeRequired: {
		Token:   ErrTokenCertTypeRequired,
		Message: "cert_type is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCertStatusFailed: {
		Token:   ErrTokenCertStatusFailed,
		Message: "failed to get certificate status",
		Code:    codes.Internal,
	},
	ErrTokenCertServerGenerationFailed: {
		Token:   ErrTokenCertServerGenerationFailed,
		Message: "failed to generate server certificates",
		Code:    codes.Internal,
	},
	ErrTokenCertDirCreateFailed: {
		Token:   ErrTokenCertDirCreateFailed,
		Message: "failed to create certificate directory",
		Code:    codes.Internal,
	},
	ErrTokenCertGeneratorNotFound: {
		Token:   ErrTokenCertGeneratorNotFound,
		Message: "certificate generator binary not found",
		Code:    codes.Internal,
	},
	ErrTokenCACertReadFailed: {
		Token:   ErrTokenCACertReadFailed,
		Message: "failed to read CA certificate",
		Code:    codes.Internal,
	},
	ErrTokenCACertCopyFailed: {
		Token:   ErrTokenCACertCopyFailed,
		Message: "failed to copy CA certificate",
		Code:    codes.Internal,
	},
	ErrTokenCAKeyReadFailed: {
		Token:   ErrTokenCAKeyReadFailed,
		Message: "failed to read CA private key",
		Code:    codes.Internal,
	},
	ErrTokenCAKeyCopyFailed: {
		Token:   ErrTokenCAKeyCopyFailed,
		Message: "failed to copy CA private key",
		Code:    codes.Internal,
	},
	ErrTokenInvalidValidityPeriod: {
		Token:   ErrTokenInvalidValidityPeriod,
		Message: "invalid certificate validity period (must be 1-1095 days)",
		Code:    codes.InvalidArgument,
	},
	ErrTokenNoCertsToRenew: {
		Token:   ErrTokenNoCertsToRenew,
		Message: "no existing certificates found to renew",
		Code:    codes.FailedPrecondition,
	},

	// Blueprint/Catalog errors
	ErrTokenManufacturerNotFound: {
		Token:   ErrTokenManufacturerNotFound,
		Message: "manufacturer not found",
		Code:    codes.NotFound,
	},
	ErrTokenManufacturerNameExists: {
		Token:   ErrTokenManufacturerNameExists,
		Message: "manufacturer with this name already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenDeviceModelNotFound: {
		Token:   ErrTokenDeviceModelNotFound,
		Message: "device model not found",
		Code:    codes.NotFound,
	},
	ErrTokenDeviceModelNameExists: {
		Token:   ErrTokenDeviceModelNameExists,
		Message: "device model with this name already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenBlueprintNotFound: {
		Token:   ErrTokenBlueprintNotFound,
		Message: "blueprint not found",
		Code:    codes.NotFound,
	},
	ErrTokenBlueprintNameExists: {
		Token:   ErrTokenBlueprintNameExists,
		Message: "blueprint with this name already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenBlueprintInvalid: {
		Token:   ErrTokenBlueprintInvalid,
		Message: "blueprint configuration is invalid",
		Code:    codes.InvalidArgument,
	},
	ErrTokenRegistrySubmitFailed: {
		Token:   ErrTokenRegistrySubmitFailed,
		Message: "failed to submit blueprint to registry",
		Code:    codes.Internal,
	},
	ErrTokenManufacturerHasModels: {
		Token:   ErrTokenManufacturerHasModels,
		Message: "cannot delete manufacturer with existing device models",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenDeviceModelHasBlueprints: {
		Token:   ErrTokenDeviceModelHasBlueprints,
		Message: "cannot delete device model with existing blueprints",
		Code:    codes.FailedPrecondition,
	},

	// Blueprint operation errors
	ErrTokenCreateManufacturerFailed: {
		Token:   ErrTokenCreateManufacturerFailed,
		Message: "failed to create manufacturer",
		Code:    codes.Internal,
	},
	ErrTokenUpdateManufacturerFailed: {
		Token:   ErrTokenUpdateManufacturerFailed,
		Message: "failed to update manufacturer",
		Code:    codes.Internal,
	},
	ErrTokenDeleteManufacturerFailed: {
		Token:   ErrTokenDeleteManufacturerFailed,
		Message: "failed to delete manufacturer",
		Code:    codes.Internal,
	},
	ErrTokenListManufacturersFailed: {
		Token:   ErrTokenListManufacturersFailed,
		Message: "failed to list manufacturers",
		Code:    codes.Internal,
	},
	ErrTokenCreateDeviceModelFailed: {
		Token:   ErrTokenCreateDeviceModelFailed,
		Message: "failed to create device model",
		Code:    codes.Internal,
	},
	ErrTokenUpdateDeviceModelFailed: {
		Token:   ErrTokenUpdateDeviceModelFailed,
		Message: "failed to update device model",
		Code:    codes.Internal,
	},
	ErrTokenDeleteDeviceModelFailed: {
		Token:   ErrTokenDeleteDeviceModelFailed,
		Message: "failed to delete device model",
		Code:    codes.Internal,
	},
	ErrTokenListDeviceModelsFailed: {
		Token:   ErrTokenListDeviceModelsFailed,
		Message: "failed to list device models",
		Code:    codes.Internal,
	},
	ErrTokenCreateBlueprintFailed: {
		Token:   ErrTokenCreateBlueprintFailed,
		Message: "failed to create blueprint",
		Code:    codes.Internal,
	},
	ErrTokenUpdateBlueprintFailed: {
		Token:   ErrTokenUpdateBlueprintFailed,
		Message: "failed to update blueprint",
		Code:    codes.Internal,
	},
	ErrTokenDeleteBlueprintFailed: {
		Token:   ErrTokenDeleteBlueprintFailed,
		Message: "failed to delete blueprint",
		Code:    codes.Internal,
	},
	ErrTokenListBlueprintsFailed: {
		Token:   ErrTokenListBlueprintsFailed,
		Message: "failed to list blueprints",
		Code:    codes.Internal,
	},
	ErrTokenSetDefaultBlueprintFailed: {
		Token:   ErrTokenSetDefaultBlueprintFailed,
		Message: "failed to set default blueprint",
		Code:    codes.Internal,
	},
	ErrTokenInvalidTypeEUIFormat: {
		Token:   ErrTokenInvalidTypeEUIFormat,
		Message: "invalid type_eui hex format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenSubmitBlueprintToRegistryFailed: {
		Token:   ErrTokenSubmitBlueprintToRegistryFailed,
		Message: "failed to submit blueprint to registry",
		Code:    codes.Internal,
	},
	ErrTokenAtomicModelBlueprintFailed: {
		Token:   ErrTokenAtomicModelBlueprintFailed,
		Message: "failed to create device model with blueprint atomically",
		Code:    codes.Internal,
	},

	// Blueprint preview errors
	ErrTokenSlugGenerationFailed: {
		Token:   ErrTokenSlugGenerationFailed,
		Message: "slug generation failed after exhausting all suffix attempts",
		Code:    codes.Internal,
	},
	ErrTokenPreviewPayloadRequired: {
		Token:   ErrTokenPreviewPayloadRequired,
		Message: "preview request requires a payload",
		Code:    codes.InvalidArgument,
	},
	ErrTokenPreviewDecodeFailed: {
		Token:   ErrTokenPreviewDecodeFailed,
		Message: "blueprint decoder returned an error",
		Code:    codes.Internal,
	},
	ErrTokenPreviewInvalidFormatID: {
		Token:   ErrTokenPreviewInvalidFormatID,
		Message: "format ID not recognized by decoder",
		Code:    codes.InvalidArgument,
	},
	ErrTokenPreviewInvalidPayload: {
		Token:   ErrTokenPreviewInvalidPayload,
		Message: "payload bytes failed validation",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidModelCode: {
		Token:   ErrTokenInvalidModelCode,
		Message: "invalid model code: must contain only lowercase alphanumeric characters and hyphens",
		Code:    codes.InvalidArgument,
	},

	// Integration operation errors
	ErrTokenCreateIntegrationFailed: {
		Token:   ErrTokenCreateIntegrationFailed,
		Message: "failed to create integration",
		Code:    codes.Internal,
	},
	ErrTokenGetIntegrationFailed: {
		Token:   ErrTokenGetIntegrationFailed,
		Message: "failed to get integration",
		Code:    codes.Internal,
	},
	ErrTokenUpdateIntegrationFailed: {
		Token:   ErrTokenUpdateIntegrationFailed,
		Message: "failed to update integration",
		Code:    codes.Internal,
	},
	ErrTokenDeleteIntegrationFailed: {
		Token:   ErrTokenDeleteIntegrationFailed,
		Message: "failed to delete integration",
		Code:    codes.Internal,
	},
	ErrTokenListIntegrationsFailed: {
		Token:   ErrTokenListIntegrationsFailed,
		Message: "failed to list integrations",
		Code:    codes.Internal,
	},
	ErrTokenIntegrationNotFound: {
		Token:   ErrTokenIntegrationNotFound,
		Message: "integration not found",
		Code:    codes.NotFound,
	},
	ErrTokenMissingOrgContext: {
		Token:   ErrTokenMissingOrgContext,
		Message: "organization context is missing",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenInvalidRequest: {
		Token:   ErrTokenInvalidRequest,
		Message: "invalid request parameters",
		Code:    codes.InvalidArgument,
	},

	// Endpoint Stats/Operations errors
	ErrTokenGetEndpointStatsFailed: {
		Token:   ErrTokenGetEndpointStatsFailed,
		Message: "failed to get endpoint statistics",
		Code:    codes.Internal,
	},
	ErrTokenGetEndpointOperationsFailed: {
		Token:   ErrTokenGetEndpointOperationsFailed,
		Message: "failed to get endpoint operations",
		Code:    codes.Internal,
	},
	ErrTokenConfigRequired: {
		Token:   ErrTokenConfigRequired,
		Message: "config is required",
		Code:    codes.InvalidArgument,
	},

	// Registry Provider Errors
	ErrTokenRegistryProviderDisabled: {
		Token:   ErrTokenRegistryProviderDisabled,
		Message: "registry provider is not enabled",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenRegistryAuthFailed: {
		Token:   ErrTokenRegistryAuthFailed,
		Message: "registry authentication failed",
		Code:    codes.Unauthenticated,
	},
	ErrTokenRegistryRateLimited: {
		Token:   ErrTokenRegistryRateLimited,
		Message: "registry API rate limit exceeded",
		Code:    codes.ResourceExhausted,
	},
	ErrTokenRegistryBranchCreateFailed: {
		Token:   ErrTokenRegistryBranchCreateFailed,
		Message: "failed to create registry branch",
		Code:    codes.Internal,
	},
	ErrTokenRegistryCommitFailed: {
		Token:   ErrTokenRegistryCommitFailed,
		Message: "failed to commit to registry",
		Code:    codes.Internal,
	},
	ErrTokenRegistrySubmissionCreateFailed: {
		Token:   ErrTokenRegistrySubmissionCreateFailed,
		Message: "failed to create submission request",
		Code:    codes.Internal,
	},
	ErrTokenRegistryAPIError: {
		Token:   ErrTokenRegistryAPIError,
		Message: "registry API error",
		Code:    codes.Internal,
	},
	ErrTokenBlueprintAlreadySubmitted: {
		Token:   ErrTokenBlueprintAlreadySubmitted,
		Message: "blueprint has already been submitted to registry",
		Code:    codes.AlreadyExists,
	},
	ErrTokenContributorNameRequired: {
		Token:   ErrTokenContributorNameRequired,
		Message: "contributor_name is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenContributorEmailRequired: {
		Token:   ErrTokenContributorEmailRequired,
		Message: "contributor_email is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenRegistryAPIURLRequired: {
		Token:   ErrTokenRegistryAPIURLRequired,
		Message: "registry api_url is required when enabled",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenRegistryPermissionDenied: {
		Token:   ErrTokenRegistryPermissionDenied,
		Message: "registry token lacks required permissions — verify the token has repo and pull-request scopes",
		Code:    codes.PermissionDenied,
	},
	ErrTokenRegistryVersionAlreadyExists: {
		Token:   ErrTokenRegistryVersionAlreadyExists,
		Message: "this blueprint version already exists in the registry — use a different version",
		Code:    codes.AlreadyExists,
	},
	ErrTokenRegistryInvalidPathSegment: {
		Token:   ErrTokenRegistryInvalidPathSegment,
		Message: "registry path segment is invalid — check manufacturer name, model code, or version",
		Code:    codes.InvalidArgument,
	},

	// SCACI Monitoring errors
	ErrTokenScaciSessionNotFound: {
		Token:   ErrTokenScaciSessionNotFound,
		Message: "SCACI session not found",
		Code:    codes.NotFound,
	},
	ErrTokenScaciQueueNotFound: {
		Token:   ErrTokenScaciQueueNotFound,
		Message: "SCACI queue entry not found",
		Code:    codes.NotFound,
	},
	ErrTokenScaciServiceOffline: {
		Token:   ErrTokenScaciServiceOffline,
		Message: "SCACI service is offline",
		Code:    codes.Unavailable,
	},
	ErrTokenScaciListSessionsFailed: {
		Token:   ErrTokenScaciListSessionsFailed,
		Message: "failed to list SCACI sessions",
		Code:    codes.Internal,
	},
	ErrTokenScaciStatisticsFailed: {
		Token:   ErrTokenScaciStatisticsFailed,
		Message: "failed to get SCACI statistics",
		Code:    codes.Internal,
	},
	ErrTokenScaciListErrorsFailed: {
		Token:   ErrTokenScaciListErrorsFailed,
		Message: "failed to list SCACI errors",
		Code:    codes.Internal,
	},
	ErrTokenScaciListQueuesFailed: {
		Token:   ErrTokenScaciListQueuesFailed,
		Message: "failed to list SCACI queues",
		Code:    codes.Internal,
	},
	ErrTokenScaciStatusFailed: {
		Token:   ErrTokenScaciStatusFailed,
		Message: "failed to get SCACI status",
		Code:    codes.Internal,
	},

	// Auth claim resolution errors
	ErrTokenTenantResolverRequired: {
		Token:   ErrTokenTenantResolverRequired,
		Message: "tenant resolver required for org UUID claims",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenOrgResolutionFailed: {
		Token:   ErrTokenOrgResolutionFailed,
		Message: "failed to resolve tenant from organization",
		Code:    codes.PermissionDenied,
	},

	// External auth errors
	ErrTokenAuthOIDCDisabled: {
		Token:   ErrTokenAuthOIDCDisabled,
		Message: "OIDC authentication is not enabled",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenAuthOAuth2Disabled: {
		Token:   ErrTokenAuthOAuth2Disabled,
		Message: "OAuth2 authentication is not enabled",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenAuthStateNotFound: {
		Token:   ErrTokenAuthStateNotFound,
		Message: "auth state not found or expired",
		Code:    codes.NotFound,
	},
	ErrTokenAuthTokenExchangeFailed: {
		Token:   ErrTokenAuthTokenExchangeFailed,
		Message: "token exchange failed",
		Code:    codes.Internal,
	},
	ErrTokenAuthIDTokenInvalid: {
		Token:   ErrTokenAuthIDTokenInvalid,
		Message: "ID token validation failed",
		Code:    codes.Unauthenticated,
	},
	ErrTokenAuthEmailNotVerified: {
		Token:   ErrTokenAuthEmailNotVerified,
		Message: "email is not verified",
		Code:    codes.PermissionDenied,
	},
	ErrTokenAuthRegistrationDisabled: {
		Token:   ErrTokenAuthRegistrationDisabled,
		Message: "user registration via external auth is disabled",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenAuthRedisUnavailable: {
		Token:   ErrTokenAuthRedisUnavailable,
		Message: "state storage unavailable",
		Code:    codes.Unavailable,
	},
	ErrTokenAuthUserInfoFailed: {
		Token:   ErrTokenAuthUserInfoFailed,
		Message: "userinfo request failed",
		Code:    codes.Internal,
	},
	ErrTokenAuthStateInvalid: {
		Token:   ErrTokenAuthStateInvalid,
		Message: "invalid auth state",
		Code:    codes.InvalidArgument,
	},
	ErrTokenAuthExternalOrgFailed: {
		Token:   ErrTokenAuthExternalOrgFailed,
		Message: "external org claim resolution failed",
		Code:    codes.PermissionDenied,
	},

	// Self-service registration errors
	ErrTokenRegistrationDisabled: {
		Token:   ErrTokenRegistrationDisabled,
		Message: "registration is not available",
		Code:    codes.InvalidArgument,
	},
	ErrTokenFirstNameRequired: {
		Token:   ErrTokenFirstNameRequired,
		Message: "first name is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenLastNameRequired: {
		Token:   ErrTokenLastNameRequired,
		Message: "last name is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCompanyNameRequired: {
		Token:   ErrTokenCompanyNameRequired,
		Message: "company name is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenRegistrationFailed: {
		Token:   ErrTokenRegistrationFailed,
		Message: "account registration failed",
		Code:    codes.Internal,
	},
	ErrTokenOrgAdminRequired: {
		Token:   ErrTokenOrgAdminRequired,
		Message: "organization admin access required",
		Code:    codes.PermissionDenied,
	},
	ErrTokenUserEmailRequired: {
		Token:   ErrTokenUserEmailRequired,
		Message: "either user ID or email is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenRateLimitExceeded: {
		Token:   ErrTokenRateLimitExceeded,
		Message: "too many requests, please try again later",
		Code:    codes.ResourceExhausted,
	},
	ErrTokenEmailInvalidFormat: {
		Token:   ErrTokenEmailInvalidFormat,
		Message: "invalid email format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenFieldTooLong: {
		Token:   ErrTokenFieldTooLong,
		Message: "field value exceeds maximum length",
		Code:    codes.InvalidArgument,
	},

	// SCACI operation mapping errors
	ErrTokenScaciCntDependMismatch: {
		Token:   ErrTokenScaciCntDependMismatch,
		Message: "counter dependency mismatch in downlink payload",
		Code:    codes.InvalidArgument,
	},
	ErrTokenScaciCntDependPacketCntOmit: {
		Token:   ErrTokenScaciCntDependPacketCntOmit,
		Message: "packet count must be omitted for counter-dependent payloads",
		Code:    codes.InvalidArgument,
	},
	ErrTokenScaciNonCntDependMultiPayload: {
		Token:   ErrTokenScaciNonCntDependMultiPayload,
		Message: "multiple payloads only allowed for counter-dependent downlinks",
		Code:    codes.InvalidArgument,
	},
	ErrTokenScaciQueueIDOutOfRange: {
		Token:   ErrTokenScaciQueueIDOutOfRange,
		Message: "queue ID out of valid range",
		Code:    codes.InvalidArgument,
	},
	ErrTokenScaciFailedPersistDownlink: {
		Token:   ErrTokenScaciFailedPersistDownlink,
		Message: "failed to persist downlink to queue",
		Code:    codes.InvalidArgument,
	},
	ErrTokenScaciQueueIDExists: {
		Token:   ErrTokenScaciQueueIDExists,
		Message: "queue ID already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenScaciSchedulerUnavailable: {
		Token:   ErrTokenScaciSchedulerUnavailable,
		Message: "scheduler service unavailable",
		Code:    codes.Unavailable,
	},
	ErrTokenScaciBaseStationUnavailable: {
		Token:   ErrTokenScaciBaseStationUnavailable,
		Message: "base station unavailable for downlink",
		Code:    codes.Unavailable,
	},
	ErrTokenScaciEndpointNotFound: {
		Token:   ErrTokenScaciEndpointNotFound,
		Message: "endpoint not found for SCACI operation",
		Code:    codes.NotFound,
	},
	ErrTokenScaciDownlinkNotFound: {
		Token:   ErrTokenScaciDownlinkNotFound,
		Message: "downlink entry not found",
		Code:    codes.NotFound,
	},
	ErrTokenScaciOperationFailed: {
		Token:   ErrTokenScaciOperationFailed,
		Message: "SCACI operation failed",
		Code:    codes.Internal,
	},

	// Message/Analytics errors
	ErrTokenMessageNotFound: {
		Token:   ErrTokenMessageNotFound,
		Message: "message not found",
		Code:    codes.NotFound,
	},
	ErrTokenInvalidTimeRange: {
		Token:   ErrTokenInvalidTimeRange,
		Message: "invalid time range specified",
		Code:    codes.InvalidArgument,
	},
	ErrTokenExportFailed: {
		Token:   ErrTokenExportFailed,
		Message: "failed to export data",
		Code:    codes.Internal,
	},
	ErrTokenStreamDisconnected: {
		Token:   ErrTokenStreamDisconnected,
		Message: "stream connection lost",
		Code:    codes.Unavailable,
	},
	ErrTokenAnalyticsOverviewFail: {
		Token:   ErrTokenAnalyticsOverviewFail,
		Message: "failed to get analytics overview",
		Code:    codes.Internal,
	},
	ErrTokenAnalyticsActivityFail: {
		Token:   ErrTokenAnalyticsActivityFail,
		Message: "failed to get activity analytics",
		Code:    codes.Internal,
	},
	ErrTokenAnalyticsSignalFail: {
		Token:   ErrTokenAnalyticsSignalFail,
		Message: "failed to get signal quality analytics",
		Code:    codes.Internal,
	},
	ErrTokenListEventsFailed: {
		Token:   ErrTokenListEventsFailed,
		Message: "failed to list events",
		Code:    codes.Internal,
	},
	ErrTokenListBSEventsFailed: {
		Token:   ErrTokenListBSEventsFailed,
		Message: "failed to list base station events",
		Code:    codes.Internal,
	},
	ErrTokenListEPEventsFailed: {
		Token:   ErrTokenListEPEventsFailed,
		Message: "failed to list endpoint events",
		Code:    codes.Internal,
	},
	ErrTokenListAlertsFailed: {
		Token:   ErrTokenListAlertsFailed,
		Message: "failed to list alerts",
		Code:    codes.Internal,
	},
	ErrTokenAlertSummaryFailed: {
		Token:   ErrTokenAlertSummaryFailed,
		Message: "failed to get alert summary",
		Code:    codes.Internal,
	},

	// Event streaming errors
	ErrTokenStreamEventsStartFailed: {
		Token:   ErrTokenStreamEventsStartFailed,
		Message: "failed to start event stream",
		Code:    codes.Internal,
	},
	ErrTokenStreamBSEventsStartFailed: {
		Token:   ErrTokenStreamBSEventsStartFailed,
		Message: "failed to start base station event stream",
		Code:    codes.Internal,
	},
	ErrTokenStreamEPEventsStartFailed: {
		Token:   ErrTokenStreamEPEventsStartFailed,
		Message: "failed to start endpoint event stream",
		Code:    codes.Internal,
	},

	// Export operation errors
	ErrTokenExportFetchFailed: {
		Token:   ErrTokenExportFetchFailed,
		Message: "failed to fetch messages for export",
		Code:    codes.Internal,
	},
	ErrTokenExportUnsupportedFormat: {
		Token:   ErrTokenExportUnsupportedFormat,
		Message: "unsupported export format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenExportEncodeFailed: {
		Token:   ErrTokenExportEncodeFailed,
		Message: "failed to encode export data",
		Code:    codes.Internal,
	},
	ErrTokenExportCSVWriteFailed: {
		Token:   ErrTokenExportCSVWriteFailed,
		Message: "failed to write CSV data",
		Code:    codes.Internal,
	},

	// Downlink operation errors
	ErrTokenDownlinkPayloadRequired: {
		Token:   ErrTokenDownlinkPayloadRequired,
		Message: "downlink payload is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenDownlinkPayloadTooLarge: {
		Token:   ErrTokenDownlinkPayloadTooLarge,
		Message: "downlink payload exceeds maximum size",
		Code:    codes.InvalidArgument,
	},
	ErrTokenDownlinkFormatInvalid: {
		Token:   ErrTokenDownlinkFormatInvalid,
		Message: "invalid downlink format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenDownlinkQueueFailed: {
		Token:   ErrTokenDownlinkQueueFailed,
		Message: "failed to queue downlink",
		Code:    codes.Internal,
	},
	ErrTokenDownlinkRevokeFailed: {
		Token:   ErrTokenDownlinkRevokeFailed,
		Message: "failed to revoke downlink",
		Code:    codes.Internal,
	},
	ErrTokenDownlinkGetResultsFailed: {
		Token:   ErrTokenDownlinkGetResultsFailed,
		Message: "failed to get downlink results",
		Code:    codes.Internal,
	},
	ErrTokenDownlinkQueueListFailed: {
		Token:   ErrTokenDownlinkQueueListFailed,
		Message: "failed to list downlink queue",
		Code:    codes.Internal,
	},
	ErrTokenEndpointNotBidirectional: {
		Token:   ErrTokenEndpointNotBidirectional,
		Message: "endpoint does not support bidirectional communication",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenBaseStationNotBidirectional: {
		Token:   ErrTokenBaseStationNotBidirectional,
		Message: "base station does not support bidirectional communication",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenDownlinkRevokeNotFound: {
		Token:   ErrTokenDownlinkRevokeNotFound,
		Message: "downlink to revoke not found",
		Code:    codes.NotFound,
	},
	ErrTokenDownlinkStatusNotFound: {
		Token:   ErrTokenDownlinkStatusNotFound,
		Message: "downlink status not found",
		Code:    codes.NotFound,
	},
	ErrTokenDownlinkAttachFailed: {
		Token:   ErrTokenDownlinkAttachFailed,
		Message: "failed to attach endpoint",
		Code:    codes.Internal,
	},
	ErrTokenDownlinkDetachFailed: {
		Token:   ErrTokenDownlinkDetachFailed,
		Message: "failed to detach endpoint",
		Code:    codes.Internal,
	},
	ErrTokenQueueIDRequired: {
		Token:   ErrTokenQueueIDRequired,
		Message: "queue_id is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidQueueIDFormat: {
		Token:   ErrTokenInvalidQueueIDFormat,
		Message: "invalid queue_id format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenNoBaseStationsConnected: {
		Token:   ErrTokenNoBaseStationsConnected,
		Message: "no base stations connected",
		Code:    codes.Unavailable,
	},
	ErrTokenQueueIDNonNegative: {
		Token:   ErrTokenQueueIDNonNegative,
		Message: "queue_id cannot be negative",
		Code:    codes.InvalidArgument,
	},
	ErrTokenDownlinkGetFailed: {
		Token:   ErrTokenDownlinkGetFailed,
		Message: "failed to retrieve downlink message",
		Code:    codes.Internal,
	},
	ErrTokenNoBaseStationOwnsQueue: {
		Token:   ErrTokenNoBaseStationOwnsQueue,
		Message: "no base station owns this queue item",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenDownlinkRevokeNoResult: {
		Token:   ErrTokenDownlinkRevokeNoResult,
		Message: "downlink revoke operation returned no result",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenResultCountOverflow: {
		Token:   ErrTokenResultCountOverflow,
		Message: "result count exceeds int32 range",
		Code:    codes.Internal,
	},

	// UL Transmit errors
	ErrTokenBaseStationOwnershipFailed: {
		Token:   ErrTokenBaseStationOwnershipFailed,
		Message: "failed to verify base station ownership",
		Code:    codes.Internal,
	},
	ErrTokenNwkSnKeyLength: {
		Token:   ErrTokenNwkSnKeyLength,
		Message: "nwkSnKey must be exactly 16 bytes",
		Code:    codes.InvalidArgument,
	},
	ErrTokenAppKeyLength: {
		Token:   ErrTokenAppKeyLength,
		Message: "appKey must be exactly 16 bytes",
		Code:    codes.InvalidArgument,
	},
	ErrTokenShortAddressZero: {
		Token:   ErrTokenShortAddressZero,
		Message: "short address cannot be zero",
		Code:    codes.InvalidArgument,
	},
	ErrTokenShortAddressOverflow: {
		Token:   ErrTokenShortAddressOverflow,
		Message: "short address exceeds uint16 range",
		Code:    codes.InvalidArgument,
	},
	ErrTokenFormatOverflow: {
		Token:   ErrTokenFormatOverflow,
		Message: "format exceeds uint8 range",
		Code:    codes.InvalidArgument,
	},
	ErrTokenULTransmitFailed: {
		Token:   ErrTokenULTransmitFailed,
		Message: "failed to send UL transmit",
		Code:    codes.Internal,
	},

	// DL RX Status errors
	ErrTokenDLRxStatusFailed: {
		Token:   ErrTokenDLRxStatusFailed,
		Message: "failed to get DL RX status",
		Code:    codes.Internal,
	},
	ErrTokenDLRxStatusQueryHistoryFailed: {
		Token:   ErrTokenDLRxStatusQueryHistoryFailed,
		Message: "failed to get DL RX status query history",
		Code:    codes.Internal,
	},
	ErrTokenLimitNonNegative: {
		Token:   ErrTokenLimitNonNegative,
		Message: "limit must be non-negative",
		Code:    codes.InvalidArgument,
	},
	ErrTokenOffsetNonNegative: {
		Token:   ErrTokenOffsetNonNegative,
		Message: "offset must be non-negative",
		Code:    codes.InvalidArgument,
	},
	ErrTokenEUIHexLengthInvalid: {
		Token:   ErrTokenEUIHexLengthInvalid,
		Message: "endpoint EUI must be exactly 16 hexadecimal characters",
		Code:    codes.InvalidArgument,
	},
	ErrTokenEUIHexCharsInvalid: {
		Token:   ErrTokenEUIHexCharsInvalid,
		Message: "endpoint EUI must contain only valid hexadecimal characters",
		Code:    codes.InvalidArgument,
	},
	ErrTokenEndpointSessionLookupFailed: {
		Token:   ErrTokenEndpointSessionLookupFailed,
		Message: "failed to lookup endpoint session",
		Code:    codes.Internal,
	},

	// System/Release errors
	ErrTokenReleaseManifestFailed: {
		Token:   ErrTokenReleaseManifestFailed,
		Message: "failed to load release manifest",
		Code:    codes.Internal,
	},

	// BSSCI Session errors
	ErrTokenBSSessionNotFound: {
		Token:   ErrTokenBSSessionNotFound,
		Message: "base station session not found",
		Code:    codes.NotFound,
	},
	ErrTokenHandshakeIncomplete: {
		Token:   ErrTokenHandshakeIncomplete,
		Message: "handshake not complete",
		Code:    codes.FailedPrecondition,
	},

	// Message listing operation errors
	ErrTokenListMessagesFailed: {
		Token:   ErrTokenListMessagesFailed,
		Message: "failed to list messages",
		Code:    codes.Internal,
	},
	ErrTokenStreamStartFailed: {
		Token:   ErrTokenStreamStartFailed,
		Message: "failed to start message stream",
		Code:    codes.Internal,
	},
	ErrTokenListBSMessagesFailed: {
		Token:   ErrTokenListBSMessagesFailed,
		Message: "failed to list base station messages",
		Code:    codes.Internal,
	},
	ErrTokenMessageStatsFailed: {
		Token:   ErrTokenMessageStatsFailed,
		Message: "failed to get message stats",
		Code:    codes.Internal,
	},
	ErrTokenSearchMessagesFailed: {
		Token:   ErrTokenSearchMessagesFailed,
		Message: "failed to search messages",
		Code:    codes.Internal,
	},
	ErrTokenExportMessagesFailed: {
		Token:   ErrTokenExportMessagesFailed,
		Message: "failed to export messages",
		Code:    codes.Internal,
	},
	ErrTokenUnsupportedGranularity: {
		Token:   ErrTokenUnsupportedGranularity,
		Message: "unsupported time series granularity",
		Code:    codes.InvalidArgument,
	},
	ErrTokenStreamBSMessagesStartFailed: {
		Token:   ErrTokenStreamBSMessagesStartFailed,
		Message: "failed to start base station message stream",
		Code:    codes.Internal,
	},
	// Events/Alerts errors
	ErrTokenEventNotFound: {
		Token:   ErrTokenEventNotFound,
		Message: "event not found",
		Code:    codes.NotFound,
	},
	ErrTokenAlertNotFound: {
		Token:   ErrTokenAlertNotFound,
		Message: "alert not found",
		Code:    codes.NotFound,
	},

	// Base Station errors
	ErrTokenBaseStationNotFound: {
		Token:   ErrTokenBaseStationNotFound,
		Message: "base station not found",
		Code:    codes.NotFound,
	},
	ErrTokenBaseStationOffline: {
		Token:   ErrTokenBaseStationOffline,
		Message: "base station is offline",
		Code:    codes.Unavailable,
	},
	ErrTokenBaseStationRequired: {
		Token:   ErrTokenBaseStationRequired,
		Message: "base station is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCreateBaseStationFailed: {
		Token:   ErrTokenCreateBaseStationFailed,
		Message: "failed to create base station",
		Code:    codes.Internal,
	},
	ErrTokenGetBaseStationFailed: {
		Token:   ErrTokenGetBaseStationFailed,
		Message: "failed to get base station",
		Code:    codes.Internal,
	},
	ErrTokenUpdateBaseStationFailed: {
		Token:   ErrTokenUpdateBaseStationFailed,
		Message: "failed to update base station",
		Code:    codes.Internal,
	},
	ErrTokenDeleteBaseStationFailed: {
		Token:   ErrTokenDeleteBaseStationFailed,
		Message: "failed to delete base station",
		Code:    codes.Internal,
	},
	ErrTokenListBaseStationsFailed: {
		Token:   ErrTokenListBaseStationsFailed,
		Message: "failed to list base stations",
		Code:    codes.Internal,
	},
	ErrTokenGetBaseStationStatsFailed: {
		Token:   ErrTokenGetBaseStationStatsFailed,
		Message: "failed to get base station stats",
		Code:    codes.Internal,
	},
	ErrTokenTenantAccessDenied: {
		Token:   ErrTokenTenantAccessDenied,
		Message: "cannot access resource for different tenant",
		Code:    codes.PermissionDenied,
	},
	ErrTokenBaseStationEUIExists: {
		Token:   ErrTokenBaseStationEUIExists,
		Message: "base station EUI already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenUpdateBaseStationEUIFailed: {
		Token:   ErrTokenUpdateBaseStationEUIFailed,
		Message: "failed to update base station EUI",
		Code:    codes.Internal,
	},
	ErrTokenNewBaseStationEUIRequired: {
		Token:   ErrTokenNewBaseStationEUIRequired,
		Message: "new base station EUI is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenLatLonPairRequired: {
		Token:   ErrTokenLatLonPairRequired,
		Message: "both latitude and longitude are required together, or both must be absent",
		Code:    codes.InvalidArgument,
	},

	// Endpoint errors
	ErrTokenEndpointNotFound: {
		Token:   ErrTokenEndpointNotFound,
		Message: "endpoint not found",
		Code:    codes.NotFound,
	},
	ErrTokenEndpointExists: {
		Token:   ErrTokenEndpointExists,
		Message: "endpoint already exists",
		Code:    codes.AlreadyExists,
	},
	ErrTokenEndpointRequired: {
		Token:   ErrTokenEndpointRequired,
		Message: "endpoint is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCreateEndpointFailed: {
		Token:   ErrTokenCreateEndpointFailed,
		Message: "failed to create endpoint",
		Code:    codes.Internal,
	},
	ErrTokenGetEndpointFailed: {
		Token:   ErrTokenGetEndpointFailed,
		Message: "failed to get endpoint",
		Code:    codes.Internal,
	},
	ErrTokenUpdateEndpointFailed: {
		Token:   ErrTokenUpdateEndpointFailed,
		Message: "failed to update endpoint",
		Code:    codes.Internal,
	},
	ErrTokenDeleteEndpointFailed: {
		Token:   ErrTokenDeleteEndpointFailed,
		Message: "failed to delete endpoint",
		Code:    codes.Internal,
	},
	ErrTokenListEndpointsFailed: {
		Token:   ErrTokenListEndpointsFailed,
		Message: "failed to list endpoints",
		Code:    codes.Internal,
	},
	ErrTokenInvalidPageToken: {
		Token:   ErrTokenInvalidPageToken,
		Message: "invalid page token",
		Code:    codes.InvalidArgument,
	},
	ErrTokenTenantContextRequired: {
		Token:   ErrTokenTenantContextRequired,
		Message: "tenant context required",
		Code:    codes.Unauthenticated,
	},
	ErrTokenAttachFailed: {
		Token:   ErrTokenAttachFailed,
		Message: "endpoint attach operation failed",
		Code:    codes.Internal,
	},
	ErrTokenDetachFailed: {
		Token:   ErrTokenDetachFailed,
		Message: "endpoint detach operation failed",
		Code:    codes.Internal,
	},
	ErrTokenUpdateEndpointEUIFailed: {
		Token:   ErrTokenUpdateEndpointEUIFailed,
		Message: "failed to update endpoint EUI",
		Code:    codes.Internal,
	},
	ErrTokenResolveTypeEUIFailed: {
		Token:   ErrTokenResolveTypeEUIFailed,
		Message: "failed to resolve TypeEUI from device model's default blueprint",
		Code:    codes.Internal,
	},

	// Required field errors
	ErrTokenNameRequired: {
		Token:   ErrTokenNameRequired,
		Message: "name is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenIDRequired: {
		Token:   ErrTokenIDRequired,
		Message: "id is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenManufacturerIDRequired: {
		Token:   ErrTokenManufacturerIDRequired,
		Message: "manufacturer_id is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenDeviceModelIDRequired: {
		Token:   ErrTokenDeviceModelIDRequired,
		Message: "device_model_id is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenVersionRequired: {
		Token:   ErrTokenVersionRequired,
		Message: "version is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenCodeRequired: {
		Token:   ErrTokenCodeRequired,
		Message: "code is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenQueryRequired: {
		Token:   ErrTokenQueryRequired,
		Message: "query is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenFormatRequired: {
		Token:   ErrTokenFormatRequired,
		Message: "format is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenMessageIDRequired: {
		Token:   ErrTokenMessageIDRequired,
		Message: "message_id is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenStateRequired: {
		Token:   ErrTokenStateRequired,
		Message: "state is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenOrgIDRequired: {
		Token:   ErrTokenOrgIDRequired,
		Message: "organization id is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenUserIDRequired: {
		Token:   ErrTokenUserIDRequired,
		Message: "user id is required",
		Code:    codes.InvalidArgument,
	},

	// Invalid format errors
	ErrTokenInvalidIDFormat: {
		Token:   ErrTokenInvalidIDFormat,
		Message: "invalid id format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidManufacturerIDFormat: {
		Token:   ErrTokenInvalidManufacturerIDFormat,
		Message: "invalid manufacturer_id format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidDeviceModelIDFormat: {
		Token:   ErrTokenInvalidDeviceModelIDFormat,
		Message: "invalid device_model_id format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidEUIFormat: {
		Token:   ErrTokenInvalidEUIFormat,
		Message: "invalid EUI format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidRoleFormat: {
		Token:   ErrTokenInvalidRoleFormat,
		Message: "invalid role format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidExportFormat: {
		Token:   ErrTokenInvalidExportFormat,
		Message: "invalid export format",
		Code:    codes.InvalidArgument,
	},

	// Generic errors
	ErrTokenInternalError: {
		Token:   ErrTokenInternalError,
		Message: "internal server error",
		Code:    codes.Internal,
	},
	ErrTokenInvalidArgument: {
		Token:   ErrTokenInvalidArgument,
		Message: "invalid argument",
		Code:    codes.InvalidArgument,
	},
	ErrTokenNotImplemented: {
		Token:   ErrTokenNotImplemented,
		Message: "feature not implemented",
		Code:    codes.Unimplemented,
	},
	ErrTokenServiceUnavailable: {
		Token:   ErrTokenServiceUnavailable,
		Message: "service temporarily unavailable",
		Code:    codes.Unavailable,
	},
	ErrTokenDatabaseError: {
		Token:   ErrTokenDatabaseError,
		Message: "database error",
		Code:    codes.Internal,
	},
	ErrTokenTenantRequired: {
		Token:   ErrTokenTenantRequired,
		Message: "tenant context is required",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenServiceNotConfigured: {
		Token:   ErrTokenServiceNotConfigured,
		Message: "service not configured",
		Code:    codes.Unimplemented,
	},
	ErrTokenOrgRequired: {
		Token:   ErrTokenOrgRequired,
		Message: "organization context is required",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenUserRequired: {
		Token:   ErrTokenUserRequired,
		Message: "user context is required",
		Code:    codes.FailedPrecondition,
	},
	ErrTokenContextMissing: {
		Token:   ErrTokenContextMissing,
		Message: "required context value is missing",
		Code:    codes.FailedPrecondition,
	},

	// Org resolver interceptor errors
	ErrTokenOrgResolverRequired: {
		Token:   ErrTokenOrgResolverRequired,
		Message: "org resolver is required for fail-closed interceptor",
		Code:    codes.Internal,
	},
	ErrTokenOrgIDHeaderRequired: {
		Token:   ErrTokenOrgIDHeaderRequired,
		Message: "x-organization-id header required",
		Code:    codes.Unauthenticated,
	},
	ErrTokenOrgIDHeaderInvalid: {
		Token:   ErrTokenOrgIDHeaderInvalid,
		Message: "invalid x-organization-id format: must be valid UUID",
		Code:    codes.InvalidArgument,
	},
	ErrTokenUserIDHeaderRequired: {
		Token:   ErrTokenUserIDHeaderRequired,
		Message: "x-user-id header required",
		Code:    codes.Unauthenticated,
	},
	ErrTokenUserIDHeaderInvalid: {
		Token:   ErrTokenUserIDHeaderInvalid,
		Message: "invalid x-user-id format: must be valid UUID",
		Code:    codes.InvalidArgument,
	},
	ErrTokenIdentityMismatch: {
		Token:   ErrTokenIdentityMismatch,
		Message: "identity mismatch: header values conflict with authenticated token",
		Code:    codes.PermissionDenied,
	},

	// EUI required field errors
	ErrTokenEndpointEUIRequired: {
		Token:   ErrTokenEndpointEUIRequired,
		Message: "ep_eui is required",
		Code:    codes.InvalidArgument,
	},
	ErrTokenBasestationEUIRequired: {
		Token:   ErrTokenBasestationEUIRequired,
		Message: "bs_eui is required",
		Code:    codes.InvalidArgument,
	},

	// EUI format validation errors
	ErrTokenInvalidEndpointEUIFormat: {
		Token:   ErrTokenInvalidEndpointEUIFormat,
		Message: "invalid ep_eui format",
		Code:    codes.InvalidArgument,
	},
	ErrTokenInvalidBasestationEUIFormat: {
		Token:   ErrTokenInvalidBasestationEUIFormat,
		Message: "invalid bs_eui format",
		Code:    codes.InvalidArgument,
	},

	// Unified activity feed errors
	ErrTokenListActivityFailed: {
		Token:   ErrTokenListActivityFailed,
		Message: "failed to list base station activity",
		Code:    codes.Internal,
	},

	// Gateway errors
	ErrTokenGatewayUpstreamUnavailable: {
		Token:   ErrTokenGatewayUpstreamUnavailable,
		Message: "gateway upstream service unavailable",
		Code:    codes.Unavailable,
	},
	ErrTokenGatewayProxyFailed: {
		Token:   ErrTokenGatewayProxyFailed,
		Message: "gateway proxy forwarding failed",
		Code:    codes.Internal,
	},
	ErrTokenGatewayInternalTrustInvalidHeader: {
		Token:   ErrTokenGatewayInternalTrustInvalidHeader,
		Message: "invalid or missing internal trust header",
		Code:    codes.Unauthenticated,
	},

	// CE/ECE edition-gated stubs (returned by RPCs whose owning service is nil in the running edition).
	ErrTokenCEStatusUnavailable: {
		Token:   ErrTokenCEStatusUnavailable,
		Message: "CE status not available in this edition",
		Code:    codes.Unimplemented,
	},
	ErrTokenCEOnboardingUnavailable: {
		Token:   ErrTokenCEOnboardingUnavailable,
		Message: "CE onboarding not available in this edition",
		Code:    codes.Unimplemented,
	},
	ErrTokenCERegistryUnavailable: {
		Token:   ErrTokenCERegistryUnavailable,
		Message: "CE registry not available in this edition",
		Code:    codes.Unimplemented,
	},
}

// GetErrorDefinition retrieves the error definition for a given token
// Returns a default definition if token not found
func GetErrorDefinition(token string) ErrorDefinition {
	if def, ok := errorCatalog[token]; ok {
		return def
	}
	// Return default for unknown tokens
	return ErrorDefinition{
		Token:   token,
		Message: "unknown error",
		Code:    codes.Internal,
	}
}

// ResolveErrorMessage returns the human-readable message for an error token
func ResolveErrorMessage(token string) string {
	return GetErrorDefinition(token).Message
}

// GetGRPCCode returns the gRPC status code for an error token
func GetGRPCCode(token string) codes.Code {
	return GetErrorDefinition(token).Code
}
