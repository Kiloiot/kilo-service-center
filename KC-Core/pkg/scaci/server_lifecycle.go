package scaci

import (
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
)

// Start starts the SCACI server. If TLS certificates are not yet available
// (e.g., fresh deployment before certs are generated via UI), the listener
// is deferred and polls until the certificates appear.
func (s *Server) Start() error {
	s.listener = nettransport.NewListener(nettransport.ListenerConfig{
		Name:  listenerName,
		Addr:  s.config.ListenAddr,
		Files: nettransport.TLSFiles{Cert: s.config.TLS.CertFile, Key: s.config.TLS.KeyFile, CA: s.config.TLS.CAFile},
		TLS: nettransport.TLSOptions{
			MinVersion:   s.config.TLS.MinVersion,
			CipherSuites: s.config.TLS.AllowedCiphers,
		},
		CertPollInterval: certPollRetryInterval,
		OnListening:      s.startIdleMonitor,
	}, s.logger, s.serveConnection)
	if err := s.listener.Start(s.ctx); err != nil {
		return fmt.Errorf(errFmtStartListener, err)
	}
	return nil
}

// Listening reports whether the SCACI listener is bound and accepting.
func (s *Server) Listening() bool {
	return s.listener != nil && s.listener.Listening()
}

// serveConnection is the listener's connection handler; the server's wait
// group tracks the connection alongside its other background work.
func (s *Server) serveConnection(conn net.Conn) {
	s.wg.Add(1)
	s.handleConnection(conn)
}

// Stop gracefully shuts down the SCACI server
//
// Performs:
//  1. Signal shutdown to all goroutines
//  2. Close listener to reject new connections
//  3. Wait for all connections to terminate
//
// Returns:
//   - error: Shutdown error (currently always nil)
func (s *Server) Stop() error {
	s.logger.InfoContext(s.safeCtx(), LogSCACIServerStopping)
	// Accepts stop first, then cancelling the main context closes every
	// connection and unblocks its handler. The handlers and their background
	// work are drained before the persistence tasks, and the persistence
	// context is cancelled only after that drain (or its bound).
	close(s.shutdown)
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Stop()
	}
	s.wg.Wait()

	if !s.persistTasks.closeAndWait(ShutdownPersistDrainTimeout) {
		s.logger.WarnContext(s.safeCtx(), LogSCACIPersistDrainTimedOut)
	}
	if s.persistCancel != nil {
		s.persistCancel()
	}
	s.logger.InfoContext(s.safeCtx(), LogSCACIServerStopped)
	return nil
}
