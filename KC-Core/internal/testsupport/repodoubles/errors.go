package repodoubles

import "errors"

// ErrEndedAtConflict reports a session update that both sets and clears
// ended_at, mirroring the postgres repository's rejection of that request.
var ErrEndedAtConflict = errors.New("update request cannot set and clear ended_at at once")
