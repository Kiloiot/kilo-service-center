package auth

import (
	"bytes"
	"context"
	"net/http"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	authsvc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

// RegistrationCallbackClient posts registration notifications over HTTP.
type RegistrationCallbackClient struct {
	client *http.Client
	log    logger.Logger
}

// NewRegistrationCallbackClient creates the callback client with the
// registration callback timeout.
func NewRegistrationCallbackClient(log logger.Logger) *RegistrationCallbackClient {
	return &RegistrationCallbackClient{
		client: &http.Client{Timeout: authsvc.RegistrationCallbackTimeout},
		log:    log,
	}
}

// Post delivers payload to url and reports the response status. The body is
// drained and closed so the connection can be reused.
func (c *RegistrationCallbackClient) Post(ctx context.Context, url string, payload []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set(HeaderContentType, MediaTypeJSON)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer closeResponseBody(ctx, c.log, resp.Body)

	return resp.StatusCode, nil
}
