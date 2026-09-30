package scaci

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	// Shared MIOTY helpers (FormatEUI64, EPStatus)
	// Organization resolver for propagation context
	// BSSCI §5.8-5.8.3 attach propagation contracts
	// Import neutral scheduler contracts
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

func (s *Server) startIdleMonitor() {
	s.wg.Add(1)
	go s.monitorIdleSessions()
}

// monitorIdleSessions sends keepalive pings to idle SCACI sessions per SCACI §3.4
//
// This background loop monitors all active sessions and initiates SC-to-AC pings
// when a session has been idle for longer than PingInterval.
//
// The loop:
//  1. Collects the idle sessions from the registry
//  2. Sends pings without holding the registry lock
//  3. Uses shared PingInterval constant from KC-DB/common/config
//
// Lifecycle: Launched from Start(), stopped via s.shutdown channel
func (s *Server) monitorIdleSessions() {
	defer s.wg.Done()

	ticker := time.NewTicker(dbconfig.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			now := s.clock.Now()
			idleSessions := s.registry.connectedWhere(func(session *Session) bool {
				return session.currentState() == StateActive && session.IdleFor(now) > dbconfig.PingInterval
			})

			// Send pings outside the lock
			for _, idle := range idleSessions {
				if err := s.initiatePing(idle.conn, idle.session); err != nil {
					// SCACI §1: Use session context for tenant-aware logging
					s.logger.WarnContext(s.sessionContext(idle.session), LogSCACISendKeepaliveFailed,
						logger.FieldAcEui, mioty.FormatEUI64(idle.session.AcEui),
						logger.FieldError, err)
				}
			}
		}
	}
}
