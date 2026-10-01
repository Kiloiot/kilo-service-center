package grpc

// Error message and format constants for the internal gRPC delivery layer.
// Wrapping call sites use the errMsg* prefixes with "%s: %w" and the errFmt*
// formats verbatim so error text stays centralized.
const (
	errFmtFailedToListen                       = "failed to listen on %s: %w"
	errFmtEncodeBaseStationTags                = "encode base station tags: %w"
	errFmtDecodeBaseStationTags                = "decode base station tags: %w"
	errFmtTLSCertOrKeyPathMissing              = "TLS enabled but certificate or key path is missing: cert=%q, key=%q"
	errMsgBasestationSvcCannotBeNil            = "basestationSvc cannot be nil"
	errMsgDlrxStorageCannotBeNil               = "dlrxStorage cannot be nil"
	errMsgLoggerCannotBeNil                    = "logger cannot be nil"
	errMsgDownlinkCmdCannotBeNil               = "downlinkCmd cannot be nil"
	errMsgDownlinkDepsCannotBeNil              = "downlink handlers need the commands, the queue and the results"
	errMsgServingStationsCannotBeNil           = "DL RX status handlers need the serving station locator"
	errMsgEndpointSvcCannotBeNil               = "endpointSvc cannot be nil"
	errMsgEndpointClockCannotBeNil             = "endpoint handlers need a clock"
	errMsgEndpointAttachmentCannotBeNil        = "endpoint handlers need the attachment service"
	errMsgEndpointStatsWindowCannotBeNil       = "endpoint statistics need the registration window"
	errMsgFailedToCreateAuthInterceptor        = "failed to create auth interceptor"
	errMsgFailedToCreateOrgResolverInterceptor = "failed to create org resolver interceptor"
	errMsgDefaultOrgResolverRequired           = "community mode requires a default organization resolver"
	errMsgServiceStartRequired                 = "system handlers need the process start time"
	errMsgPingCmdCannotBeNil                   = "pingCmd cannot be nil"
	errMsgRoleSourceRequired                   = "role source is required"
	errMsgSessionDirCannotBeNil                = "sessionDir cannot be nil"
	errMsgStatsStoreCannotBeNil                = "statsStore cannot be nil"
	errMsgStatusReqCannotBeNil                 = "statusReq cannot be nil"
	errMsgUlTransmitCannotBeNil                = "ulTransmit cannot be nil"
)
