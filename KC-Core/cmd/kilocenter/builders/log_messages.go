package builders

// Builder-layer log messages for service startup.
const (
	LogGRPCServerFailed             = "gRPC server failed"
	LogFailedBSSCIMgmtServer        = "Failed to start BSSCI management server"
	LogFailedCreateCoreService      = "Failed to create CoreService"
	LogFailedKeyEncryptorInit       = "Failed to initialize key encryptor, lazy migration disabled"
	LogFailedStopBSSCIServer        = "Failed to stop BSSCI server"
	LogFailedStopSCACIServer        = "Failed to stop SCACI server"
	ErrSCACIEPStatusAdapterMissing  = "SCACI EPStatus forwarding: adapter missing SetSCACIServer"
	LogFailedStartArchivalScheduler = "Failed to start archival scheduler"
	LogFailedStopArchivalScheduler  = "Failed to stop archival scheduler"
)
