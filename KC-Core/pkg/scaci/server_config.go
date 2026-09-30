package scaci

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

// Config holds SCACI server configuration per MIOTY SCACI v1.0.0
type Config struct {
	ListenAddr            string           // Address to listen on (host:port)
	TLS                   config.TLSConfig // TLS configuration (mutual TLS required per spec)
	ServiceCenterEUI      uint64           // Service Center EUI64
	Vendor                string           // SC vendor name
	Model                 string           // SC model
	Name                  string           // SC instance name
	SoftwareVersion       string           // SC software version
	OrgEnforcementEnabled bool             // Require valid X-Organization-ID header in strict mode
	LogPingOperations     bool             // Log Ping operations to scaci_operation_log (default: true)
	LogStatusOperations   bool             // Log Status operations to scaci_operation_log (default: true)
	// ConnectionEstablishmentTimeout bounds a new connection from accept to
	// the completed connect operation: TLS handshake, con and conCmp (§3.3).
	ConnectionEstablishmentTimeout time.Duration
	// SocketWriteTimeout bounds every frame write (protocol.socket_write_timeout).
	SocketWriteTimeout time.Duration
	// PlatformTenantID owns server-level events (general.tenant_id): a
	// connect refused from a certificate that resolves to no tenant.
	PlatformTenantID int64
}
