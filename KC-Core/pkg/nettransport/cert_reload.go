package nettransport

import (
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"
)

// keyPairReloader serves the server key pair on disk, reloading it when the
// certificate or the key file changes, so a renewed certificate serves the
// next handshake without a restart.
type keyPairReloader struct {
	certPath string
	keyPath  string

	mu       sync.Mutex
	pair     *tls.Certificate
	loadedAt [2]time.Time
}

// newKeyPairReloader loads the key pair once, failing when it does not load.
func newKeyPairReloader(certPath, keyPath string) (*keyPairReloader, error) {
	r := &keyPairReloader{certPath: certPath, keyPath: keyPath}
	if _, err := r.current(); err != nil {
		return nil, err
	}
	return r, nil
}

// GetCertificate serves tls.Config.GetCertificate.
func (r *keyPairReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return r.current()
}

// current reloads a changed key pair. A pair that does not load (a renewal
// still being written) keeps the last good pair serving, and the next
// handshake tries again.
func (r *keyPairReloader) current() (*tls.Certificate, error) {
	modified, err := r.modTimes()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err == nil && r.pair != nil && modified == r.loadedAt {
		return r.pair, nil
	}
	pair, loadErr := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	switch {
	case loadErr == nil && err == nil:
		r.pair, r.loadedAt = &pair, modified
		return r.pair, nil
	case r.pair != nil:
		return r.pair, nil
	case loadErr != nil:
		return nil, fmt.Errorf(errFmtLoadServerCertificate, loadErr)
	default:
		return nil, fmt.Errorf(errFmtLoadServerCertificate, err)
	}
}

func (r *keyPairReloader) modTimes() ([2]time.Time, error) {
	var times [2]time.Time
	for i, path := range []string{r.certPath, r.keyPath} {
		info, err := os.Stat(path)
		if err != nil {
			return times, err
		}
		times[i] = info.ModTime()
	}
	return times, nil
}
