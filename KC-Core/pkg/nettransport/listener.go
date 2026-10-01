// Package nettransport is the TLS listener and frame codec shared by the
// BSSCI and SCACI servers.
package nettransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// ConnHandler serves one accepted connection and returns when it is done.
type ConnHandler func(conn net.Conn)

// ListenerConfig describes where and how a protocol server listens.
type ListenerConfig struct {
	// Name labels the listener in log lines.
	Name  string
	Addr  string
	Files TLSFiles
	TLS   TLSOptions
	// CertPollInterval is how often a deferred listener re-checks for the
	// certificate files.
	CertPollInterval time.Duration
	// OnListening runs once the socket is bound, immediately or after the
	// deferred certificate wait.
	OnListening func()
}

// Listener binds a mutual-TLS socket and hands every accepted connection to
// its handler. When the certificate files are missing at Start it polls for
// them so a fresh deployment can generate certificates after boot.
type Listener struct {
	cfg    ListenerConfig
	log    logger.Logger
	handle ConnHandler

	mu      sync.Mutex
	ln      net.Listener
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	stopped bool
}

// NewListener creates a listener; Start binds it.
func NewListener(cfg ListenerConfig, log logger.Logger, handle ConnHandler) *Listener {
	return &Listener{cfg: cfg, log: log, handle: handle}
}

// Start binds the socket, or starts polling for certificates when they are
// not on disk yet. ctx bounds the polling and the accept loop.
func (l *Listener) Start(ctx context.Context) error {
	if l.cfg.CertPollInterval <= 0 {
		return ErrInvalidCertPollInterval
	}
	l.mu.Lock()
	switch {
	case l.stopped:
		l.mu.Unlock()
		return ErrListenerStopped
	case l.started:
		l.mu.Unlock()
		return ErrListenerAlreadyStarted
	}
	l.started = true
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.mu.Unlock()

	if l.cfg.Files.Exist() {
		return l.listen()
	}
	l.log.WarnContext(l.ctx, LogCertsNotFound,
		logger.FieldComponent, l.cfg.Name,
		logger.FieldCert, l.cfg.Files.Cert,
		logger.FieldKey, l.cfg.Files.Key,
		logger.FieldCa, l.cfg.Files.CA)
	l.wg.Add(1)
	go l.pollCerts()
	return nil
}

// Listening reports whether the socket is bound and accepting.
func (l *Listener) Listening() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ln != nil && !l.stopped
}

// Addr returns the bound address, or nil while the listener is deferred.
func (l *Listener) Addr() net.Addr {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ln == nil {
		return nil
	}
	return l.ln.Addr()
}

// Stop closes the socket and waits for the accept loop, the certificate poll
// and every connection handler. It is idempotent and safe before Start.
func (l *Listener) Stop() {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return
	}
	l.stopped = true
	ln, cancel := l.ln, l.cancel
	l.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if ln != nil {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			l.log.ErrorContext(l.ctx, LogCloseListenerFailed, logger.FieldComponent, l.cfg.Name, logger.FieldError, err)
		}
	}
	l.wg.Wait()
}

func (l *Listener) listen() error {
	tlsConfig, err := BuildServerTLSConfig(l.cfg.Files, l.cfg.TLS)
	if err != nil {
		return fmt.Errorf(errFmtTLSConfig, err)
	}
	ln, err := tls.Listen("tcp", l.cfg.Addr, tlsConfig)
	if err != nil {
		return fmt.Errorf(errFmtListen, l.cfg.Addr, err)
	}

	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		if err := ln.Close(); err != nil {
			l.log.ErrorContext(l.ctx, LogCloseListenerFailed, logger.FieldComponent, l.cfg.Name, logger.FieldError, err)
		}
		return ErrListenerStopped
	}
	l.ln = ln
	l.mu.Unlock()

	l.log.InfoContext(l.ctx, LogListening, logger.FieldComponent, l.cfg.Name, logger.FieldAddress, ln.Addr().String())
	l.wg.Add(1)
	go l.accept(ln)
	if l.cfg.OnListening != nil {
		l.cfg.OnListening()
	}
	return nil
}

func (l *Listener) pollCerts() {
	defer l.wg.Done()
	ticker := time.NewTicker(l.cfg.CertPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.ctx.Done():
			l.log.InfoContext(l.ctx, LogDeferredListenerCancelled, logger.FieldComponent, l.cfg.Name)
			return
		case <-ticker.C:
			if !l.cfg.Files.Exist() {
				continue
			}
			l.log.InfoContext(l.ctx, LogCertsDetected, logger.FieldComponent, l.cfg.Name)
			if err := l.listen(); err != nil {
				if errors.Is(err, ErrListenerStopped) {
					return
				}
				l.log.ErrorContext(l.ctx, LogDeferredListenerFailed, logger.FieldComponent, l.cfg.Name, logger.FieldError, err)
				continue
			}
			return
		}
	}
}

func (l *Listener) accept(ln net.Listener) {
	defer l.wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if l.ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			l.log.ErrorContext(l.ctx, LogAcceptFailed, logger.FieldComponent, l.cfg.Name, logger.FieldError, err)
			continue
		}
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.handle(conn)
		}()
	}
}
