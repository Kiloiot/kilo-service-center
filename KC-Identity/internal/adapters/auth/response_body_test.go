package auth

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

var errBodyClose = errors.New("body close refused")

// refusingBody is a response body whose Close fails.
type refusingBody struct {
	io.Reader
	closed bool
}

func (b *refusingBody) Close() error {
	b.closed = true
	return errBodyClose
}

// warnRecorder records warning messages and discards everything else.
type warnRecorder struct {
	logger.Logger
	warnings []string
}

func (r *warnRecorder) WarnContext(_ context.Context, msg string, _ ...interface{}) {
	r.warnings = append(r.warnings, msg)
}

func TestCloseResponseBody_DrainsClosesAndLogsAFailedClose(t *testing.T) {
	body := &refusingBody{Reader: strings.NewReader("unread")}
	log := &warnRecorder{Logger: logger.NewNop()}

	closeResponseBody(testutil.TestContext(), log, body)

	remaining, err := io.ReadAll(body)
	assert.NoError(t, err)
	assert.Empty(t, remaining, "the body is drained so the connection can be reused")
	assert.True(t, body.closed)
	assert.Equal(t, []string{logResponseBodyCloseFailed}, log.warnings)
}
