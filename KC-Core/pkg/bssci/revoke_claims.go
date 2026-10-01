package bssci

import "sync"

// revokeClaims keeps one dlDataRev per queue id on a session in the making:
// a queue id is claimed from the in-flight check until its send returned,
// after which the tracked operation keeps it in flight until the station
// answers. The zero value is ready to use.
type revokeClaims struct {
	mu      sync.Mutex
	claimed map[int64]struct{}
}

// claim reports whether queID was free and is now claimed by the caller.
func (c *revokeClaims) claim(queID int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, taken := c.claimed[queID]; taken {
		return false
	}
	if c.claimed == nil {
		c.claimed = make(map[int64]struct{})
	}
	c.claimed[queID] = struct{}{}
	return true
}

// release frees queID for the next claim.
func (c *revokeClaims) release(queID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.claimed, queID)
}
