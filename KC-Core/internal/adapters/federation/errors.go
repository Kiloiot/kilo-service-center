package federation

import "errors"

// Relay transport failure sentinels. Each wraps the underlying cause via
// fmt.Errorf("%w: %w", ...) so callers can classify failures with errors.Is
// while the rendered message keeps the operation prefix.
var (
	// ErrRelayInstallationRead reports a start attempt that could not read the
	// CE installation record. (Distinct from the bootstrap service's
	// federation.ErrInstallationRead, which covers its own read path.)
	ErrRelayInstallationRead = errors.New("relay client: cannot read CE installation")
	// ErrCACertRead reports an unreadable relay CA certificate file.
	ErrCACertRead = errors.New("read CA cert")
	// ErrClientCertLoad reports a client certificate/key pair that failed to load.
	ErrClientCertLoad = errors.New("load client cert/key")
	// ErrDialCredentials reports transport credentials that could not be built.
	ErrDialCredentials = errors.New("build dial credentials")
	// ErrDialECE reports a failed dial to the ECE federation ingress.
	ErrDialECE = errors.New("dial ECE")
	// ErrStreamOpen reports a Connect stream that could not be opened.
	ErrStreamOpen = errors.New("open federation stream")
	// ErrHelloSend reports a CEHello that could not be written to the stream.
	ErrHelloSend = errors.New("send CEHello")
	// ErrHelloRecv reports a stream that ended before ECEHello arrived.
	ErrHelloRecv = errors.New("recv ECEHello")
	// ErrUnexpectedHelloPayload reports a first message that was not ECEHello.
	// It is always wrapped with the offending payload type ("%w %T"), which
	// completes the sentence.
	ErrUnexpectedHelloPayload = errors.New("expected ECEHello, got")
	// ErrAuthRejected reports an ECE that refused the CE's credentials.
	ErrAuthRejected = errors.New("ECE rejected authentication")
	// ErrHeartbeatSend reports a heartbeat that could not be written; the
	// stream is treated as gone so reconnection starts immediately.
	ErrHeartbeatSend = errors.New("send heartbeat")
	// ErrOutboxMarkSent reports a relayed record that could not be marked sent.
	ErrOutboxMarkSent = errors.New("mark outbox record sent")
)
