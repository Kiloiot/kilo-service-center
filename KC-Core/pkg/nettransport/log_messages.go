package nettransport

// Log messages emitted by the listener lifecycle.
const (
	LogCertsNotFound             = "TLS certificates not found, listener deferred until certificates are generated"
	LogCertsDetected             = "TLS certificates detected, starting listener"
	LogDeferredListenerFailed    = "Failed to start deferred TLS listener"
	LogDeferredListenerCancelled = "Deferred listener polling cancelled"
	LogListening                 = "TLS listener started"
	LogAcceptFailed              = "Failed to accept connection"
	LogCloseListenerFailed       = "Failed to close listener"
)
