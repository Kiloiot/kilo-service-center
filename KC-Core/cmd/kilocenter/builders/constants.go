package builders

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// Spec compliance banners logged when the protocol servers start.
const (
	specComplianceBSSCI = "MIOTY BSSCI v1.0.0"
	specComplianceSCACI = "MIOTY SCACI v1.0.0"
)

// envKilocenterMasterKey names the environment variable holding the mandatory
// key-material master key: storage construction fails without a valid key.
const envKilocenterMasterKey = keycrypto.EnvMasterKey

// Startup timing policy.
const (
	// dbConnectionWaitTimeout bounds how long startup waits for the database
	// before failing infrastructure construction.
	dbConnectionWaitTimeout = 30 * time.Second
)
