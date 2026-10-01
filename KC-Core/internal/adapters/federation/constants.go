package federation

import "time"

// outboxIdlePollInterval is how long the outbox drainer waits before polling
// again when no pending records were found.
const outboxIdlePollInterval = 100 * time.Millisecond
